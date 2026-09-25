import { describe, expect, it } from 'vitest'
import {
  CHESSBOOK_IMPORT,
  hasChessbookImport,
  repairChessImports,
  requiresChessImport,
  typstString,
} from './typst'

describe('typstString', () => {
  it('escapes quotes and backslashes', () => {
    expect(typstString('a "b" \\ c')).toBe('"a \\"b\\" \\\\ c"')
  })
  it('collapses newlines so a multiline title cannot break the string', () => {
    expect(typstString('first\nsecond\r\nthird')).toBe('"first second third"')
  })
  it('wraps empty text in quotes', () => {
    expect(typstString('')).toBe('""')
  })
})

describe('hasChessbookImport', () => {
  it('detects package and relative imports', () => {
    expect(hasChessbookImport('#import "@local/chessbook:0.1.0": *')).toBe(true)
    expect(hasChessbookImport('#import "lib/lib.typ": *')).toBe(true)
    expect(hasChessbookImport('#import "chess_template.typ": *')).toBe(true)
    expect(hasChessbookImport('#let game-header() = {}')).toBe(true)
    expect(hasChessbookImport('= Just a document')).toBe(false)
  })
})

describe('repairChessImports', () => {
  it('prepends import when document uses chess functions without import', () => {
    const doc = '= My Game\n\n#game-header(white: "A", black: "B")'
    expect(requiresChessImport(doc)).toBe(true)
    const repaired = repairChessImports(doc)
    expect(repaired.startsWith(CHESSBOOK_IMPORT)).toBe(true)
    expect(repaired).toContain('#game-header(')
  })

  it('does nothing when import already exists', () => {
    const doc = `${CHESSBOOK_IMPORT}\n\n#game-header(white: "A")`
    expect(repairChessImports(doc)).toBe(doc)
  })
})
