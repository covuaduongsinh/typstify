import type { FromWorker, ManifestEntry, ToWorker, WasmDiagnostic } from './protocol'

// Typst's default text fonts (Libertinus Serif, New Computer Modern, DejaVu
// Sans Mono), the ones the CLI embeds. The WASM build has none, so they come
// from the same CDN typst.ts uses by default; the worker keeps them in Cache
// Storage after the first load.
const DEFAULT_FONT_BASE = 'https://cdn.jsdelivr.net/gh/typst/typst-assets@v0.13.1/files/fonts/'
const DEFAULT_FONTS = [
  'DejaVuSansMono.ttf',
  'DejaVuSansMono-Bold.ttf',
  'DejaVuSansMono-Oblique.ttf',
  'DejaVuSansMono-BoldOblique.ttf',
  'LibertinusSerif-Regular.otf',
  'LibertinusSerif-Bold.otf',
  'LibertinusSerif-Italic.otf',
  'LibertinusSerif-BoldItalic.otf',
  'LibertinusSerif-Semibold.otf',
  'LibertinusSerif-SemiboldItalic.otf',
  'NewCM10-Regular.otf',
  'NewCM10-Bold.otf',
  'NewCM10-Italic.otf',
  'NewCM10-BoldItalic.otf',
  'NewCMMath-Regular.otf',
  'NewCMMath-Book.otf',
  'NewCMMath-Bold.otf',
].map((f) => DEFAULT_FONT_BASE + f)

export interface CompileOutcome {
  action: 'reset' | 'merge' | 'none'
  delta?: Uint8Array
  diagnostics: WasmDiagnostic[]
  ms: number
}

type Pending = { resolve: (m: FromWorker) => void; reject: (e: Error) => void }

/** Whether this browser can run the in-browser compiler at all. */
export const wasmSupported = () =>
  typeof WebAssembly === 'object' && typeof Worker === 'function' && typeof indexedDB === 'object'

/**
 * WasmCompilerClient owns the compiler worker. One instance serves the
 * whole page (the worker is ~30 MB of WASM plus fonts), created on first
 * use by get().
 */
export class WasmCompilerClient {
  private static instance: WasmCompilerClient | null = null
  static get(): WasmCompilerClient {
    return (WasmCompilerClient.instance ??= new WasmCompilerClient())
  }

  readonly ready: Promise<{ fontCount: number; ms: number }>
  private worker: Worker
  private nextId = 1
  private pending = new Map<number, Pending>()
  private onReady!: (v: { fontCount: number; ms: number }) => void
  private onFatal!: (e: Error) => void

  private constructor() {
    this.worker = new Worker(new URL('./compiler.worker.ts', import.meta.url), {
      type: 'module',
    })
    this.worker.onmessage = (ev: MessageEvent<FromWorker>) => this.receive(ev.data)
    this.worker.onerror = (ev) => this.fail(new Error(ev.message || 'Không chạy được trình biên dịch WASM'))
    this.ready = new Promise((resolve, reject) => {
      this.onReady = resolve
      this.onFatal = reject
    })
    // Callers attach their own handlers; keep an unawaited rejection quiet.
    this.ready.catch(() => undefined)
    void this.start()
  }

  private async start() {
    try {
      const res = await fetch('/api/wasm/fonts', { credentials: 'include' })
      const fonts: { url: string }[] = res.ok ? await res.json() : []
      this.send({
        type: 'init',
        fontUrls: [...fonts.map((f) => f.url), ...DEFAULT_FONTS],
      })
    } catch (err) {
      this.fail(err instanceof Error ? err : new Error(String(err)))
    }
  }

  private fail(err: Error) {
    this.onFatal(err)
    for (const p of this.pending.values()) p.reject(err)
    this.pending.clear()
    WasmCompilerClient.instance = null
    this.worker.terminate()
  }

  private send(msg: ToWorker, transfer: Transferable[] = []) {
    this.worker.postMessage(msg, transfer)
  }

  private receive(msg: FromWorker) {
    if (msg.type === 'ready') return this.onReady({ fontCount: msg.fontCount, ms: msg.ms })
    if (msg.type === 'fatal') return this.fail(new Error(msg.message))
    const p = this.pending.get(msg.id)
    if (!p) return
    this.pending.delete(msg.id)
    if (msg.type === 'error') p.reject(new Error(msg.message))
    else p.resolve(msg)
  }

  private request<T extends FromWorker>(build: (id: number) => ToWorker): Promise<T> {
    const id = this.nextId++
    return new Promise<T>((resolve, reject) => {
      this.pending.set(id, {
        resolve: resolve as (m: FromWorker) => void,
        reject,
      })
      this.send(build(id))
    })
  }

  /** Reloads the project file list; call after opening a project or saving. */
  async refreshManifest(): Promise<void> {
    const res = await fetch('/api/sync/manifest', { credentials: 'include' })
    if (!res.ok) throw new Error(`manifest: ${res.status}`)
    const entries = (await res.json()) as ManifestEntry[]
    await this.ready
    this.send({ type: 'manifest', entries })
  }

  async compile(mainPath: string, source: string, reset = false): Promise<CompileOutcome> {
    await this.ready
    const m = await this.request<Extract<FromWorker, { type: 'compiled' }>>((id) => ({
      type: 'compile',
      id,
      mainPath,
      source,
      reset,
    }))
    return {
      action: m.action,
      delta: m.delta,
      diagnostics: m.diagnostics,
      ms: m.ms,
    }
  }

  /** 1-based page of every heading in document order, or null. */
  async headingPages(mainPath: string): Promise<number[] | null> {
    await this.ready
    const m = await this.request<Extract<FromWorker, { type: 'anchors' }>>((id) => ({ type: 'anchors', id, mainPath }))
    return m.pages
  }
}
