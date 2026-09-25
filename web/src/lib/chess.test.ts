import { describe, expect, it } from 'vitest'
import {
  boardToFen,
  castlingRights,
  EMPTY_BOARD,
  fenToBoard,
  INITIAL_BOARD,
  type Piece,
  squareName,
} from './chess'

const clone = (b: Piece[][]) => b.map((r) => [...r])

describe('boardToFen', () => {
  it('serialises the starting position', () => {
    expect(boardToFen(INITIAL_BOARD, 'w')).toBe('rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1')
  })
  it('handles an empty board and black to move', () => {
    expect(boardToFen(EMPTY_BOARD, 'b')).toBe('8/8/8/8/8/8/8/8 b - - 0 1')
  })
  it('run-length encodes gaps between pieces', () => {
    const b = clone(EMPTY_BOARD)
    b[0][0] = 'k'
    b[0][7] = 'r'
    b[7][3] = 'Q'
    expect(boardToFen(b, 'w').split(' ')[0]).toBe('k6r/8/8/8/8/8/8/3Q4')
  })
})

describe('fenToBoard', () => {
  it('deserialises starting position correctly', () => {
    const res = fenToBoard('rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1')
    expect(res.turn).toBe('w')
    expect(res.board[0][0]).toBe('r')
    expect(res.board[7][4]).toBe('K')
    expect(res.board[3][3]).toBeNull()
  })

  it('deserialises a puzzle position with black to move', () => {
    const fen = 'r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 6'
    const res = fenToBoard(fen)
    expect(res.turn).toBe('b')
    expect(res.board[0][4]).toBe('k')
    expect(res.board[4][4]).toBe('n')
  })
})

describe('squareName', () => {
  it('converts row/col indices to standard algebraic notation', () => {
    expect(squareName(7, 4)).toBe('e1')
    expect(squareName(4, 4)).toBe('e4')
    expect(squareName(0, 0)).toBe('a8')
    expect(squareName(7, 7)).toBe('h1')
  })
})

describe('castlingRights', () => {
  it('only grants rights with king and rook on home squares', () => {
    const b = clone(INITIAL_BOARD)
    b[7][7] = null // white h-rook gone
    b[0][4] = null // black king moved
    b[1][4] = 'k'
    expect(castlingRights(b)).toBe('Q')
  })
})
