import { MEMORY_PREFIX } from './accessModel'

export interface PackageSpec {
  namespace: string
  name: string
  version: string
}

export interface PackageResolveContext {
  untar(data: Uint8Array, cb: (path: string, data: Uint8Array, mtime: number) => void): void
}

/** Fetches a package .tar.gz synchronously (blocking XHR in the worker). */
export type SyncArchiveFetcher = (spec: PackageSpec) => Uint8Array | undefined

export interface WritableFiles {
  insertFile(path: string, data: Uint8Array, mtime: Date): void
}

export const packageKey = (s: PackageSpec) => `${s.namespace}/${s.name}/${s.version}`

/**
 * ServerPackageRegistry resolves `@namespace/name:version` imports through
 * GET /api/wasm/packages/... and unpacks them into the access model.
 *
 * `stored` holds archives persisted from earlier sessions (IndexedDB).
 * Registry (@preview) versions never change, so a stored copy is used
 * without asking the server; @local packages are re-fetched once per
 * session and the stored copy is only the offline fallback.
 */
export class ServerPackageRegistry {
  private resolved = new Map<string, string>()
  private files: WritableFiles
  private fetchArchive: SyncArchiveFetcher
  private stored: Map<string, Uint8Array>
  private onFetched?: (key: string, data: Uint8Array) => void

  constructor(
    files: WritableFiles,
    fetchArchive: SyncArchiveFetcher,
    stored: Map<string, Uint8Array> = new Map(),
    onFetched?: (key: string, data: Uint8Array) => void,
  ) {
    this.files = files
    this.fetchArchive = fetchArchive
    this.stored = stored
    this.onFetched = onFetched
  }

  resolve(spec: PackageSpec, context: PackageResolveContext): string | undefined {
    const key = packageKey(spec)
    const done = this.resolved.get(key)
    if (done) return done

    let data: Uint8Array | undefined
    const stored = this.stored.get(key)
    if (stored && spec.namespace === 'preview') {
      data = stored
    } else {
      data = this.fetchArchive(spec)
      if (data) this.onFetched?.(key, data)
      else data = stored
    }
    if (!data) return undefined

    const dir = `${MEMORY_PREFIX}packages/${key}`
    context.untar(data, (path, bytes, mtime) => {
      this.files.insertFile(`${dir}/${path.replace(/^\.?\//, '')}`, bytes, new Date(mtime))
    })
    this.resolved.set(key, dir)
    return dir
  }
}
