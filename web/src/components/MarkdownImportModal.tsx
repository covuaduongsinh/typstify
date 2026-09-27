import { useMemo, useState } from 'react'
import { convertMarkdownToTypst } from '../lib/markdownToTypst'
import { Icon } from './Icon'
import { Modal } from './Modal'

interface MarkdownImportModalProps {
  isOpen: boolean
  onClose: () => void
  onInsertCode: (code: string) => void
  onCreateNewDoc?: (fileName: string, content: string) => void
  initialMarkdown?: string
  initialFileName?: string
}

const PRESET_MARKDOWNS = {
  book: `---
title: "BÍ QUYẾT CHIẾN THUẬT CỜ VUA"
subtitle: "Giáo trình Huấn luyện Nâng cao"
author: "CLB Cờ vua Dương Sinh"
layout: "book"
---

# Chương 1: Đòn Ghim & Tấn Công Đôi

Đòn ghim là một trong những vũ khí chiến thuật phổ biến và nguy hiểm nhất trong cờ vua. Khi một quân cờ bị ghim, nó bị tê liệt hoặc hạn chế khả năng di chuyển.

> [!CONCEPT] Định nghĩa Đòn Ghim Tuyệt đối
> Đòn ghim tuyệt đối xảy ra khi quân phía sau là **Vua**. Quân bị ghim hoàn toàn không được phép di chuyển theo luật cờ vua quốc tế.

## 1. Thế cờ minh họa thực chiến

Quan sát thế cờ sau trong khai cuộc Phòng thủ Hai Mã:

\`\`\`fen
r1bqk2r/pppp1ppp/2n5/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 5
title: "Ví dụ: Tượng Đen ghim Mã Trắng"
turn: b
caption: "Đen chơi 5... Bb4 ghim Mã c3 vào Vua e1"
arrows: "b4c3"
\`\`\`

## 2. Ván đấu mẫu: Kasparov vs Topalov (1999)

\`\`\`pgn
[Event "Hoogovens Group A"]
[Site "Wijk aan Zee NED"]
[Date "1999.01.20"]
[White "Kasparov, Garry"]
[Black "Topalov, Veselin"]
[Result "1-0"]
[ECO "B07"]

1. e4 d6 2. d4 Nf6 3. Nc3 g6 4. Be3 Bg7 5. Qd2 c6 6. f3 b5 1-0
\`\`\`

> "Chiến thuật là biết phải làm gì khi có việc để làm; chiến lược là biết phải làm gì khi không có việc gì để làm." -- Garry Kasparov

| # | Khai cuộc | Mã ECO | Đánh giá |
|---|-----------|--------|----------|
| 1 | Pirc Defense | B07 | Cân bằng |
| 2 | Two Knights | C58 | Sắc nét (±) |
`,

  worksheet: `---
title: "PHIẾU BÀI TẬP CHIẾN THUẬT CỜ VUA"
subtitle: "Chủ đề: Đòn Chiếu Mở & Đòn Đôi"
author: "HLV Dương Sinh"
layout: "worksheet"
---

# Phiếu Luyện Tập Số 1

Hãy quan sát kỹ từng thế cờ và tìm ra nước đi tối ưu nhất cho bên đi trước. Ghi lại lời giải chi tiết và thời gian hoàn thành.

\`\`\`fen
r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4
title: "Bài 1: Đòn chiếu bắt Hậu"
turn: w
hint: "Quan sát nước chiếu mở bằng Mã"
\`\`\`

\`\`\`fen
r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10
title: "Bài 2: Tấn công điểm yếu f2"
turn: b
hint: "Tìm nước đi phối hợp Hậu và Mã"
\`\`\`
`,

  magazine: `---
title: "TẠP CHÍ CỜ VUA & BÌNH LUẬN VÁN ĐẤU"
subtitle: "Số chuyên đề: Các ván đấu kinh điển thế kỷ 20"
author: "Ban Biên Tập CLB Dương Sinh"
layout: "magazine"
---

# Tuyệt phẩm Tấn công: Kasparov vs Topalov (1999)

Ván đấu được mệnh danh là *"Bất tử của Kasparov" (Kasparov's Immortal)* diễn ra tại giải Wijk aan Zee năm 1999.

\`\`\`pgn
[Event "Hoogovens Group A"]
[Site "Wijk aan Zee NED"]
[Date "1999.01.20"]
[Round "4"]
[White "Kasparov, Garry"]
[Black "Topalov, Veselin"]
[Result "1-0"]
[ECO "B07"]

1. e4 d6 2. d4 Nf6 3. Nc3 g6 4. Be3 Bg7 5. Qd2 c6 6. f3 b5 7. Nge2 Nbd7 8. Bh6 Bxh6 9. Qxh6 Bb7 10. a3 e5 11. O-O-O Qe7 12. Kb1 a6 13. Nc1 O-O-O 14. Nb3 exd4 15. Rxd4 c5 16. Rd1 Nb6 17. g3 Kb8 18. Na5 Ba8 19. Bh3 d5 20. Qf4+ Ka7 21. Rhe1 d4 22. Nd5 Nbxd5 23. exd5 Qd6 24. Rxd4! cxd4 25. Re7+! Kb6 26. Qxd4+ Kxa5 27. b4+ Ka4 28. Qc3 Qxd5 29. Ra7 Bb7 30. Rxb7 Qc4 31. Qxf6 Kxa3 32. Qxa6+ Kxb4 33. c3+ Kxc3 34. Qa1+ Kd2 35. Qb2+ Kd1 36. Bf1 Rd2 37. Rd7 Rxd7 38. Bxc4 bxc4 39. Qxh8 Rd3 40. Qa8 c3 41. Qa4+ Ke1 42. f4 f5 43. Kc1 Rd2 44. Qa7 1-0
\`\`\`

> [!NOTE] Bình luận then chốt
> Nước đi **24. Rxd4!** và tiếp theo là **25. Re7+!** là một trong những chuỗi thí Xe táo bạo và chuẩn xác nhất lịch sử cờ vua hiện đại.
`,

  opening: `---
title: "CẨM NANG KHAI CUỘC CỜ VUA"
subtitle: "Phòng thủ Hai Mã (Two Knights Defense - C58)"
author: "CLB Cờ vua Dương Sinh"
layout: "book"
---

# Khai cuộc Ý & Phòng thủ Hai Mã

Phòng thủ Hai Mã là một trong những khai cuộc sắc nét và giàu tính chiến thuật nhất sau các nước đi ban đầu 1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6.

## Thế cờ then chốt: Biến thể 4. Ng5 d5

\`\`\`fen
r1bqkb1r/ppp2ppp/2n5/3Pp1N1/2B5/8/PPPP1PPP/RNBQK2R b KQkq - 0 5
title: "Biến thể chính: 5. exd5 Na5"
turn: b
caption: "Đen phản công vào Tượng c4 bằng 5... Na5"
arrows: "c6a5, g5f7"
\`\`\`

| # | Nước đi Trắng | Nước đi Đen | Đánh giá | Ghi chú |
|---|---------------|-------------|----------|---------|
| 1 | 1. e4 | e5 | = | Khai cuộc mở |
| 2 | 2. Nf3 | Nc6 | = | Phát triển quân |
| 3 | 3. Bc4 | Nf6 | = | Phòng thủ Hai Mã |
| 4 | 4. Ng5 | d5 | ± | Tấn công điểm yếu f7 |
| 5 | 5. exd5 | Na5 | ⩲ | Nước đi tối ưu cho Đen |
`,
}

export function MarkdownImportModal({
  isOpen,
  onClose,
  onInsertCode,
  onCreateNewDoc,
  initialMarkdown,
  initialFileName,
}: MarkdownImportModalProps) {
  const [markdownText, setMarkdownText] = useState(initialMarkdown ?? PRESET_MARKDOWNS.book)
  const [template, setTemplate] = useState<'book' | 'worksheet' | 'magazine' | 'article' | 'none'>('book')
  const [title, setTitle] = useState('BÀI GIẢNG CỜ VUA')
  const [subtitle, setSubtitle] = useState('Giáo trình Huấn luyện Nâng cao')
  const [author, setAuthor] = useState('CLB Cờ vua Dương Sinh')
  const [enableChess, setEnableChess] = useState(true)
  const [convertNags, setConvertNags] = useState(true)
  const [fileName, setFileName] = useState(
    initialFileName ? initialFileName.replace(/\.md$/i, '.typ') : 'bai-giang.typ'
  )
  const [copied, setCopied] = useState(false)
  const [activeTab, setActiveTab] = useState<'edit' | 'preview'>('edit')

  const generatedTypst = useMemo(() => {
    if (!markdownText.trim()) return ''
    return convertMarkdownToTypst(markdownText, {
      template,
      title: title || undefined,
      subtitle: subtitle || undefined,
      author: author || undefined,
      enableChessFeatures: enableChess,
      convertNags,
    })
  }, [markdownText, template, title, subtitle, author, enableChess, convertNags])

  // Count detected elements
  const stats = useMemo(() => {
    const fenCount = (markdownText.match(/```(?:fen|chess-fen|diagram|board)/gi) || []).length
    const pgnCount = (markdownText.match(/```(?:pgn|chess-pgn|chess)/gi) || []).length
    const tableCount = (markdownText.match(/\|[\s:-]+\|/g) || []).length
    const calloutCount = (markdownText.match(/>\s*\[!(?:NOTE|CONCEPT|IMPORTANT|WARNING|QUOTE)/gi) || []).length
    const nagCount = (markdownText.match(/(?:\$\d+|[!?]{1,2}|⩲|±|∓|⩱|⨁)/g) || []).length
    return { fenCount, pgnCount, tableCount, calloutCount, nagCount }
  }, [markdownText])

  if (!isOpen) return null

  const handleSelectPreset = (key: keyof typeof PRESET_MARKDOWNS) => {
    const text = PRESET_MARKDOWNS[key]
    setMarkdownText(text)
    if (key === 'book') {
      setTemplate('book')
      setTitle('BÍ QUYẾT CHIẾN THUẬT CỜ VUA')
      setSubtitle('Giáo trình Huấn luyện Nâng cao')
      setFileName('giao-trinh-chien-thuat.typ')
    } else if (key === 'worksheet') {
      setTemplate('worksheet')
      setTitle('PHIẾU BÀI TẬP CHIẾN THUẬT CỜ VUA')
      setSubtitle('Chủ đề: Đòn Chiếu Mở & Đòn Đôi')
      setFileName('phieu-bai-tap.typ')
    } else if (key === 'magazine') {
      setTemplate('magazine')
      setTitle('TẠP CHÍ CỜ VUA & BÌNH LUẬN')
      setSubtitle('Ván đấu kinh điển thế kỷ 20')
      setFileName('tap-chi-co-vua.typ')
    } else if (key === 'opening') {
      setTemplate('book')
      setTitle('CẨM NANG KHAI CUỘC CỜ VUA')
      setSubtitle('Phòng thủ Hai Mã (C58)')
      setFileName('khai-cuoc-c58.typ')
    }
  }

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = (ev) => {
      const content = ev.target?.result as string
      if (content) {
        setMarkdownText(content)
        const baseName = file.name.replace(/\.[^/.]+$/, '')
        setTitle(baseName.toUpperCase().replace(/[-_]/g, ' '))
        setFileName(`${baseName}.typ`)
      }
    }
    reader.readAsText(file)
  }

  const handleInsert = () => {
    if (generatedTypst) {
      onInsertCode(generatedTypst)
      onClose()
    }
  }

  const handleCreateDoc = () => {
    if (generatedTypst && onCreateNewDoc) {
      onCreateNewDoc(fileName, generatedTypst)
      onClose()
    }
  }

  const handleCopy = async () => {
    if (generatedTypst) {
      await navigator.clipboard.writeText(generatedTypst)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  return (
    <Modal
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Icon name="file-text" /> Chuyển đổi Markdown (.md) sang Typst (.typ) Cờ Vua
        </div>
      }
      className="data-import-modal modal-wide"
      onClose={onClose}
      footer={
        <div className="data-import-footer">
          <div className="footer-stats">
            <span className="badge badge-accent">
              <Icon name="board" size={13} /> {stats.fenCount} Thế cờ
            </span>
            <span className="badge badge-accent">
              <Icon name="scroll" size={13} /> {stats.pgnCount} Ván PGN
            </span>
            <span className="badge">
              <Icon name="table" size={13} /> {stats.tableCount} Bảng
            </span>
            <span className="badge">
              <Icon name="sparkles" size={13} /> {stats.calloutCount} Hộp ghi chú
            </span>
          </div>
          <div className="footer-buttons">
            <button className="btn-ghost" onClick={onClose}>
              Hủy
            </button>
            <button className="btn-ghost" onClick={handleCopy} disabled={!generatedTypst}>
              <Icon name={copied ? 'check' : 'copy'} />
              {copied ? 'Đã sao chép' : 'Sao chép mã'}
            </button>
            <button className="btn-ghost" onClick={handleInsert} disabled={!generatedTypst}>
              <Icon name="download" />
              Chèn vào tài liệu
            </button>
            {onCreateNewDoc && (
              <button className="btn-primary" onClick={handleCreateDoc} disabled={!generatedTypst}>
                <Icon name="file-plus" />
                Tạo file {fileName}
              </button>
            )}
          </div>
        </div>
      }
    >
      <div className="data-import-container">
        {/* Left Column: Markdown Input, Presets & Upload */}
        <div className="data-import-left">
          <div className="import-source-header">
            <div className="tab-buttons">
              <button
                className={`tab-btn${activeTab === 'edit' ? ' active' : ''}`}
                onClick={() => setActiveTab('edit')}
              >
                <Icon name="file-text" size={13} /> Soạn / Dán Markdown
              </button>
              <button
                className={`tab-btn${activeTab === 'preview' ? ' active' : ''}`}
                onClick={() => setActiveTab('preview')}
              >
                <Icon name="eye" size={13} /> Xem trước Typst sinh ra
              </button>
            </div>
            <label className="file-upload-btn" title="Tải lên tệp .md từ máy tính">
              <Icon name="upload" size={13} /> Tải file .md
              <input type="file" accept=".md,.markdown,.txt" onChange={handleFileUpload} style={{ display: 'none' }} />
            </label>
          </div>

          {/* Quick Preset Selector */}
          <div className="preset-bar">
            <span className="preset-label">Mẫu nhanh:</span>
            <button
              type="button"
              className="preset-btn"
              onClick={() => handleSelectPreset('book')}
            >
              📖 Giáo trình
            </button>
            <button
              type="button"
              className="preset-btn"
              onClick={() => handleSelectPreset('worksheet')}
            >
              📝 Phiếu bài tập
            </button>
            <button
              type="button"
              className="preset-btn"
              onClick={() => handleSelectPreset('magazine')}
            >
              📰 Tạp chí ván đấu
            </button>
            <button
              type="button"
              className="preset-btn"
              onClick={() => handleSelectPreset('opening')}
            >
              🏛 Khai cuộc ECO
            </button>
          </div>

          {activeTab === 'edit' ? (
            <textarea
              className="data-import-textarea font-mono"
              value={markdownText}
              onChange={(e) => setMarkdownText(e.target.value)}
              placeholder="Dán nội dung Markdown (hỗ trợ ```fen, ```pgn, callout [!NOTE], bảng markdown, v.v.)..."
              spellCheck={false}
            />
          ) : (
            <textarea
              className="data-import-textarea font-mono typst-preview-area"
              value={generatedTypst}
              readOnly
              placeholder="// Mã Typst sinh ra sẽ hiển thị ở đây..."
              spellCheck={false}
            />
          )}
        </div>

        {/* Right Column: Settings & Configuration */}
        <div className="data-import-right">
          <div className="control-group">
            <label className="control-label">Khổ in & Mẫu tài liệu Typst</label>
            <div className="layout-picker-grid">
              <button
                type="button"
                className={`layout-card${template === 'book' ? ' selected' : ''}`}
                onClick={() => setTemplate('book')}
              >
                <div className="layout-icon">
                  <Icon name="book" size={18} />
                </div>
                <div className="layout-info">
                  <span className="layout-name">Sách 16x24cm</span>
                  <span className="layout-desc">Sách xuất bản, giáo trình chuẩn</span>
                </div>
              </button>

              <button
                type="button"
                className={`layout-card${template === 'worksheet' ? ' selected' : ''}`}
                onClick={() => setTemplate('worksheet')}
              >
                <div className="layout-icon">
                  <Icon name="table" size={18} />
                </div>
                <div className="layout-info">
                  <span className="layout-name">Worksheet A4</span>
                  <span className="layout-desc">Phiếu bài tập, tài liệu học</span>
                </div>
              </button>

              <button
                type="button"
                className={`layout-card${template === 'magazine' ? ' selected' : ''}`}
                onClick={() => setTemplate('magazine')}
              >
                <div className="layout-icon">
                  <Icon name="scroll" size={18} />
                </div>
                <div className="layout-info">
                  <span className="layout-name">Tạp chí A4</span>
                  <span className="layout-desc">Bản tin, bình luận cờ vua</span>
                </div>
              </button>

              <button
                type="button"
                className={`layout-card${template === 'none' ? ' selected' : ''}`}
                onClick={() => setTemplate('none')}
              >
                <div className="layout-icon">
                  <Icon name="code" size={18} />
                </div>
                <div className="layout-info">
                  <span className="layout-name">Đoạn trích thuần</span>
                  <span className="layout-desc">Chỉ đổi cú pháp, không thêm bìa</span>
                </div>
              </button>
            </div>
          </div>

          <div className="control-group">
            <label className="control-label">Thông tin tài liệu</label>
            <input
              type="text"
              className="control-input"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Tiêu đề sách / bài viết"
            />
            <input
              type="text"
              className="control-input"
              value={subtitle}
              onChange={(e) => setSubtitle(e.target.value)}
              placeholder="Tiêu đề phụ / Tập sách / Chủ đề"
            />
            <input
              type="text"
              className="control-input"
              value={author}
              onChange={(e) => setAuthor(e.target.value)}
              placeholder="Tác giả / Câu lạc bộ"
            />
          </div>

          <div className="control-group">
            <label className="control-label">Tùy chọn Cờ vua thông minh</label>
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={enableChess}
                onChange={(e) => setEnableChess(e.target.checked)}
              />
              <span>Tự động nhận diện khối <code>```fen</code>, <code>```pgn</code></span>
            </label>
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={convertNags}
                onChange={(e) => setConvertNags(e.target.checked)}
              />
              <span>Chuyển đổi ký hiệu NAG ($14, ±, ⩲, !, ?) sang <code>#nag(...)</code></span>
            </label>
          </div>

          {onCreateNewDoc && (
            <div className="control-group">
              <label className="control-label">Tên tệp đích</label>
              <input
                type="text"
                className="control-input font-mono"
                value={fileName}
                onChange={(e) => setFileName(e.target.value)}
                placeholder="ten-file.typ"
              />
            </div>
          )}
        </div>
      </div>
    </Modal>
  )
}
