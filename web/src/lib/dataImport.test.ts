import { describe, expect, it } from 'vitest'
import {
  generatePuzzleCollectionTypst,
  isLikelyFen,
  parseCsvPuzzles,
  parseFenList,
  parseJsonPuzzles,
} from './dataImport'

describe('isLikelyFen', () => {
  it('identifies standard chess FEN strings', () => {
    expect(
      isLikelyFen('r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6'),
    ).toBe(true)
    expect(isLikelyFen('rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1')).toBe(true)
    expect(isLikelyFen('not a fen')).toBe(false)
    expect(isLikelyFen('')).toBe(false)
  })
})

describe('parseJsonPuzzles', () => {
  it('parses valid JSON puzzle array', () => {
    const json = JSON.stringify([
      {
        fen: 'r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6',
        title: 'Đòn đánh đôi',
        turn: 'w',
        difficulty: 2,
        hint: 'Quan sát Mã',
        solution: '1. dxe4',
      },
      {
        fen: 'r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10',
        title: 'Tấn công f2',
        turn: 'b',
        difficulty: 3,
        solution: '1... Qc5',
      },
    ])

    const result = parseJsonPuzzles(json)
    expect(result).toHaveLength(2)
    expect(result[0].title).toBe('Đòn đánh đôi')
    expect(result[0].turn).toBe('w')
    expect(result[1].turn).toBe('b')
  })

  it('handles array of pure FEN strings in JSON', () => {
    const json = JSON.stringify([
      'r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6',
      'r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10',
    ])
    const result = parseJsonPuzzles(json)
    expect(result).toHaveLength(2)
    expect(result[0].turn).toBe('w')
    expect(result[1].turn).toBe('b')
  })
})

describe('parseCsvPuzzles', () => {
  it('parses CSV with headers', () => {
    const csv = `fen,title,turn,difficulty,hint,solution
"r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6","Đòn đánh đôi",w,2,"Gợi ý","1. dxe4"
"r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10","Tấn công f2",b,3,"","1... Qc5"`

    const result = parseCsvPuzzles(csv)
    expect(result).toHaveLength(2)
    expect(result[0].title).toBe('Đòn đánh đôi')
    expect(result[0].difficulty).toBe(2)
    expect(result[1].title).toBe('Tấn công f2')
  })
})

describe('parseFenList', () => {
  it('parses text with multiple FEN lines', () => {
    const text = `r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6
r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10`

    const result = parseFenList(text)
    expect(result).toHaveLength(2)
    expect(result[0].turn).toBe('w')
    expect(result[1].turn).toBe('b')
  })
})

describe('generatePuzzleCollectionTypst', () => {
  it('generates complete Typst document code', () => {
    const puzzles = [
      {
        fen: 'r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6',
        title: 'Bài 1',
        turn: 'w' as const,
        difficulty: 2,
        solution: '1. dxe4',
      },
    ]

    const code = generatePuzzleCollectionTypst(puzzles, {
      title: 'BỘ ĐỀ 1',
      layout: 'a4-3x4',
      embedMode: 'direct',
    })

    expect(code).toContain('#import "@local/chessbook:0.1.0": *')
    expect(code).toContain('chess-worksheet-init')
    expect(code).toContain('#render-puzzle-collection')
    expect(code).toContain('layout: "a4-3x4"')
  })

  it('generates json_reference mode Typst code', () => {
    const code = generatePuzzleCollectionTypst([], {
      title: 'SÁCH BÀI TẬP',
      layout: '16x24-2x3',
      embedMode: 'json_reference',
      dataPath: 'data/puzzles.json',
    })

    expect(code).toContain('chess-book-init')
    expect(code).toContain('#let puzzle-data = json("data/puzzles.json")')
    expect(code).toContain('layout: "16x24-2x3"')
  })
})
