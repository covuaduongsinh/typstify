import { describe, expect, it } from 'vitest'
import { precacheList, swStrategy } from './swStrategy'

const origin = 'https://typst.example.com'
const s = (path: string, mode = 'cors', method = 'GET') => swStrategy(new URL(path, origin), origin, method, mode)

describe('swStrategy', () => {
  it('never caches live data or non-GET requests', () => {
    expect(s('/api/workspace/tree')).toBe('bypass')
    expect(s('/ws/collab?path=a')).toBe('bypass')
    expect(s('/preview/x')).toBe('bypass')
    expect(s('/api', 'navigate')).toBe('bypass')
    expect(s('/assets/a.js', 'cors', 'POST')).toBe('bypass')
  })

  it('serves pages network-first and hashed assets cache-first', () => {
    expect(s('/?join=abc', 'navigate')).toBe('network-first')
    expect(s('/assets/index-abc123.js')).toBe('cache-first')
    expect(s('/assets/typst_ts_web_compiler_bg-x.wasm')).toBe('cache-first')
    expect(s('/manifest.webmanifest')).toBe('stale-while-revalidate')
    expect(s('/icons/icon-192.png')).toBe('stale-while-revalidate')
    expect(s('/apiary.js')).toBe('bypass')
  })

  it('caches Google Fonts and leaves other hosts alone', () => {
    expect(swStrategy(new URL('https://fonts.gstatic.com/s/roboto.woff2'), origin, 'GET', 'cors')).toBe(
      'stale-while-revalidate',
    )
    expect(swStrategy(new URL('https://cdn.jsdelivr.net/x.otf'), origin, 'GET', 'cors')).toBe('bypass')
  })
})

describe('precacheList', () => {
  it('keeps the app shell and skips WebAssembly, maps and the worker', () => {
    expect(
      precacheList(
        ['index.html', 'sw.js', 'assets/index-1.js', 'assets/index-1.js.map', 'assets/compiler-2.wasm', 'assets/a.css'],
        ['manifest.webmanifest', 'favicon.svg'],
      ),
    ).toEqual(['/', '/assets/a.css', '/assets/index-1.js', '/favicon.svg', '/manifest.webmanifest'])
  })
})
