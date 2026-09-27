import { useMemo, useState } from 'react'
import {
  generatePuzzleCollectionTypst,
  parseCsvPuzzles,
  parseFenList,
  parseJsonPuzzles,
  type ParsedPuzzle,
} from '../lib/dataImport'
import { Icon } from './Icon'
import { Modal } from './Modal'

interface DataImportModalProps {
  isOpen: boolean
  onClose: () => void
  onInsertCode: (code: string) => void
  onCreateNewDoc?: (fileName: string, content: string) => void
}

const PRESET_DATA = {
  csvTactics: `fen,title,turn,difficulty,hint,solution
"r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6",Đòn đánh đôi,w,1,Quan sát Mã e4,1. dxe4 Bxc3+ 2. bxc3
"r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10",Tấn công f2,b,2,Chiếu Vua,1... Qc5 2. Bxf7+ Kxf7
"r2qkb1r/pp2pppp/2n2n2/3p4/3P2b1/2N2N2/PPP1BPPP/R1BQK2R w KQkq - 3 7",Gỡ ghim,w,2,Đổi quân,1. Ne5 Bxe2 2. Qxe2
"r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4",Chiếu bắt Hậu,w,2,Nước chiếu mở,1. Bxf7+ Kxf7 2. Nxe5+
"rnbqkb1r/pppp1ppp/5n2/4p3/4P3/2N5/PPPP1PPP/R1BQKBNR w KQkq - 1 3",Chiếm trung tâm,w,1,Phát triển Mã,1. d4 exd4 2. Qxd4
"r1bqkb1r/pp1p1ppp/2n1pn2/2p5/2B1P3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 5",Khóa Tượng,w,3,Đẩy Tốt e5,1. e5 dxe5 2. Nxe5`,

  csvMateIn2: `fen,title,turn,difficulty,hint,solution
"r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4",Chiếu hết sau 2 nước #1,w,2,Thí Tượng f7,1. Bxf7+ Ke7 2. d4#
"r2qkb1r/pp2pppp/2n2n2/3p4/3P2b1/2N2N2/PPP1BPPP/R1BQK2R w KQkq - 3 7",Chiếu hết sau 2 nước #2,w,3,Tấn công cánh Vua,1. Ne5 Bxe2 2. Qxe2#
"6k1/5ppp/8/8/8/8/5PPP/4R1K1 w - - 0 1",Chiếu hết hàng ngang cuối,w,1,Xe xuống e8,1. Re8#
"r1b2rk1/pppp1ppp/8/8/1B6/8/PPP2PPP/R3R1K1 w - - 0 1",Tận dụng ghim Xe,w,2,Xe bắt f8,1. Bxf8 Kxf8 2. Re8#`,

  jsonPuzzles: `[
  {
    "fen": "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6",
    "title": "Đòn Ghim Tượng",
    "turn": "w",
    "difficulty": 1,
    "hint": "Chú ý đường chéo",
    "solution": "1. dxe4 Bxc3+ 2. bxc3"
  },
  {
    "fen": "r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4",
    "title": "Bắt Hậu bất ngờ",
    "turn": "w",
    "difficulty": 2,
    "hint": "Thí Tượng",
    "solution": "1. Bxf7+ Kxf7 2. Nxe5+"
  }
]`,

  fenList: `r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6
r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10
r2qkb1r/pp2pppp/2n2n2/3p4/3P2b1/2N2N2/PPP1BPPP/R1BQK2R w KQkq - 3 7
r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4
rnbqkb1r/pppp1ppp/5n2/4p3/4P3/2N5/PPPP1PPP/R1BQKBNR w KQkq - 1 3
r1bqkb1r/pp1p1ppp/2n1pn2/2p5/2B1P3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 5`,
}

export function DataImportModal({
  isOpen,
  onClose,
  onInsertCode,
  onCreateNewDoc,
}: DataImportModalProps) {
  const [rawText, setRawText] = useState(PRESET_DATA.csvTactics)
  const [title, setTitle] = useState('BỘ BÀI TẬP CHIẾN THUẬT CỜ VUA')
  const [subtitle, setSubtitle] = useState('Tuyển tập thế cờ chọn lọc')
  const [author, setAuthor] = useState('CLB Cờ vua Dương Sinh')
  const [layout, setLayout] = useState<'a4-3x4' | '16x24-2x3'>('a4-3x4')
  const [upsideDown, setUpsideDown] = useState(true)
  const [appendix, setAppendix] = useState(true)
  const [fileName, setFileName] = useState('sach-bai-tap.typ')
  const [activeTab, setActiveTab] = useState<'input' | 'preview'>('input')
  const [showSolutions, setShowSolutions] = useState(false)
  const [copied, setCopied] = useState(false)

  const parsedPuzzles = useMemo<ParsedPuzzle[]>(() => {
    const trimmed = rawText.trim()
    if (!trimmed) return []

    // Check JSON
    if (trimmed.startsWith('[') || trimmed.startsWith('{')) {
      const jsonRes = parseJsonPuzzles(trimmed)
      if (jsonRes.length > 0) return jsonRes
    }

    // Check CSV
    const csvRes = parseCsvPuzzles(trimmed)
    if (csvRes.length > 0) return csvRes

    // Fallback to FEN list
    return parseFenList(trimmed)
  }, [rawText])

  const totalPuzzles = parsedPuzzles.length
  const puzzlesPerPage = layout === 'a4-3x4' ? 12 : 6
  const totalPages = Math.ceil(totalPuzzles / puzzlesPerPage) || 1

  const generatedTypst = useMemo(() => {
    if (totalPuzzles === 0) return ''
    return generatePuzzleCollectionTypst(parsedPuzzles, {
      title,
      subtitle,
      author,
      layout,
      upsideDown,
      appendix,
      embedMode: 'direct',
    })
  }, [parsedPuzzles, title, subtitle, author, layout, upsideDown, appendix, totalPuzzles])

  if (!isOpen) return null

  const handleSelectPreset = (key: keyof typeof PRESET_DATA) => {
    setRawText(PRESET_DATA[key])
    if (key === 'csvTactics') {
      setTitle('BỘ BÀI TẬP CHIẾN THUẬT CỜ VUA')
      setSubtitle('Tuyển tập đòn phối hợp chọn lọc')
      setFileName('bai-tap-chien-thuat.typ')
    } else if (key === 'csvMateIn2') {
      setTitle('TUYỂN TẬP CHIẾU HẾT SAU 2 NƯỚC')
      setSubtitle('Chủ đề: Sát cục trong trung cuộc')
      setFileName('chieu-het-2-nuoc.typ')
    } else if (key === 'jsonPuzzles') {
      setTitle('BỘ ĐỀ LUYỆN TẬP CỜ TÀN')
      setSubtitle('Dữ liệu chuẩn JSON')
      setFileName('bai-tap-co-tan.typ')
    } else if (key === 'fenList') {
      setTitle('BỘ THẾ CỜ FEN HUẤN LUYỆN')
      setSubtitle('Danh sách FEN tuyển tập')
      setFileName('the-co-fen.typ')
    }
  }

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = (ev) => {
      const content = ev.target?.result as string
      if (content) {
        setRawText(content)
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

  const handleCreateFile = () => {
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
          <Icon name="table" /> Nhập Dữ Liệu Bài Tập (Excel / CSV / JSON / FEN)
        </div>
      }
      className="data-import-modal modal-wide"
      onClose={onClose}
      footer={
        <div className="data-import-footer">
          <div className="footer-stats">
            <span className="badge badge-accent">
              <Icon name="board" size={13} /> {totalPuzzles} Thế cờ
            </span>
            <span className="badge badge-gold">
              <Icon name="file-text" size={13} /> {totalPages} Trang in ({puzzlesPerPage} bài/trang)
            </span>
            <span className="badge">
              <Icon name="check" size={13} /> Khổ {layout === 'a4-3x4' ? 'A4 (3x4)' : '16x24 (2x3)'}
            </span>
          </div>
          <div className="footer-buttons">
            <button className="btn-ghost" onClick={onClose}>
              Hủy
            </button>
            <button className="btn-ghost" onClick={handleCopy} disabled={totalPuzzles === 0}>
              <Icon name={copied ? 'check' : 'copy'} />
              {copied ? 'Đã sao chép' : 'Sao chép mã'}
            </button>
            <button className="btn-ghost" onClick={handleInsert} disabled={totalPuzzles === 0}>
              <Icon name="download" />
              Chèn vào tài liệu
            </button>
            {onCreateNewDoc && (
              <button
                className="btn-primary"
                onClick={handleCreateFile}
                disabled={totalPuzzles === 0}
              >
                <Icon name="file-plus" />
                Tạo file {fileName}
              </button>
            )}
          </div>
        </div>
      }
    >
      <div className="data-import-container">
        {/* Left Column: Data Input, Presets, Preview Table */}
        <div className="data-import-left">
          <div className="import-source-header">
            <div className="tab-buttons">
              <button
                className={`tab-btn${activeTab === 'input' ? ' active' : ''}`}
                onClick={() => setActiveTab('input')}
              >
                <Icon name="file-text" size={13} /> Dữ liệu nguồn & Xem trước
              </button>
              <button
                className={`tab-btn${activeTab === 'preview' ? ' active' : ''}`}
                onClick={() => setActiveTab('preview')}
              >
                <Icon name="eye" size={13} /> Xem trước Typst sinh ra
              </button>
            </div>

            <label className="file-upload-btn" title="Tải lên tệp CSV/JSON/TXT từ máy tính">
              <Icon name="upload" size={13} /> Tải file (.csv, .json, .txt)
              <input
                type="file"
                accept=".csv,.json,.txt,.tsv"
                onChange={handleFileUpload}
                style={{ display: 'none' }}
              />
            </label>
          </div>

          {/* Quick Preset Selector */}
          <div className="preset-bar">
            <span className="preset-label">Mẫu nhanh:</span>
            <button
              type="button"
              className="preset-btn"
              onClick={() => handleSelectPreset('csvTactics')}
            >
              🎯 CSV Chiến thuật (6 bài)
            </button>
            <button
              type="button"
              className="preset-btn"
              onClick={() => handleSelectPreset('csvMateIn2')}
            >
              👑 CSV Chiếu hết 2 nước
            </button>
            <button
              type="button"
              className="preset-btn"
              onClick={() => handleSelectPreset('jsonPuzzles')}
            >
              📦 JSON Mảng bài tập
            </button>
            <button
              type="button"
              className="preset-btn"
              onClick={() => handleSelectPreset('fenList')}
            >
              ♟ FEN Thuần
            </button>
          </div>

          {activeTab === 'input' ? (
            <>
              <textarea
                className="data-import-textarea font-mono"
                value={rawText}
                onChange={(e) => setRawText(e.target.value)}
                placeholder="Dán nội dung CSV (cột fen, title, turn, difficulty, hint, solution), JSON mảng thế cờ, hoặc danh sách FEN..."
                spellCheck={false}
                style={{ minHeight: '140px', maxHeight: '200px' }}
              />

              {/* Parsed Puzzles Preview Table */}
              <div className="control-group">
                <div className="control-label">
                  <span>
                    Danh sách thế cờ đã nhận diện ({parsedPuzzles.length} bài)
                  </span>
                  {parsedPuzzles.length > 0 && (
                    <button
                      type="button"
                      className="btn-ghost"
                      style={{ fontSize: '11px', padding: '2px 6px' }}
                      onClick={() => setShowSolutions(!showSolutions)}
                    >
                      <Icon name={showSolutions ? 'eye-off' : 'eye'} size={12} />
                      {showSolutions ? 'Ẩn đáp án' : 'Hiện đáp án'}
                    </button>
                  )}
                </div>

                <div className="puzzle-preview-box">
                  {parsedPuzzles.length > 0 ? (
                    <table className="puzzle-table">
                      <thead>
                        <tr>
                          <th style={{ width: '32px' }}>#</th>
                          <th>Tiêu đề thế cờ</th>
                          <th style={{ width: '65px' }}>Lượt</th>
                          <th style={{ width: '70px' }}>Độ khó</th>
                          <th>{showSolutions ? 'Đáp án' : 'Gợi ý'}</th>
                        </tr>
                      </thead>
                      <tbody>
                        {parsedPuzzles.map((p, idx) => (
                          <tr key={idx}>
                            <td style={{ fontWeight: 700, color: 'var(--brand)' }}>{idx + 1}</td>
                            <td style={{ fontWeight: 500 }}>{p.title || `Bài tập ${idx + 1}`}</td>
                            <td>
                              <span className={`turn-badge ${p.turn === 'b' ? 'black' : 'white'}`}>
                                {p.turn === 'b' ? '⚫ Đen' : '⚪ Trắng'}
                              </span>
                            </td>
                            <td>
                              <span className="diff-stars">{'★'.repeat(p.difficulty)}</span>
                            </td>
                            <td style={{ fontSize: '11.5px', color: 'var(--text-dim)' }}>
                              {showSolutions ? (
                                <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--brand)' }}>
                                  {p.solution || '—'}
                                </span>
                              ) : (
                                <span>{p.hint || '—'}</span>
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  ) : (
                    <div style={{ padding: '20px', textAlign: 'center', color: 'var(--text-dim)', fontSize: '12px' }}>
                      Chưa tìm thấy thế cờ hợp lệ. Vui lòng dán CSV / JSON hoặc danh sách FEN.
                    </div>
                  )}
                </div>
              </div>
            </>
          ) : (
            <textarea
              className="data-import-textarea font-mono typst-preview-area"
              value={generatedTypst}
              readOnly
              placeholder="// Mã Typst sinh ra sẽ hiển thị ở đây..."
              spellCheck={false}
              style={{ minHeight: '340px' }}
            />
          )}
        </div>

        {/* Right Column: Layout & Metadata Configuration */}
        <div className="data-import-right">
          <div className="control-group">
            <label className="control-label">Khổ giấy & Bố cục lưới bài tập</label>
            <div className="layout-picker-grid">
              <button
                type="button"
                className={`layout-card${layout === 'a4-3x4' ? ' selected' : ''}`}
                onClick={() => setLayout('a4-3x4')}
              >
                <div className="layout-icon">
                  <Icon name="table" size={18} />
                </div>
                <div className="layout-info">
                  <span className="layout-name">Khổ A4 (12 bài/trang)</span>
                  <span className="layout-desc">Lưới 3 cột x 4 hàng, tự căn chỉnh trang in</span>
                </div>
              </button>

              <button
                type="button"
                className={`layout-card${layout === '16x24-2x3' ? ' selected' : ''}`}
                onClick={() => setLayout('16x24-2x3')}
              >
                <div className="layout-icon">
                  <Icon name="book" size={18} />
                </div>
                <div className="layout-info">
                  <span className="layout-name">Sách 16x24cm (6 bài/trang)</span>
                  <span className="layout-desc">Lưới 2 cột x 3 hàng, chuẩn xuất bản sách</span>
                </div>
              </button>
            </div>
          </div>

          <div className="control-group">
            <label className="control-label">Tùy chọn Lời giải & Xuất bản</label>
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={upsideDown}
                onChange={(e) => setUpsideDown(e.target.checked)}
              />
              <span>In đáp án lật ngược ở chân mỗi trang</span>
            </label>
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={appendix}
                onChange={(e) => setAppendix(e.target.checked)}
              />
              <span>Gom toàn bộ đáp án ra phụ lục cuối sách</span>
            </label>
          </div>

          <div className="control-group">
            <label className="control-label">Thông tin tài liệu bài tập</label>
            <input
              type="text"
              className="control-input"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Tiêu đề tập bài tập"
            />
            <input
              type="text"
              className="control-input"
              value={subtitle}
              onChange={(e) => setSubtitle(e.target.value)}
              placeholder="Phụ đề / Chủ đề (VD: Tuyển tập thế cờ)"
            />
            <input
              type="text"
              className="control-input"
              value={author}
              onChange={(e) => setAuthor(e.target.value)}
              placeholder="Tác giả / CLB Cờ vua"
            />
          </div>

          {onCreateNewDoc && (
            <div className="control-group">
              <label className="control-label">Tên tệp đích (.typ)</label>
              <input
                type="text"
                className="control-input font-mono"
                value={fileName}
                onChange={(e) => setFileName(e.target.value)}
                placeholder="sach-bai-tap.typ"
              />
            </div>
          )}
        </div>
      </div>
    </Modal>
  )
}
