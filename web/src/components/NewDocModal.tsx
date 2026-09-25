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
  table.header([*#*], [*Trắng*], [*Đen*]),
  [1.], [#strong[e4]], [#strong[e5]],
  [2.], [#strong[Nf3]], [#strong[Nc6]],
  [3.], [#strong[Bc4]], [#strong[Nf6]],
)
`,
  },
  {
    id: 'puzzle_book',
    name: 'Sách Bài Tập Chiến Thuật (Khổ A5 - Lưới 2x2)',
    desc: 'Tập hợp bài tập câu đố kèm độ khó, gợi ý và dải đáp án',
    icon: 'puzzle',
    content: `${CHESSBOOK_IMPORT}

#show: chess-book-init.with(
  title: "100 THẾ CỜ CHIẾN THUẬT KINH ĐIỂN",
  subtitle: "Sách bài tập rèn luyện tư duy cờ vua",
  author: "CLB Cờ vua Dương Sinh",
  paper-size: "a5",
)

= BÀI TẬP: ĐÒN CHIẾU BẮT ĐÔI

#grid(
  columns: (1fr, 1fr),
  gutter: 10pt,
  puzzle-card(
    "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6",
    number: 1,
    title: "Thế cờ 1",
    turn: "w",
    difficulty: 2,
    hint: "Gỡ ghim và bắt quân",
    solution: "1. dxe4 Bxc3+ 2. bxc3"
  ),
  puzzle-card(
    "r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10",
    number: 2,
    title: "Thế cờ 2",
    turn: "b",
    difficulty: 3,
    hint: "Tấn công ô f2",
    solution: "1... Qc5 2. Bxf7+ Kxf7"
  )
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
