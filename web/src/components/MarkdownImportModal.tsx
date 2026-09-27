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

const SAMPLE_CHESS_MARKDOWN = `---
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
`

export function MarkdownImportModal({
  isOpen,
  onClose,
  onInsertCode,
  onCreateNewDoc,
  initialMarkdown,
  initialFileName,
}: MarkdownImportModalProps) {
  const [markdownText, setMarkdownText] = useState(initialMarkdown ?? SAMPLE_CHESS_MARKDOWN)
  const [template, setTemplate] = useState<'book' | 'worksheet' | 'magazine' | 'article' | 'none'>('book')
  const [title, setTitle] = useState('BÀI GIẢNG CỜ VUA')
  const [subtitle, setSubtitle] = useState('')
  const [author, setAuthor] = useState('CLB Cờ vua Dương Sinh')
  const [enableChess, setEnableChess] = useState(true)
  const [convertNags, setConvertNags] = useState(true)
  const [fileName, setFileName] = useState(initialFileName ? initialFileName.replace(/\.md$/i, '.typ') : 'bai-giang.typ')
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
    return { fenCount, pgnCount, tableCount, calloutCount }
  }, [markdownText])

  if (!isOpen) return null

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
      title="Chuyển đổi Markdown (.md) sang Typst (.typ) Cờ Vua"
      className="data-import-modal modal-wide"
      onClose={onClose}
      footer={
        <div className="data-import-footer">
          <div className="footer-stats">
            <span className="badge badge-accent">
              <Icon name="board" size={14} /> {stats.fenCount} Thế cờ
            </span>
            <span className="badge badge-accent">
              <Icon name="scroll" size={14} /> {stats.pgnCount} Ván PGN
            </span>
            <span className="badge">
              <Icon name="table" size={14} /> {stats.tableCount} Bảng
            </span>
            <span className="badge">
              <Icon name="sparkles" size={14} /> {stats.calloutCount} Hộp ghi chú
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
              Chèn vào tài liệu hiện tại
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
        {/* Left Column: Markdown Input & Upload */}
        <div className="data-import-left">
          <div className="import-source-header">
            <div className="tab-buttons">
              <button
                className={`tab-btn${activeTab === 'edit' ? ' active' : ''}`}
                onClick={() => setActiveTab('edit')}
              >
                <Icon name="file-text" size={14} /> Soạn / Dán Markdown
              </button>
              <button
                className={`tab-btn${activeTab === 'preview' ? ' active' : ''}`}
                onClick={() => setActiveTab('preview')}
              >
                <Icon name="eye" size={14} /> Xem trước Typst sinh ra
              </button>
            </div>
            <label className="file-upload-btn btn-ghost" title="Tải lên tệp .md từ máy tính">
              <Icon name="upload" size={14} /> Tải file .md
              <input type="file" accept=".md,.markdown,.txt" onChange={handleFileUpload} style={{ display: 'none' }} />
            </label>
          </div>

          {activeTab === 'edit' ? (
            <textarea
              className="data-import-textarea font-mono"
              value={markdownText}
              onChange={(e) => setMarkdownText(e.target.value)}
              placeholder="Dán nội dung Markdown (hỗ trợ ```fen, ```pgn, callout, bảng...) vào đây..."
              spellCheck={false}
            />
          ) : (
            <textarea
              className="data-import-textarea font-mono typst-preview-area"
              value={generatedTypst}
              readOnly
              spellCheck={false}
            />
          )}
        </div>

        {/* Right Column: Settings & Configuration */}
        <div className="data-import-right">
          <div className="control-group">
            <label className="control-label">Khổ in & Mẫu tài liệu</label>
            <div className="layout-picker-grid">
              <button
                type="button"
                className={`layout-card${template === 'book' ? ' selected' : ''}`}
                onClick={() => setTemplate('book')}
              >
                <div className="layout-icon">
                  <Icon name="file-text" size={20} />
                </div>
                <div className="layout-info">
                  <span className="layout-name">Sách 16x24cm</span>
                  <span className="layout-desc">Sách xuất bản, giáo trình tiêu chuẩn</span>
                </div>
              </button>
              <button
                type="button"
                className={`layout-card${template === 'worksheet' ? ' selected' : ''}`}
                onClick={() => setTemplate('worksheet')}
              >
                <div className="layout-icon">
                  <Icon name="table" size={20} />
                </div>
                <div className="layout-info">
                  <span className="layout-name">Worksheet A4</span>
                  <span className="layout-desc">Tài liệu học tập, phiếu bài tập in</span>
                </div>
              </button>
              <button
                type="button"
                className={`layout-card${template === 'magazine' ? ' selected' : ''}`}
                onClick={() => setTemplate('magazine')}
              >
                <div className="layout-icon">
                  <Icon name="scroll" size={20} />
                </div>
                <div className="layout-info">
                  <span className="layout-name">Tạp chí A4</span>
                  <span className="layout-desc">Bản tin, bài viết bình luận cờ vua</span>
                </div>
              </button>
              <button
                type="button"
                className={`layout-card${template === 'none' ? ' selected' : ''}`}
                onClick={() => setTemplate('none')}
              >
                <div className="layout-icon">
                  <Icon name="code" size={20} />
                </div>
                <div className="layout-info">
                  <span className="layout-name">Đoạn trích thuần</span>
                  <span className="layout-desc">Chỉ chuyển cú pháp, không thêm bìa</span>
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
              style={{ marginTop: 6 }}
              value={subtitle}
              onChange={(e) => setSubtitle(e.target.value)}
              placeholder="Tiêu đề phụ / Tập sách"
            />
            <input
              type="text"
              className="control-input"
              style={{ marginTop: 6 }}
              value={author}
              onChange={(e) => setAuthor(e.target.value)}
              placeholder="Tác giả"
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
              <span>Tự động nhận diện khối <code>```fen</code>, <code>```pgn</code>, <code>```puzzle</code></span>
            </label>
            <label className="checkbox-label" style={{ marginTop: 6 }}>
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
