// Board model for the visual position editor (ChessBoardModal).

/** 'P','N','B','R','Q','K' (white), lowercase for black, null = empty. */
export type Piece = string | null

/** Row 0 is rank 8, column 0 is file a. */
export const INITIAL_BOARD: Piece[][] = [
  ['r', 'n', 'b', 'q', 'k', 'b', 'n', 'r'],
  ['p', 'p', 'p', 'p', 'p', 'p', 'p', 'p'],
  [null, null, null, null, null, null, null, null],
  [null, null, null, null, null, null, null, null],
  [null, null, null, null, null, null, null, null],
  [null, null, null, null, null, null, null, null],
  ['P', 'P', 'P', 'P', 'P', 'P', 'P', 'P'],
  ['R', 'N', 'B', 'Q', 'K', 'B', 'N', 'R'],
]

export const EMPTY_BOARD: Piece[][] = Array.from({ length: 8 }, () => Array<Piece>(8).fill(null))

export const PIECE_SYMBOLS: Record<string, string> = {
  K: '♔',
  Q: '♕',
  R: '♖',
  B: '♗',
  N: '♘',
  P: '♙',
  k: '♚',
  q: '♛',
  r: '♜',
  b: '♝',
  n: '♞',
  p: '♟',
}

/** castlingRights derives KQkq from where kings and rooks stand: a right
 * is only possible with the king and that rook on their home squares. */
export function castlingRights(board: Piece[][]): string {
  let rights = ''
  if (board[7][4] === 'K') {
    if (board[7][7] === 'R') rights += 'K'
    if (board[7][0] === 'R') rights += 'Q'
  }
  if (board[0][4] === 'k') {
    if (board[0][7] === 'r') rights += 'k'
    if (board[0][0] === 'r') rights += 'q'
  }
  return rights || '-'
}

/** boardToFen serialises a position with the given side to move. */
export function boardToFen(board: Piece[][], turn: 'w' | 'b'): string {
  const rows = board.map((row) => {
    let out = ''
    let empty = 0
    for (const p of row) {
      if (!p) {
        empty++
        continue
      }
      if (empty > 0) out += empty
      empty = 0
      out += p
    }
    return empty > 0 ? out + empty : out
  })
  return `${rows.join('/')} ${turn} ${castlingRights(board)} - 0 1`
}

/** Vietnamese piece names, for accessible labels. */
export const PIECE_NAMES: Record<string, string> = {
  K: 'Vua trắng',
  Q: 'Hậu trắng',
  R: 'Xe trắng',
  B: 'Tượng trắng',
  N: 'Mã trắng',
  P: 'Tốt trắng',
  k: 'Vua đen',
  q: 'Hậu đen',
  r: 'Xe đen',
  b: 'Tượng đen',
  n: 'Mã đen',
  p: 'Tốt đen',
}
