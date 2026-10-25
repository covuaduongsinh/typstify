/// <reference lib="webworker" />
// The Typst compiler (typst.ts, ~30 MB of WASM) runs here so compiling
// never blocks typing. It keeps one IncrementalServer for the whole session
// and posts back only the vector delta since the previous compile; the
// renderer on the main thread (WasmPreview.tsx) patches the SVG with it.
//
// File access is synchronous inside the compiler, so project files and
// packages are fetched with blocking XHRs, which are allowed in workers.
import { createTypstCompiler, loadFonts } from '@myriaddreamin/typst.ts'
import { withAccessModel, withPackageRegistry } from '@myriaddreamin/typst.ts/options.init'
import type { IncrementalServer, TypstCompiler } from '@myriaddreamin/typst.ts/compiler'
import compilerWasmUrl from '@myriaddreamin/typst-ts-web-compiler/wasm?url'
import { ProjectAccessModel } from './accessModel'
import { idbEntries, idbGet, idbPut } from './idb'
import { ServerPackageRegistry, type PackageSpec } from './packageRegistry'
import type { FromWorker, ManifestEntry, ToWorker, WasmDiagnostic } from './protocol'

declare const self: DedicatedWorkerGlobalScope

const post = (msg: FromWorker, transfer: Transferable[] = []) => self.postMessage(msg, transfer)

function syncGet(url: string): { data: Uint8Array; hash: string | null } | undefined {
  const req = new XMLHttpRequest()
  req.open('GET', url, false)
  // A binary-safe string: each byte becomes one char code.
  req.overrideMimeType('text/plain; charset=x-user-defined')
  try {
    req.send(null)
  } catch {
    return undefined
  }
  if (req.status !== 200) return undefined
  const text = req.responseText
  const out = new Uint8Array(text.length)
  for (let i = 0; i < text.length; i++) out[i] = text.charCodeAt(i) & 0xff
  return { data: out, hash: req.getResponseHeader('X-Content-Sha256') }
}

const files = new ProjectAccessModel(
  (rel) => syncGet(`/api/sync/pull?path=${encodeURIComponent(rel)}`),
  (entry, data) => void idbPut('files', entry.hash, data),
)

let compiler: TypstCompiler | null = null
let incr: IncrementalServer | null = null
let releaseIncr: (() => void) | null = null
let incrMain = ''
// Set when a new incremental server is opened and cleared by its first
// successful compile: until then every delta is a whole document, even
// if the compiles in between failed.
let pendingReset = false
let queue: Promise<unknown> = Promise.resolve()

const FONT_CACHE = 'typstify-wasm-fonts-v1'

/** Font bytes, from Cache Storage when possible: fonts are large and their
 * URLs are versioned, so they never need to be fetched twice. */
async function fetchFont(url: string): Promise<Uint8Array | null> {
  let cache: Cache | null = null
  try {
    cache = await caches.open(FONT_CACHE)
    const hit = await cache.match(url)
    if (hit) return new Uint8Array(await hit.arrayBuffer())
  } catch {
    cache = null
  }
  try {
    const res = await fetch(url, { credentials: 'include' })
    if (!res.ok) return null
    if (cache) void cache.put(url, res.clone()).catch(() => undefined)
    return new Uint8Array(await res.arrayBuffer())
  } catch {
    return null
  }
}

async function init(fontUrls: string[]) {
  const t0 = performance.now()
  const fonts = (await Promise.all(fontUrls.map(fetchFont))).filter((f): f is Uint8Array => !!f)
  const stored = await idbEntries('packages')
  const registry = new ServerPackageRegistry(
    files,
    (s: PackageSpec) => syncGet(`/api/wasm/packages/${s.namespace}/${s.name}/${s.version}`)?.data,
    stored,
    (key, data) => void idbPut('packages', key, data),
  )
  const c = createTypstCompiler()
  await c.init({
    getModule: () => compilerWasmUrl,
    beforeBuild: [loadFonts(fonts, { assets: false }), withAccessModel(files), withPackageRegistry(registry)],
  })
  compiler = c
  post({
    type: 'ready',
    fontCount: fonts.length,
    ms: Math.round(performance.now() - t0),
  })
}

/** Seeds file contents cached in IndexedDB for the new manifest. Shadows
 * are dropped: a saved file is now read from the server, and a buffer
 * from another file or project must not hide what is on disk. */
async function setManifest(entries: ManifestEntry[]) {
  files.setManifest(entries)
  compiler?.resetShadow()
  await Promise.all(
    entries.map(async (e) => {
      if (!files.missing('/' + e.path)) return
      const data = await idbGet('files', e.hash)
      if (data) files.seed('/' + e.path, e.hash, data)
    }),
  )
}

/** Opens (or reopens) the long-lived incremental server. withIncrementalServer
 * frees it when its callback settles, so the callback stays pending until
 * the next reset. */
function openIncr(c: TypstCompiler): Promise<IncrementalServer> {
  releaseIncr?.()
  releaseIncr = null
  return new Promise((resolve, reject) => {
    c.withIncrementalServer(
      (srv) =>
        new Promise<void>((release) => {
          releaseIncr = release
          resolve(srv)
        }),
    ).catch(reject)
  })
}

async function compile(id: number, mainPath: string, source: string, reset: boolean) {
  const c = compiler
  if (!c) throw new Error('compiler not ready')
  const t0 = performance.now()
  const main = '/' + mainPath
  // Only the file being edited is shadowed; the previous main file is read
  // from disk again (its unsaved edits were abandoned with it).
  if (incrMain !== main) c.resetShadow()
  c.addSource(main, source)

  if (!incr || reset || incrMain !== main) {
    incr = null
    incr = await openIncr(c)
    incrMain = main
    pendingReset = true
  }
  const res = await c.compile({
    mainFilePath: main,
    root: '/',
    incrementalServer: incr,
    diagnostics: 'full',
  })
  const diagnostics = (res.diagnostics ?? []) as WasmDiagnostic[]
  const ms = Math.round(performance.now() - t0)
  if (!res.result) {
    // A failed compile leaves the server where it was; the next successful
    // one still diffs against the last good document.
    post({ type: 'compiled', id, action: 'none', diagnostics, ms })
    return
  }
  const delta = res.result
  const action = pendingReset ? 'reset' : 'merge'
  pendingReset = false
  post({ type: 'compiled', id, action, delta, diagnostics, ms }, [delta.buffer])
}

// Heading pages via a wrapper document, the same introspection
// typst.QueryHeadingPages does on the server.
const ANCHOR_LABEL = '__typstify_heading_pages'

async function anchors(id: number, mainPath: string) {
  const c = compiler
  if (!c) throw new Error('compiler not ready')
  const wrapper = '/__typstify_anchors__.typ'
  c.addSource(
    wrapper,
    `#include ${JSON.stringify('/' + mainPath)}\n#context [#metadata(query(heading).map(h => h.location().page()))<${ANCHOR_LABEL}>]\n`,
  )
  const pages = await c.runWithWorld({ mainFilePath: wrapper, root: '/' }, async (world) => {
    const r = await world.compile({ diagnostics: 'none' })
    if (r.hasError) return null
    const found = (await world.query({
      selector: `<${ANCHOR_LABEL}>`,
      field: 'value',
    })) as unknown
    const list = Array.isArray(found) ? found[0] : null
    return Array.isArray(list) ? (list as number[]) : null
  })
  post({ type: 'anchors', id, pages })
}

self.onmessage = (ev: MessageEvent<ToWorker>) => {
  const msg = ev.data
  // One task at a time: the compiler is not reentrant.
  queue = queue.then(async () => {
    try {
      switch (msg.type) {
        case 'init':
          await init(msg.fontUrls)
          break
        case 'manifest':
          await setManifest(msg.entries)
          break
        case 'compile':
          await compile(msg.id, msg.mainPath, msg.source, msg.reset)
          break
        case 'anchors':
          await anchors(msg.id, msg.mainPath)
          break
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      if (msg.type === 'init') post({ type: 'fatal', message })
      else if ('id' in msg) post({ type: 'error', id: msg.id, message })
    }
  })
}
