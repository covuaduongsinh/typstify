// How the service worker (src/sw/sw.ts) answers a request. Kept free of
// DOM and worker types so both the worker and the tests can import it.

export type Strategy = 'bypass' | 'network-first' | 'cache-first' | 'stale-while-revalidate'

const FONT_HOSTS = new Set(['fonts.googleapis.com', 'fonts.gstatic.com'])

/** Static files at the site root that keep their name across builds. */
const ROOT_STATIC = /^\/(manifest\.webmanifest|favicon\.svg|icons\.svg|icons\/[^/]+\.png)$/

export function swStrategy(url: URL, origin: string, method: string, mode: string): Strategy {
  if (method !== 'GET') return 'bypass'
  if (url.origin === origin) {
    // Live data is never served from a cache.
    if (/^\/(api|ws|preview)(\/|$)/.test(url.pathname)) return 'bypass'
    if (mode === 'navigate') return 'network-first'
    // Content-hashed by Vite: a name never changes content.
    if (url.pathname.startsWith('/assets/')) return 'cache-first'
    if (ROOT_STATIC.test(url.pathname)) return 'stale-while-revalidate'
    return 'bypass'
  }
  if (FONT_HOSTS.has(url.hostname)) return 'stale-while-revalidate'
  // Other hosts (the compiler worker's font CDN keeps its own cache).
  return 'bypass'
}

/** Build outputs worth caching at install: everything but the large
 * compiler/renderer WebAssembly (cached on first use instead), source maps
 * and the worker itself. */
export function precacheList(bundleFiles: string[], rootFiles: string[]): string[] {
  const out = new Set<string>(['/'])
  for (const f of bundleFiles) {
    if (f === 'index.html' || f === 'sw.js' || f.endsWith('.map') || f.endsWith('.wasm')) continue
    out.add('/' + f)
  }
  for (const f of rootFiles) out.add('/' + f)
  return [...out].sort()
}
