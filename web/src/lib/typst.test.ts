import { describe, expect, it } from 'vitest'
import { hasChessbookImport, typstString } from './typst'

describe('typstString', () => {
  it('quotes and escapes backslashes and quotes', () => {
    expect(typstString('Giải "Dương Sinh" \\ 2026')).toBe('"Giải \\"Dương Sinh\\" \\\\ 2026"')
  })
  it('keeps markup characters literal (string literal, no escaping needed)', () => {
    expect(typstString('*bold* $x$ #let _u_')).toBe('"*bold* $x$ #let _u_"')
  })
  it('turns newlines into spaces', () => {
    expect(typstString('a\nb\r\nc')).toBe('"a b c"')
  })
})

describe('hasChessbookImport', () => {
  it('detects package and relative imports', () => {
    expect(hasChessbookImport('#import "@local/chessbook:0.1.0": *')).toBe(true)
    expect(hasChessbookImport('#import "../lib/lib.typ": *')).toBe(true)
    expect(hasChessbookImport('= Title')).toBe(false)
  })
})
