import { describe, expect, it } from 'vitest'
import { boardToFen, castlingRights, EMPTY_BOARD, INITIAL_BOARD, type Piece } from './chess'

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

describe('castlingRights', () => {
  it('only grants rights with king and rook on home squares', () => {
    const b = clone(INITIAL_BOARD)
    b[7][7] = null // white h-rook gone
    b[0][4] = null // black king moved
    b[1][4] = 'k'
    expect(castlingRights(b)).toBe('Q')
  })
})
