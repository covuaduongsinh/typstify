// Messages between WasmCompilerClient (main thread, client.ts) and the
// compiler worker (compiler.worker.ts).

/** One project file, as listed by GET /api/sync/manifest. */
export interface ManifestEntry {
  path: string
  hash: string
  size: number
  mod_time: string
}

/** The subset of typst.ts' DiagnosticMessage the UI reads. */
export interface WasmDiagnostic {
  package: string
  path: string
  severity: string
  range: string
  message: string
}

export type ToWorker =
  | { type: 'init'; fontUrls: string[] }
  | { type: 'manifest'; entries: ManifestEntry[] }
  | {
      type: 'compile'
      id: number
      mainPath: string
      source: string
      reset: boolean
    }
  | { type: 'anchors'; id: number; mainPath: string }

export type FromWorker =
  | { type: 'ready'; fontCount: number; ms: number }
  | { type: 'fatal'; message: string }
  | {
      type: 'compiled'
      id: number
      /** 'reset' replaces the renderer's document, 'merge' patches it. */
      action: 'reset' | 'merge' | 'none'
      delta?: Uint8Array
      diagnostics: WasmDiagnostic[]
      ms: number
    }
  | { type: 'anchors'; id: number; pages: number[] | null }
  | { type: 'error'; id: number; message: string }
