import { describe, expect, it } from 'vitest'
import { formatDate, formatResult, gameToTypst, movetextToTypst, parsePgn, pgnToTypst } from './pgn'

const TWO_GAMES = `[Event "Giải CLB"]
[White "Nguyễn Văn A"]
[Black "Trần B"]
[Result "*"]

1. e4 {Khai cuộc} e5 2. Nf3 $1 Nc6 *

[Event "Ván 2"]
[White "Lê C"]
[Black "Phạm D"]
[Result "1/2-1/2"]

1. d4 d5 1/2-1/2`

describe('parsePgn', () => {
  it('splits multiple games', () => {
    const games = parsePgn(TWO_GAMES)
    expect(games).toHaveLength(2)
    expect(games[0].headers['White']).toBe('Nguyễn Văn A')
    expect(games[1].headers['Event']).toBe('Ván 2')
  })
  it('removes the result token and takes the result from tags', () => {
    const [g1, g2] = parsePgn(TWO_GAMES)
    expect(g1.movetext).toBe('1. e4 {Khai cuộc} e5 2. Nf3 $1 Nc6')
    expect(g1.result).toBe('') // "*" = unknown
    expect(g2.result).toBe('1/2-1/2')
  })
  it('falls back to the trailing result token when there is no Result tag', () => {
    expect(parsePgn('1. e4 e5 0-1')[0].result).toBe('0-1')
  })
  it('unescapes quotes in tag values', () => {
    expect(parsePgn('[Event "Giải \\"X\\""]\n1. e4')[0].headers['Event']).toBe('Giải "X"')
  })
})

describe('movetextToTypst', () => {
  it('emits moves as string literals, comments as emph, NAGs via #nag', () => {
    expect(movetextToTypst('1. e4 {hay *nhất*} e5 2. Nf3 $14 Nc6')).toBe(
      '#"1. e4 "#emph("hay *nhất*")#" e5 2. Nf3 "#nag("14")#" Nc6"',
    )
  })
  it('keeps variations and black move numbers verbatim', () => {
    expect(movetextToTypst('3. Bc4 (3. Bb5 a6) 3... Nf6')).toBe('#"3. Bc4 (3. Bb5 a6) 3... Nf6"')
  })
})

describe('formatting', () => {
  it('formats results and partial dates', () => {
    expect(formatResult('1-0')).toBe('1 - 0')
    expect(formatResult('1/2-1/2')).toBe('½ - ½')
    expect(formatResult('*')).toBe('')
    expect(formatDate('2026.??.??')).toBe('2026')
    expect(formatDate('????.??.??')).toBe('')
  })
})

describe('gameToTypst', () => {
  it('never invents missing tags', () => {
    const out = gameToTypst(parsePgn('1. e4 e5')[0])
    expect(out).toContain('white-elo: "",')
    expect(out).toContain('white-title: "",')
    expect(out).toContain('result: "",')
    expect(out).not.toMatch(/2700|GM|Hà Nội/)
  })
  it('escapes player names', () => {
    expect(gameToTypst(parsePgn('[White "A \\"B\\""]\n1. e4')[0])).toContain('white: "A \\"B\\"",')
  })
  it('converts every game in a file', () => {
    expect(pgnToTypst(TWO_GAMES).match(/#game-header\(/g)).toHaveLength(2)
  })
})
