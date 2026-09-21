import { useState, useMemo } from 'react'

interface ChessBoardModalProps {
  isOpen: boolean
  onClose: () => void
  onInsertCode: (code: string) => void
}

type Piece = string | null // 'P', 'N', 'B', 'R', 'Q', 'K', 'p', 'n', 'b', 'r', 'q', 'k', null

const INITIAL_BOARD: Piece[][] = [
  ['r', 'n', 'b', 'q', 'k', 'b', 'n', 'r'],
  ['p', 'p', 'p', 'p', 'p', 'p', 'p', 'p'],
  [null, null, null, null, null, null, null, null],
  [null, null, null, null, null, null, null, null],
  [null, null, null, null, null, null, null, null],
  [null, null, null, null, null, null, null, null],
  ['P', 'P', 'P', 'P', 'P', 'P', 'P', 'P'],
  ['R', 'N', 'B', 'Q', 'K', 'B', 'N', 'R'],
]

const PIECE_SYMBOLS: Record<string, string> = {
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

function boardToFen(board: Piece[][], turn: 'w' | 'b'): string {
  const rows: string[] = []
  for (let r = 0; r < 8; r++) {
    let empty = 0
    let rowStr = ''
    for (let c = 0; c < 8; c++) {
      const p = board[r][c]
      if (!p) {
        empty++
      } else {
        if (empty > 0) {
          rowStr += empty
          empty = 0
        }
        rowStr += p
      }
    }
    if (empty > 0) rowStr += empty
    rows.push(rowStr)
  }
  return `${rows.join('/')} ${turn} KQkq - 0 1`
}

export function ChessBoardModal({ isOpen, onClose, onInsertCode }: ChessBoardModalProps) {
  const [boardState, setBoardState] = useState<Piece[][]>(INITIAL_BOARD)
  const [selectedTool, setSelectedTool] = useState<Piece | 'trash'>('P')
  const [turn, setTurn] = useState<'w' | 'b'>('w')
  const [flipped, setFlipped] = useState(false)
  const [formatType, setFormatType] = useState<'puzzle' | 'eco' | 'magazine' | 'courseware'>('puzzle')
  const [title, setTitle] = useState('Đòn chiến thuật')
  const [puzzleNum, setPuzzleNum] = useState(1)
  const [difficulty, setDifficulty] = useState(2)
  const [hint, setHint] = useState('')
  const [solution, setSolution] = useState('')

  const currentFen = useMemo(() => boardToFen(boardState, turn), [boardState, turn])

  const generatedTypstCode = useMemo(() => {
    switch (formatType) {
      case 'puzzle':
        return `#puzzle-card(
  "${currentFen}",
  number: ${puzzleNum},
  title: "${title}",
  to-move: "${turn}",
  difficulty: ${difficulty},
  hint: "${hint}",
  solution: "${solution}"
)\n`
      case 'eco':
        return `#opening-diagram-box(
  "${currentFen}",
  title: "${title}",
  turn: "${turn}",
  eval-text: "± (Trắng ưu thế)",
  caption: "Thế cờ then chốt sau biến thể chính."
)\n`
      case 'magazine':
        return `#column-diagram(
  "${currentFen}",
  move-num: "${title}",
  turn: "${turn}",
  caption: "Nước đi tạo ra sự đột biến của thế trận."
)\n`
      case 'courseware':
        return `#teaching-diagram(
  "${currentFen}",
  title: "${title}",
  turn: "${turn}",
  size: 16pt,
  caption: "Thế cờ minh họa cho bài học chiến thuật."
)\n`
    }
  }, [formatType, currentFen, puzzleNum, title, turn, difficulty, hint, solution])

  if (!isOpen) return null

  const handleSquareClick = (r: number, c: number) => {
    const newBoard = boardState.map((row) => [...row])
    if (selectedTool === 'trash') {
      newBoard[r][c] = null
    } else {
      newBoard[r][c] = selectedTool
    }
    setBoardState(newBoard)
  }

  const handleClearBoard = () => {
    setBoardState(
      Array(8)
        .fill(null)
        .map(() => Array(8).fill(null)),
    )
  }

  const handleResetBoard = () => {
    setBoardState(INITIAL_BOARD)
    setTurn('w')
  }

  return (
    <div className="chess-modal-overlay">
      <div className="chess-modal-container">
        <div className="chess-modal-header">
          <h3>♟️ Bàn Cờ Trực Quan & Trình Tạo Thế Cờ Typst</h3>
          <button className="chess-modal-close" onClick={onClose}>
            ✕
          </button>
        </div>

        <div className="chess-modal-body">
          {/* Left: 8x8 Board & Palette */}
          <div className="chess-board-area">
            <div className="chess-board-grid">
              {(flipped ? [7, 6, 5, 4, 3, 2, 1, 0] : [0, 1, 2, 3, 4, 5, 6, 7]).map((r) => (
                <div key={r} className="chess-board-row">
                  {(flipped ? [7, 6, 5, 4, 3, 2, 1, 0] : [0, 1, 2, 3, 4, 5, 6, 7]).map((c) => {
                    const isDark = (r + c) % 2 === 1
                    const piece = boardState[r][c]
                    return (
                      <div
                        key={`${r}-${c}`}
                        className={`chess-square ${isDark ? 'dark' : 'light'}`}
                        onClick={() => handleSquareClick(r, c)}
                      >
                        {piece && (
                          <span className={`chess-piece ${piece === piece.toUpperCase() ? 'white' : 'black'}`}>
                            {PIECE_SYMBOLS[piece]}
                          </span>
                        )}
                      </div>
                    )
                  })}
                </div>
              ))}
            </div>

            {/* Piece Selection Palette */}
            <div className="chess-piece-palette">
              <div className="palette-label">Chọn quân để đặt vào ô:</div>
              <div className="palette-row">
                {['K', 'Q', 'R', 'B', 'N', 'P'].map((p) => (
                  <button
                    key={p}
                    className={`palette-btn ${selectedTool === p ? 'selected' : ''}`}
                    onClick={() => setSelectedTool(p)}
                  >
                    {PIECE_SYMBOLS[p]}
                  </button>
                ))}
              </div>
              <div className="palette-row">
                {['k', 'q', 'r', 'b', 'n', 'p'].map((p) => (
                  <button
                    key={p}
                    className={`palette-btn ${selectedTool === p ? 'selected' : ''}`}
                    onClick={() => setSelectedTool(p)}
                  >
                    {PIECE_SYMBOLS[p]}
                  </button>
                ))}
                <button
                  className={`palette-btn trash ${selectedTool === 'trash' ? 'selected' : ''}`}
                  title="Xóa quân cờ"
                  onClick={() => setSelectedTool('trash')}
                >
                  🧹
                </button>
              </div>

              <div className="board-actions-row">
                <button className="small-action-btn" onClick={() => setFlipped(!flipped)}>
                  🔄 Đảo góc nhìn
                </button>
                <button className="small-action-btn" onClick={handleResetBoard}>
                  🏁 Ván cờ đầu
                </button>
                <button className="small-action-btn" onClick={handleClearBoard}>
                  🗑️ Xóa trắng
                </button>
              </div>
            </div>
          </div>

          {/* Right: Parameters & Generated Typst Code */}
          <div className="chess-config-area">
            <div className="config-group">
              <label>Định dạng Xuất bản:</label>
              <select
                value={formatType}
                onChange={(e) => setFormatType(e.target.value as any)}
                className="chess-select"
              >
                <option value="puzzle">🧩 Dạng A: Sách Bài Tập (Puzzle Card)</option>
                <option value="eco">📖 Dạng B: Bách Khoa Khai Cuộc (ECO Diagram)</option>
                <option value="magazine">📰 Dạng C: Tạp Chí Cờ Vua (Column Diagram)</option>
                <option value="courseware">🎓 Dạng D: Giáo Trình Bài Giảng (Teaching Diagram)</option>
              </select>
            </div>

            <div className="config-grid-2">
              <div className="config-group">
                <label>Lượt đi:</label>
                <select
                  value={turn}
                  onChange={(e) => setTurn(e.target.value as 'w' | 'b')}
                  className="chess-select"
                >
                  <option value="w">⬜ Trắng đi trước (w)</option>
                  <option value="b">⬛ Đen đi trước (b)</option>
                </select>
              </div>
              {formatType === 'puzzle' && (
                <div className="config-group">
                  <label>Số thứ tự / Bài #:</label>
                  <input
                    type="number"
                    value={puzzleNum}
                    onChange={(e) => setPuzzleNum(Number(e.target.value))}
                    className="chess-input"
                  />
                </div>
              )}
            </div>

            <div className="config-group">
              <label>Tiêu đề thế cờ / Ghi chú:</label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="chess-input"
                placeholder="VD: Đòn chiếu bắt Hậu, Khai cuộc Ý..."
              />
            </div>

            {formatType === 'puzzle' && (
              <>
                <div className="config-grid-2">
                  <div className="config-group">
                    <label>Độ khó (sao):</label>
                    <select
                      value={difficulty}
                      onChange={(e) => setDifficulty(Number(e.target.value))}
                      className="chess-select"
                    >
                      <option value="1">★☆☆☆☆ (Dễ / 1 Sao)</option>
                      <option value="2">★★☆☆☆ (Trung bình / 2 Sao)</option>
                      <option value="3">★★★☆☆ (Khá / 3 Sao)</option>
                      <option value="4">★★★★☆ (Khó / 4 Sao)</option>
                      <option value="5">★★★★★ (Siêu khó / 5 Sao)</option>
                    </select>
                  </div>
                  <div className="config-group">
                    <label>Gợi ý:</label>
                    <input
                      type="text"
                      value={hint}
                      onChange={(e) => setHint(e.target.value)}
                      className="chess-input"
                      placeholder="VD: Chú ý quân Xe d1..."
                    />
                  </div>
                </div>

                <div className="config-group">
                  <label>Lời giải bài tập (Solution):</label>
                  <input
                    type="text"
                    value={solution}
                    onChange={(e) => setSolution(e.target.value)}
                    className="chess-input"
                    placeholder="VD: 1. Qh7+ Kxh7 2. Rh5# (Chiếu hết)"
                  />
                </div>
              </>
            )}

            <div className="config-group">
              <label>Mã Typst được tạo tự động:</label>
              <pre className="chess-code-preview">{generatedTypstCode}</pre>
            </div>

            <div className="chess-modal-footer">
              <button
                className="copy-fen-btn"
                onClick={() => navigator.clipboard.writeText(currentFen)}
              >
                📋 Sao chép FEN
              </button>
              <button
                className="insert-code-btn"
                onClick={() => {
                  onInsertCode(generatedTypstCode)
                  onClose()
                }}
              >
                ✨ Chèn vào Tài Liệu
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
