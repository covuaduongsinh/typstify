import { useState } from 'react'
import { CHESSBOOK_IMPORT } from '../lib/typst'
import { Icon } from './Icon'
import { Modal } from './Modal'

export interface TemplateOption {
  id: string
  name: string
  desc: string
  icon: string
  content: string
}

export const STARTER_TEMPLATES: TemplateOption[] = [
  {
    id: 'worksheet_a4_12',
    name: 'Phiếu Bài Tập A4 (12 Bài - Lưới 3x4)',
    desc: 'Bố cục 12 bài tập cờ vua xếp 3 cột x 4 hàng vừa khít 1 trang A4, kèm dải đáp án lật ngược chân trang',
    icon: 'layout-grid',
    content: `${CHESSBOOK_IMPORT}

#show: chess-worksheet-init.with(
  title: "PHIẾU BÀI TẬP CHIẾN THUẬT CỜ VUA",
  subtitle: "Chủ đề: Đòn phối hợp khai cuộc",
  author: "CLB Cờ vua Dương Sinh",
  date: "Tháng 09/2026",
  paper-size: "a4",
)

#puzzle-grid-a4(
  puzzles: (
    (fen: "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6", number: 1, title: "Đòn đánh đôi", turn: "w", difficulty: 1, hint: "Quan sát Mã e4"),
    (fen: "r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10", number: 2, title: "Tấn công f2", turn: "b", difficulty: 2, hint: "Chiếu Vua"),
    (fen: "r2qkb1r/pp2pppp/2n2n2/3p4/3P2b1/2N2N2/PPP1BPPP/R1BQK2R w KQkq - 3 7", number: 3, title: "Gỡ ghim", turn: "w", difficulty: 2, hint: "Đổi quân"),
    (fen: "r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4", number: 4, title: "Chiếu bắt Hậu", turn: "w", difficulty: 2, hint: "Nước chiếu mở"),
    (fen: "rnbqkb1r/pppp1ppp/5n2/4p3/4P3/2N5/PPPP1PPP/R1BQKBNR w KQkq - 1 3", number: 5, title: "Chiếm trung tâm", turn: "w", difficulty: 1, hint: "Phát triển quân"),
    (fen: "r1bqkb1r/pp1p1ppp/2n1pn2/2p5/2B1P3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 5", number: 6, title: "Khóa Tượng", turn: "w", difficulty: 2, hint: "Đẩy Tốt trung tâm"),
    (fen: "r1bq1rk1/pppp1ppp/2n2n2/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R w KQ - 1 6", number: 7, title: "Phòng thủ chắc", turn: "w", difficulty: 2, hint: "Nhập thành"),
    (fen: "r2qk2r/ppp2ppp/2n1bn2/3pp3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6", number: 8, title: "Đòn ghim Tượng", turn: "w", difficulty: 3, hint: "Đổi Tốt trung tâm"),
    (fen: "r1bqkb1r/pp3ppp/2np1n2/2p1p3/2B1P3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6", number: 9, title: "Tấn công điểm f7", turn: "w", difficulty: 2, hint: "Đưa Mã lên g5"),
    (fen: "r1b1kb1r/ppppqppp/2n5/4p3/2B1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6", number: 10, title: "Phản công cánh Vua", turn: "w", difficulty: 3, hint: "Đòn mở đường"),
    (fen: "r1bqk2r/ppppbppp/2n2n2/4p3/2B1P3/3P1N2/PPP2PPP/RNBQK2R w KQkq - 1 5", number: 11, title: "Tạo thế cờ mở", turn: "w", difficulty: 2, hint: "Đẩy Tốt d4"),
    (fen: "r1bqkb1r/pppp1ppp/8/4n3/2B1P3/8/PPPP1PPP/RNBQK2R w KQkq - 0 5", number: 12, title: "Đòn chiếu bắt Xe", turn: "w", difficulty: 3, hint: "Chiếu Vua bắt Xe"),
  )
)

#upside-down-solutions((
  "1": "1. dxe4 Bxc3+ 2. bxc3",
  "2": "1... Qc5 2. Bxf7+",
  "3": "1. Ne5 Bxe2 2. Qxe2",
  "4": "1. Bxf7+ Kxf7 2. Nxe5+",
  "5": "1. d4 exd4 2. Qxd4",
  "6": "1. e5 dxe5 2. Nxe5",
  "7": "1. O-O Bxc3 2. bxc3",
  "8": "1. exd5 Nxd5 2. Nxd5",
  "9": "1. Ng5 O-O 2. Qh5",
  "10": "1. dxe4 Bxc3+ 2. Bd2",
  "11": "1. d4 exd4 2. Nxd4",
  "12": "1. Bxf7+ Kxf7 2. Nxe5+"
))
`,
  },
  {
    id: 'puzzle_16x24_6',
    name: 'Sách Bài Tập 16x24cm (6 Bài/Trang - Lưới 2x3)',
    desc: 'Khổ sách chuẩn NXB Việt Nam (16x24cm): 6 bài tập 2 cột x 3 hàng, lề đóng gáy sách và đáp án chân trang',
    icon: 'book-open',
    content: `${CHESSBOOK_IMPORT}

#show: chess-book-init.with(
  title: "100 THẾ CỜ CHIẾN THUẬT ĐỈNH CAO",
  subtitle: "Tuyển tập bài tập cờ vua thực chiến",
  author: "CLB Cờ vua Dương Sinh",
  paper-size: "16x24",
)

= Chương 1: Đòn Ghim Quân Chiến Thuật

#v(4pt)

#puzzle-grid-16x24(
  puzzles: (
    (fen: "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6", number: 1, title: "Đòn đánh đôi cơ bản", turn: "w", difficulty: 1, hint: "Quan sát Mã e4", solution: "1. dxe4 Bxc3+ 2. bxc3"),
    (fen: "r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10", number: 2, title: "Tấn công điểm f2", turn: "b", difficulty: 2, hint: "Chiếu Vua", solution: "1... Qc5 2. Bxf7+ Kxf7"),
    (fen: "r2qkb1r/pp2pppp/2n2n2/3p4/3P2b1/2N2N2/PPP1BPPP/R1BQK2R w KQkq - 3 7", number: 3, title: "Gỡ ghim chủ động", turn: "w", difficulty: 2, hint: "Đổi Tượng lấy Mã", solution: "1. Ne5 Bxe2 2. Qxe2"),
    (fen: "r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4", number: 4, title: "Chiếu bắt Hậu đen", turn: "w", difficulty: 2, hint: "Nước chiếu mở bằng Mã", solution: "1. Bxf7+ Kxf7 2. Nxe5+"),
    (fen: "rnbqkb1r/pppp1ppp/5n2/4p3/4P3/2N5/PPPP1PPP/R1BQKBNR w KQkq - 1 3", number: 5, title: "Chiếm ưu thế trung tâm", turn: "w", difficulty: 1, hint: "Phát triển quân Mã", solution: "1. d4 exd4 2. Qxd4"),
    (fen: "r1bqkb1r/pp1p1ppp/2n1pn2/2p5/2B1P3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 5", number: 6, title: "Khóa chặt Tượng", turn: "w", difficulty: 3, hint: "Đẩy Tốt trung tâm e5", solution: "1. e5 dxe5 2. Nxe5")
  )
)

#upside-down-solutions((
  "1": "1. dxe4 Bxc3+ 2. bxc3",
  "2": "1... Qc5 2. Bxf7+",
  "3": "1. Ne5 Bxe2 2. Qxe2",
  "4": "1. Bxf7+ Kxf7 2. Nxe5+",
  "5": "1. d4 exd4 2. Qxd4",
  "6": "1. e5 dxe5 2. Nxe5"
))
`,
  },
  {
    id: 'book',
    name: 'Sách Cờ Vua Chuẩn (Khổ A5)',
    desc: 'Bố cục sách chuẩn: Tiêu đề chương, bài tập, phân tích ván đấu',
    icon: 'book',
    content: `${CHESSBOOK_IMPORT}

#show: chess-book-init.with(
  title: "CẨM NANG CỜ VUA",
  subtitle: "Tuyển tập đòn phối hợp chiến thuật",
  author: "CLB Cờ vua Dương Sinh",
  paper-size: "a5",
)

= Chương 1: Đòn đánh đôi trong khai cuộc

#puzzle-card(
  "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6",
  number: 1,
  title: "Đòn đánh đôi cơ bản",
  turn: "w",
  difficulty: 1,
  hint: "Quan sát quân Mã trung tâm",
  solution: "1. dxe4 Bxc3+ 2. bxc3 Trắng hơn quân."
)

Nội dung phân tích chi tiết thế trận...
`,
  },
  {
    id: 'magazine',
    name: 'Tạp Chí Cờ Vua (Khổ A4 - 2 cột)',
    desc: 'Chuẩn phong cách New In Chess: Thẻ ván đấu, bảng nước đi 2 cột, bình luận',
    icon: 'newspaper',
    content: `${CHESSBOOK_IMPORT}

#show: chess-magazine-init.with(
  magazine-title: "TẠP CHÍ CỜ VUA VIỆT NAM",
  issue: "SỐ 01 - 2026",
)

= TIÊU ĐIỂM GIẢI ĐẤU ĐỈNH CAO

#game-header(
  white: "Ding Liren",
  white-title: "GM",
  white-elo: "2728",
  black: "Gukesh D",
  black-title: "GM",
  black-elo: "2794",
  event: "FIDE World Championship 2024",
  result: "1 - 0",
  eco: "C58"
)

#v(6pt)

#table(
  columns: (22pt, 1fr, 1fr),
  stroke: none,
  table.header([*\\#*], [*Trắng*], [*Đen*]),
  [1.], [#strong[e4]], [#strong[e5]],
  [2.], [#strong[Nf3]], [#strong[Nc6]],
  [3.], [#strong[Bc4]], [#strong[Nf6]],
)
`,
  },
  {
    id: 'courseware',
    name: 'Giáo Trình Giảng Dạy HLV (Khổ A4)',
    desc: 'Giáo án huấn luyện: Khung mục tiêu bài học, lý thuyết, diagram khổ lớn, câu hỏi',
    icon: 'graduation-cap',
    content: `${CHESSBOOK_IMPORT}

#lesson-header(
  lesson-num: 1,
  title: "ĐÒN GHIM QUÂN TRONG CHIẾN THUẬT",
  level: "Trình độ: Cơ bản",
  duration: "Thời lượng: 60 phút",
  objective: "Học viên hiểu rõ đòn ghim tuyệt đối và tương đối, nhận biết cơ hội trong thực chiến."
)

#v(10pt)

#concept-box(title: "Khái niệm Then chốt")[
  Đòn ghim là chiến thuật lợi dụng đường thẳng hoặc đường chéo để khóa chặt quân đối phương.
]

#v(10pt)

#teaching-diagram(
  "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6",
  title: "Ví dụ minh họa: Đòn ghim Mã",
  turn: "w",
  caption: "Quân Mã c3 bị ghim vào Vua trắng."
)

#v(10pt)

#practice-question(
  number: 1,
  question: "Quân cờ nào có thể thực hiện đòn ghim theo đường chéo?",
  choices: ("A. Xe", "B. Tượng và Hậu", "C. Mã", "D. Vua"),
  answer: "B. Tượng và Hậu"
)
`,
  },
  {
    id: 'puzzle_collection',
    name: 'Sách Tuyển Tập Bài Tập (CSDL JSON / CSV)',
    desc: 'Tự động phân trang hàng chục/hàng trăm bài tập (khổ A4 hoặc 16x24cm), có đáp án lật ngược và phụ lục cuối sách',
    icon: 'table',
    content: `${CHESSBOOK_IMPORT}

#show: chess-book-init.with(
  title: "TUYỂN TẬP BÀI TẬP CỜ VUA",
  subtitle: "Tự động phân trang & xuất đáp án",
  author: "CLB Cờ vua Dương Sinh",
  paper-size: "16x24",
)

// Đọc dữ liệu từ file JSON hoặc CSV
// #let puzzle-data = json("puzzles.json")
// #let puzzle-data = csv-to-puzzles(csv("puzzles.csv"))

#let puzzle-data = (
  (fen: "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6", title: "Đòn đánh đôi", turn: "w", hint: "Mã e4", solution: "1. dxe4 Bxc3+ 2. bxc3"),
  (fen: "r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10", title: "Tấn công f2", turn: "b", hint: "Chiếu Vua", solution: "1... Qc5 2. Bxf7+"),
  (fen: "r2qkb1r/pp2pppp/2n2n2/3p4/3P2b1/2N2N2/PPP1BPPP/R1BQK2R w KQkq - 3 7", title: "Gỡ ghim", turn: "w", hint: "Đổi quân", solution: "1. Ne5 Bxe2 2. Qxe2"),
  (fen: "r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4", title: "Chiếu bắt Hậu", turn: "w", hint: "Chiếu mở", solution: "1. Bxf7+ Kxf7 2. Nxe5+"),
  (fen: "rnbqkb1r/pppp1ppp/5n2/4p3/4P3/2N5/PPPP1PPP/R1BQKBNR w KQkq - 1 3", title: "Trung tâm", turn: "w", hint: "Phát triển", solution: "1. d4 exd4 2. Qxd4"),
  (fen: "r1bqkb1r/pp1p1ppp/2n1pn2/2p5/2B1P3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 5", title: "Khóa Tượng", turn: "w", hint: "Đẩy Tốt", solution: "1. e5 dxe5 2. Nxe5"),
)

#render-puzzle-collection(
  puzzle-data,
  layout: "16x24-2x3",
  title-prefix: "Chương 1: Rèn luyện chiến thuật",
  show-page-solutions: true,
  show-end-appendix: true,
)
`,
  },
  {
    id: 'blank',
    name: 'Tài Liệu Typst Tối Giản',
    desc: 'Tài liệu trắng chỉ kèm khai báo thư viện cờ vua',
    icon: 'file-text',
    content: `${CHESSBOOK_IMPORT}

= Tiêu đề tài liệu

Bắt đầu viết nội dung của bạn tại đây...
`,
  },
]

export function NewDocModal({
  isOpen,
  onClose,
  onCreate,
}: {
  isOpen: boolean
  onClose: () => void
  onCreate: (fileName: string, content: string) => void
}) {
  const [fileName, setFileName] = useState('chess_document.typ')
  const [selectedTemplateId, setSelectedTemplateId] = useState('book')

  if (!isOpen) return null

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    let target = fileName.trim()
    if (!target) return
    if (!target.includes('.')) target += '.typ'
    const tmpl = STARTER_TEMPLATES.find((t) => t.id === selectedTemplateId) ?? STARTER_TEMPLATES[0]
    onCreate(target, tmpl.content)
  }

  return (
    <Modal
      title={
        <>
          <Icon name="file-plus" /> Tạo tài liệu cờ vua mới
        </>
      }
      onClose={onClose}
      className="new-doc-modal"
    >
      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
        <div className="config-group">
          <label style={{ fontWeight: 600, marginBottom: '6px', display: 'block' }}>Tên file:</label>
          <input
            type="text"
            className="chess-input"
            value={fileName}
            onChange={(e) => setFileName(e.target.value)}
            placeholder="VD: chuong1.typ, sach_bai_tap.typ"
            autoFocus
          />
        </div>

        <div className="config-group">
          <label style={{ fontWeight: 600, marginBottom: '6px', display: 'block' }}>Chọn Mẫu Tài Liệu (Template):</label>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', maxHeight: '280px', overflowY: 'auto' }}>
            {STARTER_TEMPLATES.map((tmpl) => (
              <div
                key={tmpl.id}
                onClick={() => setSelectedTemplateId(tmpl.id)}
                style={{
                  padding: '10px 12px',
                  borderRadius: '6px',
                  border: `1.5px solid ${selectedTemplateId === tmpl.id ? 'var(--color-accent, #2563eb)' : 'var(--color-border, #e2e8f0)'}`,
                  background: selectedTemplateId === tmpl.id ? 'var(--color-accent-soft, #eff6ff)' : 'transparent',
                  cursor: 'pointer',
                  transition: 'all 0.15s ease',
                }}
              >
                <div style={{ fontWeight: 600, fontSize: '13px', color: 'var(--color-text, #0f172a)' }}>
                  {tmpl.name}
                </div>
                <div style={{ fontSize: '11px', opacity: 0.75, marginTop: '2px' }}>{tmpl.desc}</div>
              </div>
            ))}
          </div>
        </div>

        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', marginTop: '8px' }}>
          <button type="button" className="btn-ghost" onClick={onClose}>
            Hủy
          </button>
          <button type="submit" className="btn-primary" disabled={!fileName.trim()}>
            <Icon name="check" size={14} /> Tạo tài liệu
          </button>
        </div>
      </form>
    </Modal>
  )
}
