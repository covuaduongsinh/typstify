import { useCallback, useRef, useState } from 'react'
import { useDismiss } from '../lib/useDismiss'
import { Icon, ICON_NAMES, type IconName } from './Icon'

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
    label: 'Lưới 12 bài A4 (3x4)',
    title: 'Lưới 12 bài tập cờ vua xếp 3 cột x 4 hàng vừa vặn khổ A4',
    code: `#puzzle-grid-a4(
  puzzles: (
    (fen: "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6", number: 1, title: "Đòn đánh đôi", turn: "w", difficulty: 1, hint: "Quan sát Mã e4"),
    (fen: "r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10", number: 2, title: "Tấn công f2", turn: "b", difficulty: 2, hint: "Chiếu Vua"),
    (fen: "r2qkb1r/pp2pppp/2n2n2/3p4/3P2b1/2N2N2/PPP1BPPP/R1BQK2R w KQkq - 3 7", number: 3, title: "Gỡ ghim", turn: "w", difficulty: 2, hint: "Đổi quân"),
    (fen: "r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4", number: 4, title: "Chiếu bắt Hậu", turn: "w", difficulty: 2, hint: "Nước chiếu mở"),
    (fen: "rnbqkb1r/pppp1ppp/5n2/4p3/4P3/2N5/PPPP1PPP/R1BQKBNR w KQkq - 1 3", number: 5, title: "Chiếm trung tâm", turn: "w", difficulty: 1, hint: "Phát triển quân"),
    (fen: "r1bqkb1r/pp1p1ppp/2n1pn2/2p5/2B1P3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 5", number: 6, title: "Khóa Tượng", turn: "w", difficulty: 2, hint: "Đẩy Tốt trung tâm"),
    (fen: "r1bq1rk1/pppp1ppp/2n2n2/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R w KQ - 1 6", number: 7, title: "Phòng thủ chắc", turn: "w", difficulty: 2, hint: "Nhập thành"),
    (fen: "r2qk2r/ppp2ppp/2n1bn2/3pp3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6", number: 8, title: "Đòn ghim Tượng", turn: "w", difficulty: 3, hint: "Đổi Tốt trung tâm"),
    (fen: "r1bqkb1r/pp3ppp/2np1n2/2p1p3/2B1P3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6", number: 9, title: "Tấn công f7", turn: "w", difficulty: 2, hint: "Đưa Mã lên g5"),
    (fen: "r1b1kb1r/ppppqppp/2n5/4p3/2B1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6", number: 10, title: "Phản công cánh Vua", turn: "w", difficulty: 3, hint: "Đòn mở đường"),
    (fen: "r1bqk2r/ppppbppp/2n2n2/4p3/2B1P3/3P1N2/PPP2PPP/RNBQK2R w KQkq - 1 5", number: 11, title: "Thế cờ mở", turn: "w", difficulty: 2, hint: "Đẩy Tốt d4"),
    (fen: "r1bqkb1r/pppp1ppp/8/4n3/2B1P3/8/PPPP1PPP/RNBQK2R w KQkq - 0 5", number: 12, title: "Chiếu bắt Xe", turn: "w", difficulty: 3, hint: "Chiếu Vua bắt Xe")
  )
)\n`,
  },
  {
    label: 'Lưới 6 bài 16x24 (2x3)',
    title: 'Lưới 6 bài tập cờ vua xếp 2 cột x 3 hàng chuẩn sách 16x24cm',
    code: `#puzzle-grid-16x24(
  puzzles: (
    (fen: "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6", number: 1, title: "Đòn đánh đôi", turn: "w", difficulty: 1, hint: "Quan sát Mã e4", solution: "1. dxe4 Bxc3+ 2. bxc3"),
    (fen: "r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10", number: 2, title: "Tấn công f2", turn: "b", difficulty: 2, hint: "Chiếu Vua", solution: "1... Qc5 2. Bxf7+ Kxf7"),
    (fen: "r2qkb1r/pp2pppp/2n2n2/3p4/3P2b1/2N2N2/PPP1BPPP/R1BQK2R w KQkq - 3 7", number: 3, title: "Gỡ ghim", turn: "w", difficulty: 2, hint: "Đổi quân", solution: "1. Ne5 Bxe2 2. Qxe2"),
    (fen: "r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4", number: 4, title: "Chiếu bắt Hậu", turn: "w", difficulty: 2, hint: "Nước chiếu mở", solution: "1. Bxf7+ Kxf7 2. Nxe5+"),
    (fen: "rnbqkb1r/pppp1ppp/5n2/4p3/4P3/2N5/PPPP1PPP/R1BQKBNR w KQkq - 1 3", number: 5, title: "Chiếm trung tâm", turn: "w", difficulty: 1, hint: "Phát triển quân", solution: "1. d4 exd4 2. Qxd4"),
    (fen: "r1bqkb1r/pp1p1ppp/2n1pn2/2p5/2B1P3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 5", number: 6, title: "Khóa Tượng", turn: "w", difficulty: 3, hint: "Đẩy Tốt e5", solution: "1. e5 dxe5 2. Nxe5")
  )
)\n`,
  },
  {
    label: 'Đáp án lật ngược',
    title: 'Khung đáp án in úp ngược 180 độ ở chân trang sách / bài tập',
    code: `#upside-down-solutions((
  "1": "1. dxe4 Bxc3+ 2. bxc3",
  "2": "1... Qc5 2. Bxf7+",
  "3": "1. Ne5 Bxe2 2. Qxe2",
  "4": "1. Bxf7+ Kxf7 2. Nxe5+",
  "5": "1. d4 exd4 2. Qxd4",
  "6": "1. e5 dxe5 2. Nxe5"
))\n`,
  },
  {
    label: 'Bài tập đơn lẻ',
    title: 'Khung 1 bài tập đơn (puzzle card): FEN, độ khó, gợi ý, lời giải',
    code: `#puzzle-card(
  "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6",
  number: 1,
  title: "Đòn đánh đôi",
  turn: "w",
  difficulty: 2,
  hint: "Gỡ ghim và phản công",
  solution: "1. dxe4 Bxc3+ 2. bxc3 Trắng hơn quân."
)\n`,
  },
  {
    label: 'Khai cuộc ECO',
    title: 'Tiêu đề chuyên khảo khai cuộc theo mã ECO và diễn biến mở đầu',
    code: `#eco-header(
  code: "C 58",
  name: "Phòng thủ Hai Mã",
  subname: "Biến thể Polerio - Bogoljubow",
  intro-moves: "1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6 4. Ng5 d5 5. exd5 Na5"
)\n`,
  },
  {
    label: 'Thẻ ván đấu (tạp chí)',
    title: 'Thông tin ván đấu: kỳ thủ, Elo, giải, vòng, kết quả',
    code: `#game-header(
  white: "Ding Liren",
  white-title: "GM",
  white-elo: "2728",
  white-fed: "CHN",
  black: "Gukesh D",
  black-title: "GM",
  black-elo: "2794",
  black-fed: "IND",
  event: "FIDE World Championship 2024",
  site: "Singapore",
  date: "2024.11.25",
  round: "1",
  result: "1 - 0",
  eco: "C58",
  opening: "Two Knights Defense"
)\n`,
  },
  {
    label: 'Bài giảng huấn luyện',
    title: 'Tiêu đề bài giảng: số bài, cấp độ, thời lượng, mục tiêu',
    code: `#lesson-header(
  lesson-num: 1,
  title: "ĐÒN GHIM QUÂN TRONG CHIẾN THUẬT",
  level: "Nhập môn & Cơ bản",
  duration: "60 phút",
  objective: "Học viên nắm vững định nghĩa đòn ghim và nhận biết cơ hội trong thực chiến."
)\n`,
  },
  {
    label: 'Khái niệm then chốt',
    title: 'Khung lý thuyết đóng khung nổi bật với biểu tượng',
    code: `#concept-box(title: "Khái niệm Then chốt")[
  Đòn ghim tuyệt đối là đòn ghim mà quân phía sau là Vua, quân bị ghim không được phép di chuyển theo luật.
]\n`,
  },
  {
    label: 'Hộp trích dẫn danh ngôn',
    title: 'Khung trích dẫn nhận định ván cờ hoặc câu nói nổi tiếng',
    code: `#chess-quote(author: "Garry Kasparov")[
  Cờ vua là sự thử thách của trí tuệ, nơi mỗi nước đi đều phản ánh chiều sâu tư duy chiến lược.
]\n`,
  },
  {
    label: 'Lưu ý cho HLV',
    title: 'Ghi chú nghiệp vụ sư phạm dành cho giáo viên / huấn luyện viên',
    code: `#instructor-note[
  Nhắc học sinh quan sát đường chéo trước khi quyết định di chuyển quân Tượng.
]\n`,
  },
  {
    label: 'Câu hỏi trắc nghiệm',
    title: 'Câu hỏi kiểm tra kèm các phương án lựa chọn và đáp án',
    code: `#practice-question(
  number: 1,
  question: "Đâu là nước đi tối ưu nhất cho Trắng?",
  choices: ("A. 1. Qh5+", "B. 1. Bxf7+", "C. 1. Nf3", "D. 1. d4"),
  answer: "B. 1. Bxf7+ (Chiếu Vua và bắt Hậu)"
)\n`,
  },
]

const QUICK_NAGS = NAGS.slice(0, 6)

type Menu = 'pieces' | 'nags' | 'templates' | null

const POPOVER_WIDTH: Record<Exclude<Menu, null>, number> = { pieces: 344, nags: 372, templates: 340 }

export function ChessToolbar({ onInsertText, onOpenBoard, onOpenPgn }: ChessToolbarProps) {
  const [menu, setMenu] = useState<Menu>(null)
  const [anchor, setAnchor] = useState<{ left: number; top: number }>({ left: 0, top: 0 })
  const barRef = useRef<HTMLDivElement>(null)
  const close = useCallback(() => setMenu(null), [])
  useDismiss(barRef, menu !== null, close)

  const toggle = (m: Exclude<Menu, null>, trigger: HTMLElement) => {
    if (menu === m) {
      setMenu(null)
      return
    }
    const r = trigger.getBoundingClientRect()
    setAnchor({
      left: Math.max(8, Math.min(r.left, window.innerWidth - POPOVER_WIDTH[m] - 8)),
      top: r.bottom + 6,
    })
    setMenu(m)
  }
  const popoverStyle = { left: anchor.left, top: anchor.top }
  const insert = (code: string) => {
    onInsertText(code)
    setMenu(null)
  }

  const trigger = (m: Exclude<Menu, null>, icon: IconName | string, label: string) => (
    <button
      className={`chess-tool-btn${menu === m ? ' active' : ''}`}
      aria-label={label}
      title={label}
      aria-haspopup="menu"
      aria-expanded={menu === m}
      onClick={(e) => toggle(m, e.currentTarget)}
    >
      {icon in ICON_NAMES ? <Icon name={icon as IconName} size={14} /> : <span className="tb-glyph">{icon}</span>}
      <span className="tb-label">{label}</span>
      <Icon name="chevron-down" size={12} />
    </button>
  )

  return (
    <div className="chess-toolbar" ref={barRef} role="toolbar" aria-label="Công cụ cờ vua">
      <div className="chess-toolbar-group">
        <button
          className="chess-tool-btn btn-primary"
          aria-label="Xếp bàn cờ"
          title="Mở bàn cờ trực quan để xếp thế cờ và sinh mã Typst"
          onClick={onOpenBoard}
        >
          <Icon name="board" size={14} /> <span className="tb-label">Xếp bàn cờ</span>
        </button>
        <button
          className="chess-tool-btn"
          aria-label="Nhập PGN"
          title="Nhập và chuyển đổi ván cờ từ file PGN"
          onClick={onOpenPgn}
        >
          <Icon name="scroll" size={14} /> <span className="tb-label">Nhập PGN</span>
        </button>
      </div>

      <div className="chess-toolbar-divider" />

      <div className="chess-toolbar-group popover-anchor">
        {trigger('pieces', '♞', 'Quân cờ')}
        {menu === 'pieces' && (
          <div className="chess-popover pieces-popover" role="menu" style={popoverStyle}>
            {PIECES.map((p) => (
              <button
                key={p.code}
                role="menuitem"
                className="piece-cell"
                title={`${p.title} — chèn ${p.code}`}
                onClick={() => insert(`${p.code} `)}
              >
                <span className="piece-glyph">{p.label}</span>
                <span className="piece-name">{p.title.split(' ')[0]}</span>
              </button>
            ))}
          </div>
        )}
      </div>

      <div className="chess-toolbar-group popover-anchor">
        {trigger('nags', '!?', 'Ký hiệu')}
        {menu === 'nags' && (
          <div className="chess-popover nags-popover" role="menu" style={popoverStyle}>
            {NAGS.map((n) => (
              <button key={n.code} role="menuitem" className="nag-row" onClick={() => insert(`${n.code} `)}>
                <span className="nag-glyph">{n.label}</span>
                <span className="nag-desc">{n.title.replace(/\s*\(.*\)$/, '')}</span>
              </button>
            ))}
          </div>
        )}
      </div>

      <div className="chess-toolbar-group quick-nags" aria-label="Ký hiệu nhanh">
        {QUICK_NAGS.map((n) => (
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

      <div className="chess-toolbar-group popover-anchor">
        {trigger('templates', 'template', 'Chèn mẫu')}
        {menu === 'templates' && (
          <div className="chess-popover templates-popover" role="menu" style={{ ...popoverStyle, maxHeight: '360px', overflowY: 'auto' }}>
            {TEMPLATES.map((tmpl) => (
              <button key={tmpl.label} role="menuitem" className="template-row" onClick={() => insert(tmpl.code)}>
                <span className="template-name">{tmpl.label}</span>
                <span className="template-desc">{tmpl.title}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
