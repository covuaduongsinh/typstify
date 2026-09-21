import { useState } from 'react'

interface ChessToolbarProps {
  onInsertText: (text: string) => void
  onOpenBoard: () => void
  onOpenPgn: () => void
}

const PIECES = [
  { label: '♔', code: '#wK', title: 'Vua Trắng' },
  { label: '♕', code: '#wQ', title: 'Hậu Trắng' },
  { label: '♖', code: '#wR', title: 'Xe Trắng' },
  { label: '♗', code: '#wB', title: 'Tượng Trắng' },
  { label: '♘', code: '#wN', title: 'Mã Trắng' },
  { label: '♙', code: '#wP', title: 'Tốt Trắng' },
  { label: '♚', code: '#bK', title: 'Vua Đen' },
  { label: '♛', code: '#bQ', title: 'Hậu Đen' },
  { label: '♜', code: '#bR', title: 'Xe Đen' },
  { label: '♝', code: '#bB', title: 'Tượng Đen' },
  { label: '♞', code: '#bN', title: 'Mã Đen' },
  { label: '♟', code: '#bP', title: 'Tốt Đen' },
]

const NAGS = [
  { label: '!', code: '!', title: 'Nước cờ hay' },
  { label: '?', code: '?', title: 'Nước cờ yếu' },
  { label: '!!', code: '!!', title: 'Nước cờ xuất sắc' },
  { label: '??', code: '??', title: 'Đại sai lầm' },
  { label: '!?', code: '!?', title: 'Nước cờ sắc bén' },
  { label: '?!', code: '?!', title: 'Nước cờ đáng ngờ' },
  { label: '±', code: '#nag("16")', title: 'Trắng ưu thế (±)' },
  { label: '∓', code: '#nag("17")', title: 'Đen ưu thế (∓)' },
  { label: '⩲', code: '#nag("14")', title: 'Trắng hơi ưu thế (⩲)' },
  { label: '⩱', code: '#nag("15")', title: 'Đen hơi ưu thế (⩱)' },
  { label: '=', code: '#nag("10")', title: 'Cân bằng (=)' },
  { label: '∞', code: '#nag("13")', title: 'Không rõ ràng (∞)' },
  { label: '□', code: '#nag("7")', title: 'Nước duy nhất (□)' },
]

const TEMPLATES = [
  {
    label: '🧩 Bài tập (A)',
    title: 'Chèn khung bài tập cờ vua A5',
    code: `#puzzle-card(
  "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6",
  number: 1,
  title: "Đòn đánh đôi",
  to-move: "w",
  difficulty: 2,
  hint: "Gỡ ghim và phản công",
  solution: "1. dxe4 Bxc3+ 2. bxc3 Trắng hơn quân."
)\n`,
  },
  {
    label: '📖 Khai cuộc ECO (B)',
    title: 'Chèn tiêu đề & Bảng khai cuộc ECO',
    code: `#eco-header(
  code: "C 58",
  name: "Phòng thủ Hai Mã",
  subname: "Biến thể Polerio - Bogoljubow",
  intro-moves: "1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6 4. Ng5 d5 5. exd5 Na5"
)\n`,
  },
  {
    label: '📰 Tạp chí / Thẻ ván (C)',
    title: 'Chèn Thẻ ván cờ danh thủ',
    code: `#game-header(
  white: "Magnus Carlsen",
  white-title: "GM",
  white-elo: "2835",
  white-fed: "NOR",
  black: "Hikaru Nakamura",
  black-title: "GM",
  black-elo: "2802",
  black-fed: "USA",
  event: "FIDE Candidates Tournament",
  site: "Toronto",
  date: "2024.04.15",
  round: "10",
  result: "1 - 0",
  eco: "C58",
  opening: "Phòng thủ Hai Mã"
)\n`,
  },
  {
    label: '🎓 Giáo trình / Bài giảng (D)',
    title: 'Chèn tiêu đề Bài giảng huấn luyện',
    code: `#lesson-header(
  lesson-num: 1,
  title: "ĐÒN GHIM QUÂN TRONG CHIẾN THUẬT",
  level: "Nhập môn & Cơ bản",
  duration: "60 phút",
  objective: "Học viên nắm vững định nghĩa đòn ghim và nhận biết cơ hội trong thực chiến."
)\n`,
  },
]

export function ChessToolbar({ onInsertText, onOpenBoard, onOpenPgn }: ChessToolbarProps) {
  const [showTemplates, setShowTemplates] = useState(false)

  return (
    <div className="chess-toolbar">
      <div className="chess-toolbar-group">
        <button
          className="chess-tool-btn primary"
          title="Mở Bàn cờ trực quan để xếp thế cờ và sinh mã Typst"
          onClick={onOpenBoard}
        >
          ♟️ Xếp Bàn Cờ
        </button>
        <button
          className="chess-tool-btn"
          title="Nhập và chuyển đổi ván cờ từ file PGN"
          onClick={onOpenPgn}
        >
          📜 Nhập PGN
        </button>
      </div>

      <div className="chess-toolbar-divider" />

      {/* Piece Palette */}
      <div className="chess-toolbar-group pieces-palette">
        {PIECES.map((p) => (
          <button
            key={p.code}
            className="chess-sym-btn"
            title={`${p.title} (${p.code})`}
            onClick={() => onInsertText(`${p.code} `)}
          >
            {p.label}
          </button>
        ))}
      </div>

      <div className="chess-toolbar-divider" />

      {/* NAG Symbols Palette */}
      <div className="chess-toolbar-group nag-palette">
        {NAGS.map((n) => (
          <button
            key={n.code}
            className="chess-sym-btn nag"
            title={`${n.title} (${n.code})`}
            onClick={() => onInsertText(`${n.code} `)}
          >
            {n.label}
          </button>
        ))}
      </div>

      <div className="chess-toolbar-divider" />

      {/* Quick Template Dropdown */}
      <div className="chess-toolbar-group templates-dropdown-wrap">
        <button
          className="chess-tool-btn template-toggle"
          onClick={() => setShowTemplates(!showTemplates)}
        >
          📐 Chèn Mẫu ▾
        </button>
        {showTemplates && (
          <div className="chess-templates-menu" onMouseLeave={() => setShowTemplates(false)}>
            {TEMPLATES.map((tmpl) => (
              <button
                key={tmpl.label}
                className="chess-template-item"
                title={tmpl.title}
                onClick={() => {
                  onInsertText(tmpl.code)
                  setShowTemplates(false)
                }}
              >
                {tmpl.label}
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
