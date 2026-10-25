import type { ManifestEntry } from './protocol'

/** Reads one project file by its project-relative path, synchronously
 * (a blocking XHR in the worker), with the server's SHA-256 of the bytes
 * when it sent one. Undefined when the file is missing. */
export type SyncFileReader = (relPath: string) => { data: Uint8Array; hash: string | null } | undefined

/** Prefix typst.ts reserves for in-memory files (unpacked packages). */
export const MEMORY_PREFIX = '/@memory/'

/**
 * ProjectAccessModel is the compiler's view of the open project: file
 * metadata comes from the server manifest, contents are read lazily the
 * first time the compiler asks and kept by content hash, so a file is
 * downloaded once per version. Files under /@memory/ (unpacked packages)
 * are held entirely in memory.
 *
 * The buffer being edited is not read from here: the worker passes it to
 * the compiler with addSource, which shadows this model.
 */
export class ProjectAccessModel {
  private manifest = new Map<string, ManifestEntry>()
  private contents = new Map<string, { hash: string; data: Uint8Array }>()
  private memory = new Map<string, { data: Uint8Array; mtime: Date }>()
  private readFile: SyncFileReader
  private onFetched?: (entry: ManifestEntry, data: Uint8Array) => void

  /** onFetched is called with every file fetched from the server, e.g. to persist it. */
  constructor(readFile: SyncFileReader, onFetched?: (entry: ManifestEntry, data: Uint8Array) => void) {
    this.readFile = readFile
    this.onFetched = onFetched
  }

  /** Replaces the file list; cached contents of changed files are dropped. */
  setManifest(entries: ManifestEntry[]) {
    this.manifest = new Map(entries.map((e) => ['/' + e.path, e]))
    for (const [p, c] of this.contents) {
      if (this.manifest.get(p)?.hash !== c.hash) this.contents.delete(p)
    }
  }

  /** Whether path's current version still has to be read (or seeded). */
  missing(path: string): boolean {
    const e = this.manifest.get(path)
    return !!e && this.contents.get(path)?.hash !== e.hash
  }

  /** Seeds the content cache (from IndexedDB), only if the hash matches. */
  seed(path: string, hash: string, data: Uint8Array) {
    if (this.manifest.get(path)?.hash === hash) this.contents.set(path, { hash, data })
  }

  entries(): ManifestEntry[] {
    return [...this.manifest.values()]
  }

  insertFile(path: string, data: Uint8Array, mtime: Date) {
    this.memory.set(path, { data, mtime })
  }

  removeFile(path: string) {
    this.memory.delete(path)
  }

  getMTime(path: string): Date | undefined {
    if (path.startsWith(MEMORY_PREFIX)) return this.memory.get(path)?.mtime
    const e = this.manifest.get(path)
    return e ? new Date(e.mod_time) : undefined
  }

  isFile(path: string): boolean | undefined {
    if (path.startsWith(MEMORY_PREFIX)) return this.memory.has(path)
    return this.manifest.has(path)
  }

  getRealPath(path: string): string | undefined {
    return path
  }

  readAll(path: string): Uint8Array | undefined {
    if (path.startsWith(MEMORY_PREFIX)) return this.memory.get(path)?.data
    const e = this.manifest.get(path)
    if (!e) return undefined
    const cached = this.contents.get(path)
    if (cached && cached.hash === e.hash) return cached.data
    const got = this.readFile(e.path)
    if (!got) return undefined
    // The file changed after the manifest was taken: use the bytes for
    // this compile, but never cache them under the manifest's hash.
    if (got.hash !== e.hash) return got.data
    this.contents.set(path, { hash: e.hash, data: got.data })
    this.onFetched?.(e, got.data)
    return got.data
  }
}
