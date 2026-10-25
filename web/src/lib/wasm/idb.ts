// A tiny promise wrapper over IndexedDB for the compiler worker's caches:
// package archives and project file contents (keyed by content hash), so
// a reload or a dropped connection does not mean downloading them again.
// Every call degrades to a no-op when IndexedDB is unavailable (private
// mode, blocked storage).

const DB_NAME = 'typstify-wasm'
const DB_VERSION = 1
export type StoreName = 'packages' | 'files'

let dbPromise: Promise<IDBDatabase | null> | null = null

function open(): Promise<IDBDatabase | null> {
  if (dbPromise) return dbPromise
  dbPromise = new Promise((resolve) => {
    try {
      const req = indexedDB.open(DB_NAME, DB_VERSION)
      req.onupgradeneeded = () => {
        const db = req.result
        for (const s of ['packages', 'files']) {
          if (!db.objectStoreNames.contains(s)) db.createObjectStore(s)
        }
      }
      req.onsuccess = () => resolve(req.result)
      req.onerror = () => resolve(null)
    } catch {
      resolve(null)
    }
  })
  return dbPromise
}

function run<T>(
  store: StoreName,
  mode: IDBTransactionMode,
  fn: (s: IDBObjectStore) => IDBRequest<T>,
): Promise<T | undefined> {
  return open().then(
    (db) =>
      new Promise<T | undefined>((resolve) => {
        if (!db) return resolve(undefined)
        try {
          const req = fn(db.transaction(store, mode).objectStore(store))
          req.onsuccess = () => resolve(req.result)
          req.onerror = () => resolve(undefined)
        } catch {
          resolve(undefined)
        }
      }),
  )
}

export const idbGet = (store: StoreName, key: string) => run<Uint8Array>(store, 'readonly', (s) => s.get(key))

export const idbPut = (store: StoreName, key: string, value: Uint8Array) =>
  run(store, 'readwrite', (s) => s.put(value, key)).then(() => undefined)

/** All entries of a store, used once at startup for the (small) package store. */
export async function idbEntries(store: StoreName): Promise<Map<string, Uint8Array>> {
  const [keys, values] = await Promise.all([
    run<IDBValidKey[]>(store, 'readonly', (s) => s.getAllKeys()),
    run<Uint8Array[]>(store, 'readonly', (s) => s.getAll()),
  ])
  const out = new Map<string, Uint8Array>()
  if (keys && values && keys.length === values.length) {
    keys.forEach((k, i) => out.set(String(k), values[i]))
  }
  return out
}
