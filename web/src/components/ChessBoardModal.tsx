import { useMemo, useState } from 'react'
import { Icon } from './Icon'
import { Modal } from './Modal'
import { typstString } from '../lib/typst'

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
  const [fenCopied, setFenCopied] = useState(false)

  const currentFen = useMemo(() => boardToFen(boardState, turn), [boardState, turn])

  const generatedTypstCode = useMemo(() => {
    switch (formatType) {
      case 'puzzle':
        return `#puzzle-card(
  "${currentFen}",
  number: ${puzzleNum},
  title: ${typstString(title)},
  turn: "${turn}",
  difficulty: ${difficulty},
  hint: ${hint.trim() ? typstString(hint) : 'none'},
  solution: ${solution.trim() ? typstString(solution) : 'none'}
)\n`
      case 'eco':
        return `#opening-diagram-box(
  "${currentFen}",
  title: ${typstString(title)},
  turn: "${turn}",
  eval-text: "± (Trắng ưu thế)",
  caption: "Thế cờ then chốt sau biến thể chính."
)\n`
      case 'magazine':
        return `#column-diagram(
  "${currentFen}",
  move-num: ${typstString(title)},
  turn: "${turn}",
  caption: "Nước đi tạo ra sự đột biến của thế trận."
)\n`
      case 'courseware':
        return `#teaching-diagram(
  "${currentFen}",
  title: ${typstString(title)},
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

  // Drag & drop: from the palette (places a new piece) or from another
  // square (moves it). Dropping a square's piece outside the board removes it.
  const DRAG_TYPE = 'application/x-chess-piece'
  const handleDrop = (e: React.DragEvent, r: number, c: number) => {
    e.preventDefault()
    const data = e.dataTransfer.getData(DRAG_TYPE)
    if (!data) return
    const newBoard = boardState.map((row) => [...row])
    if (data.startsWith('sq:')) {
      const [fr, fc] = data.slice(3).split(',').map(Number)
      if (fr === r && fc === c) return
      newBoard[r][c] = newBoard[fr][fc]
      newBoard[fr][fc] = null
    } else {
      newBoard[r][c] = data
    }
    setBoardState(newBoard)
  }
  const removeDraggedOff = (e: React.DragEvent, r: number, c: number) => {
    if (e.dataTransfer.dropEffect === 'none') {
      const newBoard = boardState.map((row) => [...row])
      newBoard[r][c] = null
      setBoardState(newBoard)
    }
  }

  const copyFen = () => {
    void navigator.clipboard?.writeText(currentFen).then(() => {
      setFenCopied(true)
      window.setTimeout(() => setFenCopied(false), 1500)
    })
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
    <Modal
      title={
        <>
          <Icon name="board" /> Bàn cờ trực quan &amp; tạo thế cờ Typst
        </>
      }
      onClose={onClose}
    >
      <div className="chess-modal-body">
        {/* Left: 8x8 Board & Palette */}
        <div className="chess-board-area">
          <div className="chess-board-grid" role="grid" aria-label="Bàn cờ">
            {(flipped ? [7, 6, 5, 4, 3, 2, 1, 0] : [0, 1, 2, 3, 4, 5, 6, 7]).map((r, ri) => (
              <div key={r} className="chess-board-row" role="row">
                {(flipped ? [7, 6, 5, 4, 3, 2, 1, 0] : [0, 1, 2, 3, 4, 5, 6, 7]).map((c, ci) => {
                  const isDark = (r + c) % 2 === 1
                  const piece = boardState[r][c]
                  const square = `${'abcdefgh'[c]}${8 - r}`
                  return (
                    <div
                      key={`${r}-${c}`}
                      role="gridcell"
                      aria-label={square}
                      className={`chess-square ${isDark ? 'dark' : 'light'}`}
                      onClick={() => handleSquareClick(r, c)}
                      onDragOver={(e) => e.preventDefault()}
                      onDrop={(e) => handleDrop(e, r, c)}
                    >
                      {ci === 0 && <span className="board-coord rank">{8 - r}</span>}
                      {ri === 7 && <span className="board-coord file">{'abcdefgh'[c]}</span>}
                      {piece && (
                        <span
                          className={`chess-piece ${piece === piece.toUpperCase() ? 'white' : 'black'}`}
                          draggable
                          onDragStart={(e) => {
                            e.dataTransfer.setData(DRAG_TYPE, `sq:${r},${c}`)
                            e.dataTransfer.effectAllowed = 'move'
                          }}
                          onDragEnd={(e) => removeDraggedOff(e, r, c)}
                        >
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
            <div className="palette-label">Chọn quân rồi bấm vào ô, hoặc kéo-thả quân lên bàn:</div>
            <div className="palette-row">
              {['K', 'Q', 'R', 'B', 'N', 'P'].map((p) => (
                <button
                  key={p}
                  className={`palette-btn ${selectedTool === p ? 'selected' : ''}`}
                  onClick={() => setSelectedTool(p)}
                  draggable
                  onDragStart={(e) => {
                    e.dataTransfer.setData(DRAG_TYPE, p)
                    e.dataTransfer.effectAllowed = 'copy'
                  }}
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
                  draggable
                  onDragStart={(e) => {
                    e.dataTransfer.setData(DRAG_TYPE, p)
                    e.dataTransfer.effectAllowed = 'copy'
                  }}
                >
                  {PIECE_SYMBOLS[p]}
                </button>
              ))}
              <button
                className={`palette-btn trash ${selectedTool === 'trash' ? 'selected' : ''}`}
                title="Xóa quân cờ"
                onClick={() => setSelectedTool('trash')}
              >
                <Icon name="x" />
              </button>
            </div>

            <div className="board-actions-row">
              <button className="small-action-btn" onClick={() => setFlipped(!flipped)}>
                <Icon name="refresh" size={13} /> Đảo góc nhìn
              </button>
              <button className="small-action-btn" onClick={handleResetBoard}>
                <Icon name="board" size={13} /> Thế ban đầu
              </button>
              <button className="small-action-btn" onClick={handleClearBoard}>
                <Icon name="x" size={13} /> Xóa bàn
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
              onChange={(e) => setFormatType(e.target.value as typeof formatType)}
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
            <button className="copy-fen-btn" onClick={copyFen}>
              <Icon name={fenCopied ? 'check' : 'file'} size={14} /> {fenCopied ? 'Đã chép FEN' : 'Sao chép FEN'}
            </button>
            <button
              className="insert-code-btn btn-primary"
              onClick={() => {
                onInsertCode(generatedTypstCode)
                onClose()
              }}
            >
              <Icon name="download" size={14} /> Chèn vào tài liệu
            </button>
          </div>
        </div>
      </div>
    </Modal>
  )
}
