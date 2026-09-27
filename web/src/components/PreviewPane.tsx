import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { buildPageLineMap, calculateLineRatio, calculateScrollTarget, cursorLineToPageRatio, type PageLayoutInfo } from '../lib/scrollSync'
import { extractSvgDimensions, scopeSvgIds } from '../lib/svgHelper'
import { Icon } from './Icon'

type Status = 'idle' | 'loading' | 'ready' | 'error'
type ViewMode = 'svg' | 'pdf' | 'tinymist'

interface PreviewPaneProps {
  path: string | null
  version: number
  liveContent?: string | null
  cursor?: { line: number; col: number; totalLines: number } | null
}

interface PageCardProps {
  pageIndex: number
  pageCount: number
  rawSvg: string
  isActive: boolean
  containerEl: HTMLDivElement | null
  onRegisterRef: (el: HTMLDivElement | null) => void
}

function PreviewPageCard({
  pageIndex,
  pageCount,
  rawSvg,
  isActive,
  containerEl,
  onRegisterRef,
}: PageCardProps) {
  const [isVisible, setIsVisible] = useState<boolean>(pageIndex < 3) // Immediately render first 3 pages
  const cardRef = useRef<HTMLDivElement | null>(null)

  const dimensions = useMemo(() => extractSvgDimensions(rawSvg), [rawSvg])
  const scopedSvg = useMemo(() => {
    if (!isVisible) return ''
    return scopeSvgIds(rawSvg, pageIndex)
  }, [rawSvg, pageIndex, isVisible])

  useEffect(() => {
    const el = cardRef.current
    if (!el) return

    // Use IntersectionObserver to lazy-render SVGs only when near viewport
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setIsVisible(true)
          } else {
            // Keep rendered once loaded for fast smooth scrolling
            // or unload if very far (keep active/near)
          }
        }
      },
      {
        root: containerEl,
        rootMargin: '1000px 0px', // Pre-render 1000px before scrolling into view
        threshold: 0.01,
      },
    )

    observer.observe(el)
    return () => observer.disconnect()
  }, [containerEl])

  return (
    <div
      ref={(el) => {
        cardRef.current = el
        onRegisterRef(el)
      }}
      className={`preview-page-card${isActive ? ' active-page' : ''}`}
      data-page={pageIndex + 1}
    >
      <div className="preview-page-header">
        <span>Trang {pageIndex + 1} / {pageCount}</span>
      </div>
      <div className="preview-svg-content">
        {isVisible && scopedSvg ? (
          <div
            className="preview-svg-inner"
            dangerouslySetInnerHTML={{ __html: scopedSvg }}
          />
        ) : (
          <div
            className="preview-page-placeholder"
            style={{
              aspectRatio: `${dimensions.width} / ${dimensions.height}`,
              minHeight: '320px',
            }}
          >
            <span className="spinner spinner-sm" />
            <span>Đang tải trang {pageIndex + 1}…</span>
          </div>
        )}
      </div>
    </div>
  )
}

export function PreviewPane({ path, version, liveContent, cursor }: PreviewPaneProps) {
  const [status, setStatus] = useState<Status>('idle')
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [pages, setPages] = useState<string[]>([])
  const [pageCount, setPageCount] = useState<number>(0)
  const [viewMode, setViewMode] = useState<ViewMode>('svg')
  const [syncScroll, setSyncScroll] = useState<boolean>(true)
  const [zoom, setZoom] = useState<number>(100)
  const [fitWidth, setFitWidth] = useState<boolean>(false)
  const [pdfUrl, setPdfUrl] = useState<string | null>(null)
  const [nonce, setNonce] = useState(0)
  const [tinymistReady, setTinymistReady] = useState(false)

  const scrollContainerRef = useRef<HTMLDivElement | null>(null)
  const pageRefs = useRef<(HTMLDivElement | null)[]>([])
  const manualScrollTimerRef = useRef<number | undefined>(undefined)
  const isUserScrollingRef = useRef<boolean>(false)
  const lastRenderedContentRef = useRef<string | null>(null)
  // Debounce cursor scroll: only scroll after cursor is idle for 600ms
  const cursorScrollTimerRef = useRef<number | undefined>(undefined)

  const isTyp = !!path && path.endsWith('.typ')

  // Build accurate page→line map from source content and page count. Uses the
  // content that was actually compiled to produce pageCount (lastRenderedContentRef),
  // not the live editor content — liveContent can already be ahead (user kept typing
  // while the compile request was in flight), which would misalign line offsets.
  const pageLineMap = useMemo(() => {
    return buildPageLineMap(lastRenderedContentRef.current ?? liveContent ?? '', pageCount)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pageCount])

  // Calculate line ratio using the accurate page line map (falls back to linear)
  const getLineRatio = useCallback(
    (line: number, totalLines: number): number => {
      if (pageLineMap.length >= 2) {
        return cursorLineToPageRatio(line, pageLineMap, totalLines)
      }
      return calculateLineRatio(line, totalLines)
    },
    [pageLineMap],
  )

  // Calculate active page based on cursor
  const activePageIndex = useMemo(() => {
    if (!cursor || !pageCount) return 0
    const ratio = getLineRatio(cursor.line, cursor.totalLines)
    return Math.min(pageCount - 1, Math.floor(ratio * pageCount))
  }, [cursor, pageCount, getLineRatio])


  // Fetch / render live SVG or PDF. The tinymist-native mode needs neither —
  // it's driven entirely by the /preview/ iframe + /api/preview/restart (see
  // Workspace.tsx), so status just tracks readiness there instead.
  useEffect(() => {
    if (!path || !path.endsWith('.typ') || viewMode === 'tinymist') return
    const ctrl = new AbortController()

    void (async () => {
      setStatus('loading')
      try {
        if (viewMode === 'svg') {
          const res = await fetch('/api/preview/render', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            signal: ctrl.signal,
            body: JSON.stringify({
              path,
              content: liveContent ?? undefined,
              format: 'svg',
            }),
          })

          if (!res.ok) {
            let msg = res.statusText
            try {
              const body = await res.json()
              if (body?.error) msg = body.error
            } catch {
              // ignore
            }
            if (!ctrl.signal.aborted) {
              setErrorMessage(msg)
              setStatus('error')
            }
            return
          }

          const data = await res.json()
          if (!ctrl.signal.aborted) {
            if (data.ok && Array.isArray(data.pages)) {
              setPages(data.pages)
              setPageCount(data.pageCount || data.pages.length)
              setErrorMessage(null)
              setStatus('ready')
              lastRenderedContentRef.current = liveContent ?? ''
            } else {
              setErrorMessage(data.error || 'Biên dịch không thành công')
              setStatus('error')
            }
          }
        } else {
          // Native PDF mode
          const res = await fetch(`/api/preview/pdf?path=${encodeURIComponent(path)}`, {
            credentials: 'include',
            signal: ctrl.signal,
          })
          if (!res.ok) {
            let msg = res.statusText
            try {
              const body = await res.json()
              if (body?.error) msg = body.error
            } catch {
              // non-JSON
            }
            if (!ctrl.signal.aborted) {
              setErrorMessage(msg)
              setStatus('error')
            }
            return
          }
          const blobUrl = URL.createObjectURL(await res.blob())
          if (!ctrl.signal.aborted) {
            if (pdfUrl) URL.revokeObjectURL(pdfUrl)
            setPdfUrl(blobUrl)
            setErrorMessage(null)
            setStatus('ready')
          }
        }
      } catch (err) {
        if (!ctrl.signal.aborted) {
          setErrorMessage(err instanceof Error ? err.message : String(err))
          setStatus('error')
        }
      }
    })()

    return () => ctrl.abort()
  }, [path, version, liveContent, nonce, viewMode])

  // Sync scroll to cursor position — debounced 600ms so it only fires when cursor is idle
  useEffect(() => {
    if (!syncScroll || !cursor || viewMode !== 'svg' || pages.length === 0) return

    // Clear any pending debounce
    window.clearTimeout(cursorScrollTimerRef.current)

    cursorScrollTimerRef.current = window.setTimeout(() => {
      // After debounce: check if user just manually scrolled
      if (isUserScrollingRef.current) return
      const container = scrollContainerRef.current
      if (!container) return

      const lineRatio = getLineRatio(cursor.line, cursor.totalLines)

      // Gather rendered page bounds
      const pageLayouts: PageLayoutInfo[] = []
      for (let i = 0; i < pages.length; i++) {
        const el = pageRefs.current[i]
        if (el) {
          pageLayouts.push({ top: el.offsetTop, height: el.offsetHeight })
        }
      }

      const targetY = calculateScrollTarget(
        lineRatio,
        pageLayouts,
        container.clientHeight,
        container.scrollHeight,
      )

      // Only scroll if we're more than half a viewport away from the target
      const currentY = container.scrollTop
      if (Math.abs(currentY - targetY) > container.clientHeight * 0.5) {
        container.scrollTo({ top: targetY, behavior: 'smooth' })
      } else if (Math.abs(currentY - targetY) > 80) {
        container.scrollTo({ top: targetY, behavior: 'smooth' })
      }
    }, 600)

    return () => window.clearTimeout(cursorScrollTimerRef.current)
  }, [cursor, syncScroll, pages.length, viewMode, getLineRatio])

  // Tinymist-native view mode: poll readiness before mounting the iframe (the
  // preview server is (re)started by Workspace's /api/preview/restart call on
  // file open/save; starting it takes a moment, so avoid a proxy 503 flash).
  useEffect(() => {
    if (viewMode !== 'tinymist') {
      setTinymistReady(false)
      return
    }
    let cancelled = false
    let timer: number | undefined
    const poll = async () => {
      try {
        const res = await fetch('/api/preview/status', { credentials: 'include' })
        const data = await res.json()
        if (cancelled) return
        if (data.ready) {
          setTinymistReady(true)
          return
        }
      } catch {
        // ignore, retry below
      }
      if (!cancelled) timer = window.setTimeout(poll, 500)
    }
    void poll()
    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [viewMode, path])

  // Tinymist-native view mode: forward cursor changes so tinymist's own preview
  // server can scroll to the exact rendered position — the same call the
  // desktop editor makes on every selection change.
  useEffect(() => {
    if (viewMode !== 'tinymist' || !syncScroll || !cursor) return
    const timer = window.setTimeout(() => {
      void fetch('/api/preview/cursor', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ line: cursor.line, character: cursor.col }),
      }).catch(() => {
        // best-effort: a missed cursor update just skips one scroll sync tick
      })
    }, 500)
    return () => window.clearTimeout(timer)
  }, [cursor, syncScroll, viewMode])

  // Track manual scroll by user to avoid jumping while reading
  const handleScroll = () => {
    isUserScrollingRef.current = true
    window.clearTimeout(manualScrollTimerRef.current)
    manualScrollTimerRef.current = window.setTimeout(() => {
      isUserScrollingRef.current = false
    }, 1200)
  }

  // Cleanup blob URL on unmount or file switch
  useEffect(() => {
    return () => {
      if (pdfUrl) URL.revokeObjectURL(pdfUrl)
      window.clearTimeout(manualScrollTimerRef.current)
      window.clearTimeout(cursorScrollTimerRef.current)
    }
  }, [path, pdfUrl])

  if (!isTyp) {
    return (
      <div className="preview-pane">
        <div className="preview-loading">
          <Icon name="eye" size={28} />
          <span>Chọn một file .typ để xem trước</span>
        </div>
      </div>
    )
  }

  const handleZoomIn = () => {
    setFitWidth(false)
    setZoom((z) => Math.min(200, z + 15))
  }
  const handleZoomOut = () => {
    setFitWidth(false)
    setZoom((z) => Math.max(50, z - 15))
  }
  const handleZoomReset = () => {
    setFitWidth(false)
    setZoom(100)
  }
  const handleToggleFit = () => {
    setFitWidth((f) => !f)
  }

  return (
    <div className="preview-pane">
      <div className="preview-toolbar">
        {/* Status indicator */}
        {viewMode === 'tinymist' ? (
          <span className={`preview-status preview-status-${tinymistReady ? 'ready' : 'loading'}`}>
            {tinymistReady ? (
              <>
                <Icon name="check" size={13} /> Đồng bộ chính xác (tinymist)
              </>
            ) : (
              <>
                <span className="spinner" /> Đang khởi động preview…
              </>
            )}
          </span>
        ) : (
          <span className={`preview-status preview-status-${status}`}>
            {status === 'loading' && (
              <>
                <span className="spinner" /> Đang cập nhật…
              </>
            )}
            {status === 'ready' && (
              <>
                <Icon name="check" size={13} /> {viewMode === 'svg' ? `Xem trực tiếp (${pageCount} trang)` : 'Bản xem trước PDF'}
              </>
            )}
            {status === 'error' && (
              <>
                <Icon name="error" size={13} /> Lỗi biên dịch
              </>
            )}
          </span>
        )}

        <span className="preview-toolbar-spacer" />

        {/* Scroll Sync Toggle */}
        {(viewMode === 'svg' || viewMode === 'tinymist') && (
          <button
            className={`btn-ghost preview-tool preview-tool-btn${syncScroll ? ' active' : ''}`}
            title={syncScroll ? 'Đang bật đồng bộ cuộn theo con trỏ (Bấm để tắt)' : 'Đang tắt đồng bộ cuộn (Bấm để bật)'}
            aria-label="Đồng bộ cuộn"
            aria-pressed={syncScroll}
            onClick={() => setSyncScroll((s) => !s)}
          >
            <Icon name={syncScroll ? 'link' : 'unlink'} size={14} />
            <span className="preview-tool-label">Đồng bộ cuộn</span>
          </button>
        )}

        {/* Zoom controls for SVG mode */}
        {viewMode === 'svg' && (
          <div className="preview-zoom-group">
            <button
              className="btn-ghost preview-tool"
              title="Thu nhỏ (-)"
              aria-label="Thu nhỏ"
              onClick={handleZoomOut}
            >
              <Icon name="zoom-out" size={14} />
            </button>
            <button
              className="btn-ghost preview-tool preview-zoom-value"
              title="Cỡ chuẩn 100%"
              aria-label="Cỡ chuẩn 100%"
              onClick={handleZoomReset}
            >
              {fitWidth ? 'Vừa khung' : `${zoom}%`}
            </button>
            <button
              className="btn-ghost preview-tool"
              title="Phóng to (+)"
              aria-label="Phóng to"
              onClick={handleZoomIn}
            >
              <Icon name="zoom-in" size={14} />
            </button>
            <button
              className={`btn-ghost preview-tool${fitWidth ? ' active' : ''}`}
              title="Vừa chiều rộng khung nhìn"
              aria-label="Vừa chiều rộng"
              onClick={handleToggleFit}
            >
              <Icon name="maximize-2" size={14} />
            </button>
          </div>
        )}

        {/* View mode toggle (SVG Paper vs PDF Embed) */}
        <div className="preview-mode-group">
          <button
            className={`btn-ghost preview-tool preview-mode-btn${viewMode === 'svg' ? ' active' : ''}`}
            title="Chế độ trang SVG (Cập nhật tức thì, mượt mà)"
            onClick={() => setViewMode('svg')}
          >
            Trang in
          </button>
          <button
            className={`btn-ghost preview-tool preview-mode-btn${viewMode === 'pdf' ? ' active' : ''}`}
            title="Chế độ xem nhúng PDF gốc"
            onClick={() => setViewMode('pdf')}
          >
            PDF
          </button>
          <button
            className={`btn-ghost preview-tool preview-mode-btn${viewMode === 'tinymist' ? ' active' : ''}`}
            title="Đồng bộ cuộn qua tinymist (giao diện riêng, không tùy biến được). Đã kiểm chứng: chỉ tự cuộn khi bạn GÕ PHÍM tại vị trí mới — click chuột hoặc dùng phím mũi tên để di chuyển con trỏ mà không gõ gì thì KHÔNG cuộn theo (giới hạn của tinymist, không phải lỗi)"
            onClick={() => setViewMode('tinymist')}
          >
            Đồng bộ chính xác
          </button>
        </div>

        {/* Reload */}
        {viewMode !== 'tinymist' && (
          <button
            className="btn-ghost preview-tool"
            title="Biên dịch lại ngay"
            aria-label="Biên dịch lại"
            onClick={() => setNonce((n) => n + 1)}
            disabled={status === 'loading'}
          >
            <Icon name="refresh" size={14} />
          </button>
        )}

        {/* Open PDF in new tab */}
        {viewMode !== 'tinymist' && (
          <button
            className="btn-ghost preview-tool"
            title="Mở PDF trong tab mới để in ấn"
            aria-label="Mở PDF trong tab mới"
            onClick={() => window.open(`/api/preview/pdf?path=${encodeURIComponent(path)}`, '_blank')}
          >
            <Icon name="external" size={14} />
          </button>
        )}
      </div>

      <div className="preview-body">
        {/* SVG Paper View Mode */}
        {viewMode === 'svg' && (
          <div
            className="preview-scroll-container"
            ref={scrollContainerRef}
            onScroll={handleScroll}
          >
            {pages.length > 0 ? (
              <div
                className={`preview-pages-wrapper${fitWidth ? ' fit-width' : ''}`}
                style={!fitWidth && zoom !== 100 ? { transform: `scale(${zoom / 100})`, transformOrigin: 'top center' } : undefined}
              >
                {pages.map((svgContent, idx) => (
                  <PreviewPageCard
                    key={idx}
                    pageIndex={idx}
                    pageCount={pageCount}
                    rawSvg={svgContent}
                    isActive={idx === activePageIndex}
                    containerEl={scrollContainerRef.current}
                    onRegisterRef={(el) => {
                      pageRefs.current[idx] = el
                    }}
                  />
                ))}
              </div>
            ) : status === 'loading' ? (
              <div className="preview-loading">
                <span className="spinner spinner-lg" />
                <span>Đang kết xuất tài liệu xem trước…</span>
              </div>
            ) : null}
          </div>
        )}

        {/* PDF Embed Mode */}
        {viewMode === 'pdf' && (
          <div className="preview-pdf-wrapper">
            {pdfUrl && <embed className="preview-pdf" src={pdfUrl} type="application/pdf" />}
            {!pdfUrl && status === 'loading' && (
              <div className="preview-loading">
                <span className="spinner spinner-lg" />
                <span>Đang biên dịch PDF…</span>
              </div>
            )}
          </div>
        )}

        {/* Tinymist-native preview mode (exact scroll sync, tinymist's own UI) */}
        {viewMode === 'tinymist' && (
          <div className="preview-tinymist-wrapper-outer">
            {tinymistReady && (
              <div className="preview-tinymist-hint" role="note">
                Chỉ tự cuộn khi bạn <strong>gõ phím</strong> tại vị trí mới — click chuột/di chuyển con trỏ mà không gõ gì sẽ không cuộn theo (giới hạn của tinymist).
              </div>
            )}
            <div className="preview-tinymist-wrapper">
              {tinymistReady ? (
                <iframe
                  key={path ?? undefined}
                  src="/preview/"
                  className="preview-tinymist-frame"
                  title="Xem trước đồng bộ chính xác (tinymist)"
                />
              ) : (
                <div className="preview-loading">
                  <span className="spinner spinner-lg" />
                  <span>Đang khởi động preview đồng bộ…</span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Error notification banner / popup (not applicable to tinymist mode,
            which surfaces its own compile errors inside its iframe) */}
        {errorMessage && viewMode !== 'tinymist' && (
          <div className="preview-error-banner" role="alert">
            <div className="preview-error-header">
              <Icon name="error" size={15} />
              <span>Lỗi cú pháp Typst</span>
              <button
                className="btn-ghost preview-error-close"
                onClick={() => setErrorMessage(null)}
                title="Đóng thông báo"
              >
                <Icon name="x" size={13} />
              </button>
            </div>
            <pre className="preview-error-message">{errorMessage}</pre>
          </div>
        )}
      </div>
    </div>
  )
}
