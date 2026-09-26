import { describe, expect, it } from 'vitest'
import {
  CHESSBOOK_IMPORT,
  hasChessbookImport,
  hoistChessbookImport,
  repairChessImports,
  requiresChessImport,
  stripChessbookMockDefinitions,
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
    expect(hasChessbookImport('#let game-header() = {}')).toBe(false)
    expect(hasChessbookImport('= Just a document')).toBe(false)
  })
})

describe('requiresChessImport', () => {
  it('identifies chessbook functions and helpers', () => {
    expect(requiresChessImport('#concept-box[Important]')).toBe(true)
    expect(requiresChessImport('#chess-quote(author: "GK")[Great move]')).toBe(true)
    expect(requiresChessImport('#instructor-note[Attention]')).toBe(true)
    expect(requiresChessImport('#practice-question(number: 1)')).toBe(true)
    expect(requiresChessImport('#eco-header(code: "C58")')).toBe(true)
    expect(requiresChessImport('#puzzle-card("fen", turn: "w")')).toBe(true)
    expect(requiresChessImport('1. e4 e5 2. #wN f3')).toBe(true)
    expect(requiresChessImport('= Regular document heading')).toBe(false)
  })
})

describe('stripChessbookMockDefinitions', () => {
  it('removes a simple single-function mock', () => {
    const doc = [
      '#import "@local/chessbook:0.1.0": *',
      '',
      '#let puzzle-card(',
      '  fen-str,',
      '  number: 1,',
      ') = {',
      '  block()[]',
      '}',
      '',
      '#puzzle-card("fen", number: 1)',
    ].join('\n')
    const result = stripChessbookMockDefinitions(doc)
    expect(result).not.toContain('#let puzzle-card')
    expect(result).toContain('#puzzle-card("fen", number: 1)')
    expect(result).toContain('#import')
  })

  it('removes multiple mock definitions', () => {
    const doc = [
      '#import "@local/chessbook:0.1.0": *',
      '',
      '#let turn-box(turn) = {',
      '  box()',
      '}',
      '',
      '#let chess-quote(author: "", text-content) = {',
      '  rect()[]',
      '}',
      '',
      '#let instructor-note(note) = {',
      '  rect()[]',
      '}',
      '',
      '#chess-quote(author: "GK")[Hello]',
    ].join('\n')
    const result = stripChessbookMockDefinitions(doc)
    expect(result).not.toContain('#let turn-box')
    expect(result).not.toContain('#let chess-quote')
    expect(result).not.toContain('#let instructor-note')
    expect(result).toContain('#chess-quote(author: "GK")[Hello]')
  })

  it('preserves non-chessbook #let definitions', () => {
    const doc = [
      '#let my-custom-func() = {',
      '  box()',
      '}',
      '',
      '#my-custom-func()',
    ].join('\n')
    expect(stripChessbookMockDefinitions(doc)).toBe(doc)
  })

  it('handles nested braces correctly', () => {
    const doc = [
      '#let game-header(white: "", black: "") = {',
      '  block()[',
      '    #if true {',
      '      text()[hello]',
      '    }',
      '  ]',
      '}',
      '',
      '#game-header(white: "A")',
    ].join('\n')
    const result = stripChessbookMockDefinitions(doc)
    expect(result).not.toContain('#let game-header')
    expect(result).toContain('#game-header(white: "A")')
  })
})

describe('hoistChessbookImport', () => {
  it('moves import from bottom of file to top', () => {
    const doc = [
      '#puzzle-card("fen", number: 1)',
      '',
      '#import "@local/chessbook:0.1.0": *',
    ].join('\n')
    const hoisted = hoistChessbookImport(doc)
    expect(hoisted.startsWith(CHESSBOOK_IMPORT)).toBe(true)
    const lines = hoisted.split('\n')
    expect(lines[0]).toBe(CHESSBOOK_IMPORT)
    // Make sure it's not at the bottom anymore
    expect(hoisted.endsWith('#import "@local/chessbook:0.1.0": *')).toBe(false)
  })

  it('deduplicates multiple imports throughout the file', () => {
    const doc = [
      '#import "@local/chessbook:0.1.0": *',
      '#puzzle-card("fen1")',
      '#import "@local/chessbook:0.1.0": *',
      '#puzzle-card("fen2")',
      '#import "lib/lib.typ": *',
    ].join('\n')
    const hoisted = hoistChessbookImport(doc)
    const matches = hoisted.match(/#import\s+["']@local\/chessbook/g)
    expect(matches).toHaveLength(1)
    expect(hoisted).not.toContain('lib/lib.typ')
    expect(hoisted.startsWith(CHESSBOOK_IMPORT)).toBe(true)
  })

  it('does not touch documents without chess functions or chess imports', () => {
    const doc = '= My regular document\n\nSome text.'
    expect(hoistChessbookImport(doc)).toBe(doc)
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

  it('prepends import when document uses newly added courseware or quote helpers', () => {
    const doc = '= Lesson\n\n#lesson-header(lesson-num: 1)\n#concept-box[Key]'
    expect(requiresChessImport(doc)).toBe(true)
    const repaired = repairChessImports(doc)
    expect(repaired.startsWith(CHESSBOOK_IMPORT)).toBe(true)
  })

  it('hoists import from bottom to line 1 and strips mocks (full repair scenario)', () => {
    const doc = [
      '#puzzle-card(',
      '  "fen",',
      '  number: 1,',
      '  turn: "w",',
      ')',
      '',
      '#let turn-box(turn) = { box() }',
      '',
      '#import "@local/chessbook:0.1.0": *',
    ].join('\n')
    const repaired = repairChessImports(doc)
    expect(repaired.startsWith(CHESSBOOK_IMPORT)).toBe(true)
    expect(repaired).not.toContain('#let turn-box')
    expect(repaired).toContain('#puzzle-card(')
    // Only one import line
    const importCount = (repaired.match(/#import\s+["']@local\/chessbook/g) || []).length
    expect(importCount).toBe(1)
  })

  it('preserves clean documents that already have import at top', () => {
    const doc = `${CHESSBOOK_IMPORT}\n\n#game-header(white: "A")`
    expect(repairChessImports(doc)).toBe(doc)
  })

  it('strips mock definitions when import exists', () => {
    const doc = [
      '#import "@local/chessbook:0.1.0": *',
      '',
      '#let lesson-header(lesson-num: 1) = {',
      '  block()[]',
      '}',
      '',
      '#lesson-header(lesson-num: 1)',
    ].join('\n')
    const result = repairChessImports(doc)
    expect(result).not.toContain('#let lesson-header')
    expect(result).toContain('#lesson-header(lesson-num: 1)')
    expect(result).toContain('#import')
  })
})


