import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { RenderSession, TypstRenderer } from '@myriaddreamin/typst.ts/renderer'
import { createTypstRenderer } from '@myriaddreamin/typst.ts/renderer'
import { patchRoot } from '@myriaddreamin/typst.ts/render/svg/patch'
import rendererWasmUrl from '@myriaddreamin/typst-ts-renderer/wasm?url'
import {
  buildPageLineMap,
  calculateLineRatio,
  calculateScrollTarget,
  cursorLineToPageRatio,
  findHeadingLines,
  pageRatioToLine,
  scrollTopToPageRatio,
  type PageAnchor,
  type PageLayoutInfo,
} from '../lib/scrollSync'
import { WasmCompilerClient } from '../lib/wasm/client'
import { formatDiagnostics, isError } from '../lib/wasm/diagnostics'
import { findSourceLine } from '../lib/wasm/sourceMap'

export interface WasmPreviewStatus {
  status: 'loading' | 'ready' | 'error'
  pageCount: number
  message: string | null
  /** Duration of the last compile, in milliseconds. */
  ms?: number
}

interface WasmPreviewProps {
  path: string
  /** Null until the file's contents are loaded: nothing is compiled, so an
   * empty buffer never shadows the file on disk. */
  source: string | null
  /** Bumped after every save: other project files may have changed. */
  version: number
  /** Bumped by the toolbar's reload button: forces a full rebuild. */
  nonce: number
  cursor?: { line: number; col: number; totalLines: number } | null
  syncScroll: boolean
  zoom: number
  fitWidth: boolean
  onStatus: (s: WasmPreviewStatus) => void
  /** The compiler cannot run here; the caller switches to server preview. */
  onUnavailable: (reason: string) => void
  onScrollToLine?: (line: number) => void
  onJumpToLine?: (line: number) => void
}

let rendererPromise: Promise<TypstRenderer> | null = null
function getRenderer(): Promise<TypstRenderer> {
  rendererPromise ??= (async () => {
    const renderer = createTypstRenderer()
    await renderer.init({ getModule: () => rendererWasmUrl })
    return renderer
  })()
  rendererPromise.catch(() => {
    rendererPromise = null
  })
  return rendererPromise
}

// The renderer session holds the document and remembers what it last
// drew, so renderSvgDiff can refer back to SVG nodes already on screen.
// runWithSession frees the session when its callback settles; the
// callback stays pending until the session is replaced.
let current: { session: RenderSession; release: () => void } | null = null

async function getSession(fresh: boolean): Promise<RenderSession> {
  if (current && !fresh) return current.session
  const renderer = await getRenderer()
  current?.release()
  return new Promise<RenderSession>((resolve) => {
    void renderer.runWithSession(
      (session) =>
        new Promise<void>((release) => {
          current = { session, release }
          resolve(session)
        }),
    )
  })
}

/** Draws the session's document into host: the whole SVG the first time,
 * then only what changed (typst-preview's renderSvgDiff + patchRoot). */
function paint(session: RenderSession, host: HTMLElement, fresh: boolean) {
  const svg = session.renderSvgDiff({})
  const prev = host.firstElementChild
  if (fresh || !(prev instanceof SVGElement)) {
    host.innerHTML = svg
    return
  }
  const tmp = document.createElement('div')
  tmp.innerHTML = svg
  const next = tmp.firstElementChild
  if (next instanceof SVGElement) patchRoot(prev, next)
}

const SAFE_LINK = /^(https?:|mailto:|#)/i

/** Typst documents can contain links to any URL, including `javascript:`,
 * and the preview is inserted into the app's own page: keep only web,
 * mail and in-document links, and open web links in a new tab so a click
 * never navigates away from unsaved work. */
function sanitizeLinks(host: HTMLElement) {
  for (const a of host.querySelectorAll('a')) {
    for (const attr of ['href', 'xlink:href']) {
      const v = a.getAttribute(attr)
      if (v !== null && !SAFE_LINK.test(v.trim())) a.removeAttribute(attr)
    }
    const target = a.getAttribute('href') ?? a.getAttribute('xlink:href') ?? ''
    if (/^https?:/i.test(target)) {
      a.setAttribute('target', '_blank')
      a.setAttribute('rel', 'noopener noreferrer')
    }
  }
}

/** Text of the run at a click point. Glyphs only hit-test on their inked
 * pixels (a click between strokes lands on the box behind the text), so
 * runs are matched by bounding box: the one containing the point, else the
 * nearest on the same line. Each `.typst-text` group carries its text in a
 * `.tsel` element for selection. */
function textAt(host: HTMLElement, x: number, y: number): string | null {
  const SAME_LINE = 6
  let best: { dist: number; text: string } | null = null
  for (const run of host.querySelectorAll('.typst-text')) {
    const text = run.querySelector('.tsel')?.textContent
    if (!text) continue
    const r = run.getBoundingClientRect()
    const dy = y < r.top ? r.top - y : y > r.bottom ? y - r.bottom : 0
    if (dy > SAME_LINE) continue
    const dx = x < r.left ? r.left - x : x > r.right ? x - r.right : 0
    const dist = dy * 4 + dx
    if (!best || dist < best.dist) best = { dist, text }
    if (dist === 0) break
  }
  return best?.text ?? null
}

/**
 * WasmPreview renders the document compiled in the browser (lib/wasm). Each
 * compile sends only what changed since the last one, and the renderer
 * patches the existing SVG instead of replacing whole pages.
 */
export function WasmPreview({
  path,
  source,
  version,
  nonce,
  cursor,
  syncScroll,
  zoom,
  fitWidth,
  onStatus,
  onUnavailable,
  onScrollToLine,
  onJumpToLine,
}: WasmPreviewProps) {
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const docRef = useRef<HTMLDivElement | null>(null)
  const [pageCount, setPageCount] = useState(0)
  const [docWidthPt, setDocWidthPt] = useState(0)
  // The source the rendered document was compiled from (it lags the
  // editor while a compile is running).
  const [compiled, setCompiled] = useState('')
  // Anchors belong to one document; a stale set for another file or an
  // earlier rebuild is ignored rather than cleared.
  const docKey = `${path}#${nonce}`
  const [anchorState, setAnchorState] = useState<{ key: string; list: PageAnchor[] | null } | null>(null)
  const anchors = anchorState?.key === docKey ? anchorState.list : null

  const latestRef = useRef({ path, source })
  const inFlightRef = useRef(false)
  const dirtyRef = useRef(false)
  const resetRef = useRef(true)
  const lastAnchorKeyRef = useRef('')
  const programmaticRef = useRef(false)
  const userScrollRef = useRef(false)
  const timers = useRef<Record<string, number | undefined>>({})
  const statusRef = useRef(onStatus)
  const unavailableRef = useRef(onUnavailable)
  // Declared before the effects below, so they always see current values.
  useEffect(() => {
    latestRef.current = { path, source }
    statusRef.current = onStatus
    unavailableRef.current = onUnavailable
  })

  const setTimer = (name: string, ms: number, fn: () => void) => {
    window.clearTimeout(timers.current[name])
    timers.current[name] = window.setTimeout(fn, ms)
  }

  useEffect(() => {
    const t = timers.current
    return () => Object.values(t).forEach((id) => window.clearTimeout(id))
  }, [])

  const pageLayouts = useCallback((): PageLayoutInfo[] => {
    const scroller = scrollRef.current
    const doc = docRef.current
    if (!scroller || !doc) return []
    const base = scroller.getBoundingClientRect().top - scroller.scrollTop
    // patchRoot reuses page nodes in place, so DOM order is not page order;
    // each page is positioned by its own transform.
    return [...doc.querySelectorAll('.typst-page')]
      .map((el) => {
        const r = el.getBoundingClientRect()
        return { top: r.top - base, height: r.height }
      })
      .sort((a, b) => a.top - b.top)
  }, [])

  const run = useCallback(async () => {
    if (inFlightRef.current) {
      dirtyRef.current = true
      return
    }
    inFlightRef.current = true
    try {
      do {
        dirtyRef.current = false
        const { path: p, source: src } = latestRef.current
        if (src === null) break
        const client = WasmCompilerClient.get()
        // Mounting always asks for a reset (resetRef starts true), so a new,
        // empty host never receives a diff that refers to nodes it lacks.
        // The flag is only cleared once a whole document has arrived.
        const outcome = await client.compile(p, src, resetRef.current)
        if (outcome.action === 'reset') resetRef.current = false
        const errors = outcome.diagnostics.filter(isError)
        if (outcome.action !== 'none' && outcome.delta && docRef.current) {
          const fresh = outcome.action === 'reset'
          const session = await getSession(fresh)
          session.manipulateData({ action: outcome.action, data: outcome.delta })
          paint(session, docRef.current, fresh)
          sanitizeLinks(docRef.current)
          setCompiled(src)
          const pages = docRef.current.querySelectorAll('.typst-page').length
          setPageCount(pages)
          setDocWidthPt(session.docWidth)
          statusRef.current({
            status: 'ready',
            pageCount: pages,
            message: outcome.diagnostics.length ? formatDiagnostics(outcome.diagnostics) : null,
            ms: outcome.ms,
          })
        } else {
          statusRef.current({
            status: errors.length ? 'error' : 'ready',
            pageCount: docRef.current?.querySelectorAll('.typst-page').length ?? 0,
            message: formatDiagnostics(outcome.diagnostics) || 'Biên dịch không thành công',
            ms: outcome.ms,
          })
        }
      } while (dirtyRef.current)
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      statusRef.current({ status: 'error', pageCount: 0, message })
    } finally {
      inFlightRef.current = false
    }
  }, [])

  // Start the worker and load the project's file list; reload it after
  // every save so imports of other files see their new contents.
  useEffect(() => {
    let cancelled = false
    statusRef.current({ status: 'loading', pageCount: 0, message: null })
    const client = WasmCompilerClient.get()
    client.ready
      .then(() => client.refreshManifest())
      .then(() => {
        if (!cancelled) void run()
      })
      .catch((err: unknown) => {
        if (!cancelled) unavailableRef.current(err instanceof Error ? err.message : String(err))
      })
    return () => {
      cancelled = true
    }
  }, [version, run])

  useEffect(() => {
    resetRef.current = true
    lastAnchorKeyRef.current = ''
  }, [path, nonce])

  useEffect(() => {
    void run()
  }, [path, source, nonce, run])

  // Heading -> page anchors for scroll sync, refreshed only when the
  // heading structure changes (the query recompiles the document).
  useEffect(() => {
    if (!pageCount || !compiled) return
    const headingLines = findHeadingLines(compiled)
    const key = headingLines.join(',')
    if (key === lastAnchorKeyRef.current) return
    setTimer('anchors', 1200, () => {
      lastAnchorKeyRef.current = key
      if (!headingLines.length) return setAnchorState({ key: docKey, list: null })
      WasmCompilerClient.get()
        .headingPages(path)
        .then((pages) =>
          setAnchorState({
            key: docKey,
            list:
              pages && pages.length === headingLines.length
                ? headingLines.map((line, i) => ({ line, page: pages[i] }))
                : null,
          }),
        )
        .catch(() => setAnchorState({ key: docKey, list: null }))
    })
  }, [pageCount, path, compiled, docKey])

  const pageLineMap = useMemo(
    () => buildPageLineMap(compiled, pageCount, anchors ?? undefined),
    [compiled, pageCount, anchors],
  )

  const lineRatio = useCallback(
    (line: number, total: number) =>
      pageLineMap.length >= 2 ? cursorLineToPageRatio(line, pageLineMap, total) : calculateLineRatio(line, total),
    [pageLineMap],
  )

  // Editor -> preview: follow the caret once it rests.
  useEffect(() => {
    if (!syncScroll || !cursor || !pageCount) return
    setTimer('cursor', 600, () => {
      const scroller = scrollRef.current
      if (!scroller || userScrollRef.current) return
      const target = calculateScrollTarget(
        lineRatio(cursor.line, cursor.totalLines),
        pageLayouts(),
        scroller.clientHeight,
        scroller.scrollHeight,
      )
      if (Math.abs(scroller.scrollTop - target) > 80) {
        programmaticRef.current = true
        setTimer('programmatic', 700, () => {
          programmaticRef.current = false
        })
        scroller.scrollTo({ top: target, behavior: 'smooth' })
      }
    })
  }, [cursor, syncScroll, pageCount, lineRatio, pageLayouts])

  // Preview -> editor: scroll the editor to what the reader is looking at.
  const handleScroll = () => {
    userScrollRef.current = true
    setTimer('user', 1200, () => {
      userScrollRef.current = false
    })
    if (!syncScroll || !onScrollToLine || !pageCount) return
    setTimer('reverse', 300, () => {
      const scroller = scrollRef.current
      if (!scroller || programmaticRef.current) return
      const layouts = pageLayouts()
      if (!layouts.length) return
      const ratio = scrollTopToPageRatio(scroller.scrollTop, layouts, scroller.clientHeight)
      const total = cursor?.totalLines ?? compiled.split('\n').length
      onScrollToLine(pageRatioToLine(ratio, pageLineMap, total))
    })
  }

  const handleClick = (e: React.MouseEvent) => {
    if (!onJumpToLine || !docRef.current) return
    const text = textAt(docRef.current, e.clientX, e.clientY)
    if (!text) return
    const scroller = scrollRef.current
    const layouts = pageLayouts()
    const src = compiled
    const total = src.split('\n').length
    let estimate = 1
    if (scroller && layouts.length) {
      const y = e.clientY - scroller.getBoundingClientRect().top + scroller.scrollTop
      const last = layouts[layouts.length - 1]
      const ratio = Math.max(0, Math.min(1, (y - layouts[0].top) / (last.top + last.height - layouts[0].top)))
      estimate = pageRatioToLine(ratio, pageLineMap, total)
    }
    const line = findSourceLine(src, text, estimate)
    if (line) onJumpToLine(line)
  }

  // Typst sizes are in points; 1pt = 4/3 CSS px at 100%.
  const width = fitWidth || !docWidthPt ? '100%' : `${Math.round(docWidthPt * (4 / 3) * (zoom / 100))}px`

  return (
    <div className="preview-scroll-container wasm-preview-scroll" ref={scrollRef} onScroll={handleScroll}>
      <div
        className="wasm-preview-doc"
        ref={docRef}
        style={{ width }}
        onClick={handleClick}
        title={onJumpToLine ? 'Bấm vào chữ để nhảy tới dòng nguồn' : undefined}
      />
    </div>
  )
}
