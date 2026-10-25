import { describe, expect, it } from 'vitest'
import { ProjectAccessModel } from './accessModel'
import { diagnosticStart, formatDiagnostics } from './diagnostics'
import { ServerPackageRegistry, type PackageResolveContext } from './packageRegistry'
import type { ManifestEntry } from './protocol'
import { findSourceLine } from './sourceMap'

const bytes = (s: string) => new TextEncoder().encode(s)
const entry = (path: string, hash: string): ManifestEntry => ({
  path,
  hash,
  size: 1,
  mod_time: '2026-10-25T00:00:00Z',
})

describe('ProjectAccessModel', () => {
  it('reads lazily, caches by hash and refetches changed files', () => {
    const reads: string[] = []
    let serverHash = 'h1'
    const am = new ProjectAccessModel((rel) => {
      reads.push(rel)
      return { data: bytes(`content of ${rel}`), hash: serverHash }
    })
    am.setManifest([entry('chap/a.typ', 'h1')])

    expect(am.isFile('/chap/a.typ')).toBe(true)
    expect(am.isFile('/missing.typ')).toBe(false)
    expect(am.readAll('/missing.typ')).toBeUndefined()
    expect(new TextDecoder().decode(am.readAll('/chap/a.typ'))).toBe('content of chap/a.typ')
    am.readAll('/chap/a.typ')
    expect(reads).toEqual(['chap/a.typ'])

    serverHash = 'h2'
    am.setManifest([entry('chap/a.typ', 'h2')])
    am.readAll('/chap/a.typ')
    expect(reads).toEqual(['chap/a.typ', 'chap/a.typ'])
  })

  it('never caches bytes whose hash differs from the manifest', () => {
    const persisted: string[] = []
    let reads = 0
    const am = new ProjectAccessModel(
      () => {
        reads++
        return { data: bytes('newer'), hash: 'h2' }
      },
      (e) => void persisted.push(e.hash),
    )
    am.setManifest([entry('a.typ', 'h1')])
    expect(new TextDecoder().decode(am.readAll('/a.typ'))).toBe('newer')
    am.readAll('/a.typ')
    expect(reads).toBe(2)
    expect(persisted).toEqual([])
    expect(am.missing('/a.typ')).toBe(true)
  })

  it('uses seeded contents only when the hash matches', () => {
    let reads = 0
    const am = new ProjectAccessModel(() => {
      reads++
      return { data: bytes('server'), hash: 'h1' }
    })
    am.setManifest([entry('a.typ', 'h1')])
    am.seed('/a.typ', 'stale', bytes('old'))
    expect(new TextDecoder().decode(am.readAll('/a.typ'))).toBe('server')
    am.setManifest([entry('b.typ', 'h9')])
    am.seed('/b.typ', 'h9', bytes('cached'))
    expect(new TextDecoder().decode(am.readAll('/b.typ'))).toBe('cached')
    expect(reads).toBe(1)
  })

  it('keeps /@memory/ files in memory', () => {
    const am = new ProjectAccessModel(() => undefined)
    const mtime = new Date(5)
    am.insertFile('/@memory/packages/x/lib.typ', bytes('pkg'), mtime)
    expect(am.isFile('/@memory/packages/x/lib.typ')).toBe(true)
    expect(am.getMTime('/@memory/packages/x/lib.typ')).toBe(mtime)
  })
})

describe('ServerPackageRegistry', () => {
  const ctx: PackageResolveContext = {
    untar: (_data, cb) => {
      cb('typst.toml', bytes('[package]'), 0)
      cb('./src/lib.typ', bytes('#let x = 1'), 0)
    },
  }
  const files = () => {
    const inserted: string[] = []
    return { inserted, insertFile: (p: string) => void inserted.push(p) }
  }

  it('unpacks once per package into /@memory/', () => {
    const f = files()
    let fetches = 0
    const reg = new ServerPackageRegistry(f, () => {
      fetches++
      return bytes('tgz')
    })
    const spec = { namespace: 'local', name: 'chessbook', version: '0.1.0' }
    expect(reg.resolve(spec, ctx)).toBe('/@memory/packages/local/chessbook/0.1.0')
    reg.resolve(spec, ctx)
    expect(fetches).toBe(1)
    expect(f.inserted).toEqual([
      '/@memory/packages/local/chessbook/0.1.0/typst.toml',
      '/@memory/packages/local/chessbook/0.1.0/src/lib.typ',
    ])
  })

  it('serves stored @preview packages without the network and @local ones only offline', () => {
    const stored = new Map([
      ['preview/demo/1.0.0', bytes('a')],
      ['local/mine/0.1.0', bytes('b')],
    ])
    const asked: string[] = []
    const reg = new ServerPackageRegistry(
      files(),
      (s) => {
        asked.push(s.name)
        return undefined // offline
      },
      stored,
    )
    expect(reg.resolve({ namespace: 'preview', name: 'demo', version: '1.0.0' }, ctx)).toBeTruthy()
    expect(reg.resolve({ namespace: 'local', name: 'mine', version: '0.1.0' }, ctx)).toBeTruthy()
    expect(reg.resolve({ namespace: 'preview', name: 'gone', version: '1.0.0' }, ctx)).toBeUndefined()
    expect(asked).toEqual(['mine', 'gone'])
  })
})

describe('diagnostics', () => {
  it('converts 0-based ranges to 1-based positions', () => {
    expect(diagnosticStart('2:4-2:9')).toEqual({ line: 3, col: 5 })
    expect(diagnosticStart('garbage')).toBeNull()
  })

  it('formats like the CLI', () => {
    const out = formatDiagnostics([
      {
        package: '',
        path: '/main.typ',
        severity: 'Error',
        range: '0:0-0:3',
        message: 'unknown variable: foo',
      },
    ])
    expect(out).toBe('main.typ:1:1: error: unknown variable: foo')
  })
})

describe('findSourceLine', () => {
  const src = [
    '= Chương 1',
    'Khai cuộc Ý rất phổ biến.',
    '',
    '== Ghi chú',
    'Khai cuộc Ý *rất* phổ biến.',
    'Kết thúc',
  ].join('\n')

  it('finds exact runs, preferring the one nearest the estimate', () => {
    expect(findSourceLine(src, 'Khai cuộc Ý rất phổ biến.', 1)).toBe(2)
    expect(findSourceLine(src, 'Kết thúc', 1)).toBe(6)
  })

  it('falls back to word overlap when markup splits the run', () => {
    expect(findSourceLine(src, 'Khai cuộc Ý rất phổ biến.', 5)).toBe(2) // exact wins
    const marked = 'Mở đầu\nPhòng thủ *Sicilia* rất sắc bén.\nKết'
    expect(findSourceLine(marked, 'Phòng thủ Sicilia rất sắc bén.', 1)).toBe(2)
    expect(findSourceLine(src, 'Ghi chú', 1)).toBe(4)
  })

  it('returns null for text that is not in the source', () => {
    expect(findSourceLine(src, 'x', 1)).toBeNull()
    expect(findSourceLine(src, 'hoàn toàn khác', 1)).toBeNull()
  })
})
