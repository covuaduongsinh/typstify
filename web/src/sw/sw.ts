/// <reference lib="webworker" />
// Typstify's service worker, built to /sw.js (vite.config.ts injects the
// precache list and version). See docs/plans/plan_phase7_pwa.md.
import { swStrategy } from '../lib/swStrategy'

declare const self: ServiceWorkerGlobalScope & {
  __PRECACHE_MANIFEST__: string[]
  __SW_VERSION__: string
}

const PRECACHE = self.__PRECACHE_MANIFEST__
const ASSET_PREFIX = 'typstify-assets-'
const ASSETS = ASSET_PREFIX + self.__SW_VERSION__
const RUNTIME = 'typstify-runtime-v1'
const RUNTIME_MAX = 80
// Every route serves the same index.html; this version's copy is kept under
// one key and is the only page ever served from a cache, so a page always
// matches the scripts cached next to it.
const SHELL = '/'

const isHtml = (res: Response) => (res.headers.get('content-type') ?? '').includes('text/html')

/** A response worth keeping: complete, successful, and of the expected
 * kind (a page for the shell, anything but a page for other files: the
 * server's SPA fallback must never be stored under a script's URL). */
function cacheable(res: Response, page: boolean): boolean {
  return res.ok && res.status !== 206 && isHtml(res) === page
}

self.addEventListener('install', (event) => {
  event.waitUntil(
    (async () => {
      const cache = await caches.open(ASSETS)
      // All or nothing: a half-filled cache would replace a complete one.
      // If anything fails (flaky network, deploy still copying files) the
      // install fails and the current version stays in charge.
      await Promise.all(
        PRECACHE.map(async (url) => {
          const res = await fetch(url, { cache: 'no-cache' })
          if (!cacheable(res, url === SHELL)) throw new Error(`precache ${url}: ${res.status}`)
          await cache.put(url, res)
        }),
      )
      // The compiler's WebAssembly (~30 MB) is cached on first use, not
      // here; carry it over from earlier versions (hashed names: safe).
      for (const name of await caches.keys()) {
        if (!name.startsWith(ASSET_PREFIX) || name === ASSETS) continue
        const old = await caches.open(name)
        for (const req of await old.keys()) {
          if (!new URL(req.url).pathname.endsWith('.wasm') || (await cache.match(req))) continue
          const res = await old.match(req)
          if (res) await cache.put(req, res)
        }
      }
    })(),
  )
  // No skipWaiting here: the page asks for it (SKIP_WAITING) when the user
  // chooses to reload.
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    (async () => {
      // Keep the previous version too: tabs still running it lazy-load its
      // chunks, which the server may no longer have. caches.keys() lists in
      // creation order, so the last asset caches are the newest.
      const assetCaches = (await caches.keys()).filter((n) => n.startsWith(ASSET_PREFIX) && n !== ASSETS)
      for (const name of assetCaches.slice(0, -1)) await caches.delete(name)
      await self.clients.claim()
    })(),
  )
})

self.addEventListener('message', (event) => {
  if (event.data === 'SKIP_WAITING') void self.skipWaiting()
})

async function shell(): Promise<Response | undefined> {
  return (await caches.open(ASSETS)).match(SHELL)
}

/** Pages: the network when the server answers, this version's shell when
 * it does not (offline, or the server down behind a proxy: 5xx). */
async function page(request: Request): Promise<Response> {
  try {
    const res = await fetch(request)
    if (res.status >= 500) return (await shell()) ?? res
    return res
  } catch (err) {
    const cached = await shell()
    if (cached) return cached
    throw err
  }
}

async function cacheFirst(request: Request): Promise<Response> {
  if (request.headers.has('range')) return fetch(request)
  const cached = await caches.match(request)
  if (cached) return cached
  const res = await fetch(request)
  if (cacheable(res, false)) {
    const copy = res.clone()
    void caches
      .open(ASSETS)
      .then((c) => c.put(request, copy))
      .catch(() => undefined)
  }
  return res
}

async function trimRuntime(cache: Cache) {
  const keys = await cache.keys()
  for (const req of keys.slice(0, Math.max(0, keys.length - RUNTIME_MAX))) await cache.delete(req)
}

async function staleWhileRevalidate(request: Request): Promise<Response> {
  const cached = await caches.match(request)
  const update = fetch(request).then((res) => {
    if (cacheable(res, false)) {
      const copy = res.clone()
      void caches
        .open(RUNTIME)
        .then(async (c) => {
          await c.put(request, copy)
          await trimRuntime(c)
        })
        .catch(() => undefined)
    }
    return res
  })
  if (cached) {
    update.catch(() => undefined)
    return cached
  }
  return update
}

self.addEventListener('fetch', (event) => {
  const request = event.request
  switch (swStrategy(new URL(request.url), self.location.origin, request.method, request.mode)) {
    case 'network-first':
      event.respondWith(page(request))
      break
    case 'cache-first':
      event.respondWith(cacheFirst(request))
      break
    case 'stale-while-revalidate':
      event.respondWith(staleWhileRevalidate(request))
      break
    default:
    // bypass: the browser handles it as if there were no worker.
  }
})
