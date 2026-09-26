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

const SAMPLE_CSV = `fen,title,turn,difficulty,hint,solution
"r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6",Đòn đánh đôi,w,1,Quan sát Mã e4,1. dxe4 Bxc3+ 2. bxc3
"r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10",Tấn công f2,b,2,Chiếu Vua,1... Qc5 2. Bxf7+ Kxf7
"r2qkb1r/pp2pppp/2n2n2/3p4/3P2b1/2N2N2/PPP1BPPP/R1BQK2R w KQkq - 3 7",Gỡ ghim,w,2,Đổi quân,1. Ne5 Bxe2 2. Qxe2
"r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4",Chiếu bắt Hậu,w,2,Nước chiếu mở,1. Bxf7+ Kxf7 2. Nxe5+
"rnbqkb1r/pppp1ppp/5n2/4p3/4P3/2N5/PPPP1PPP/R1BQKBNR w KQkq - 1 3",Chiếm trung tâm,w,1,Phát triển Mã,1. d4 exd4 2. Qxd4
"r1bqkb1r/pp1p1ppp/2n1pn2/2p5/2B1P3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 5",Khóa Tượng,w,3,Đẩy Tốt e5,1. e5 dxe5 2. Nxe5`

export function DataImportModal({
  isOpen,
  onClose,
  onInsertCode,
  onCreateNewDoc,
}: DataImportModalProps) {
  const [rawText, setRawText] = useState(SAMPLE_CSV)
  const [title, setTitle] = useState('BỘ BÀI TẬP CHIẾN THUẬT CỜ VUA')
  const [subtitle, setSubtitle] = useState('Tuyển tập thế cờ chọn lọc')
  const [author, setAuthor] = useState('CLB Cờ vua Dương Sinh')
  const [layout, setLayout] = useState<'a4-3x4' | '16x24-2x3'>('a4-3x4')
  const [upsideDown, setUpsideDown] = useState(true)
  const [appendix, setAppendix] = useState(true)
  const [fileName, setFileName] = useState('sach-bai-tap.typ')
  const [showCodePreview, setShowCodePreview] = useState(false)

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

  return (
    <Modal onClose={onClose} title="Nhập Dữ Liệu Bài Tập (Excel / CSV / JSON / FEN)">
      <div className="space-y-4 max-h-[78vh] overflow-y-auto pr-1">
        {/* File Upload / Paste Area */}
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <label className="text-xs font-semibold text-ds-text flex items-center gap-1.5">
              <Icon name="file-text" size={14} className="text-ds-brand" />
              Nội dung dữ liệu (CSV, JSON, hoặc danh sách FEN)
            </label>
            <label className="inline-flex items-center gap-1.5 text-xs text-ds-brand hover:text-ds-brand/80 cursor-pointer font-medium">
              <Icon name="file-plus" size={13} />
              Tải file từ máy (.csv, .json, .txt)
              <input
                type="file"
                accept=".csv,.json,.txt,.tsv"
                onChange={handleFileUpload}
                className="hidden"
              />
            </label>
          </div>
          <textarea
            value={rawText}
            onChange={(e) => setRawText(e.target.value)}
            rows={5}
            placeholder="Dán nội dung CSV (cột fen, title, turn, difficulty, solution) hoặc mảng JSON thế cờ..."
            className="w-full text-xs font-mono p-2.5 rounded-lg border border-ds-line bg-ds-surface/50 text-ds-text focus:outline-none focus:border-ds-brand resize-y"
          />
        </div>

        {/* Stats Badge */}
        <div className="flex items-center justify-between p-2.5 rounded-lg bg-ds-brand-soft/30 border border-ds-brand/20">
          <div className="flex items-center gap-2">
            <div className="w-7 h-7 rounded-md bg-ds-brand text-white flex items-center justify-center font-bold text-xs">
              {totalPuzzles}
            </div>
            <div>
              <div className="text-xs font-semibold text-ds-text">
                {totalPuzzles > 0
                  ? `Đã nhận diện thành công ${totalPuzzles} thế cờ`
                  : 'Chưa nhận diện được thế cờ hợp lệ'}
              </div>
              <div className="text-[11px] text-ds-muted">
                {layout === 'a4-3x4'
                  ? `Khổ A4 (12 bài/trang) ➔ Chia thành ${totalPages} trang bài tập`
                  : `Khổ 16x24cm (6 bài/trang) ➔ Chia thành ${totalPages} trang sách`}
              </div>
            </div>
          </div>
        </div>

        {/* Preview Table of Puzzles */}
        {parsedPuzzles.length > 0 && (
          <div>
            <div className="text-xs font-semibold text-ds-text mb-1 flex items-center justify-between">
              <span>Xem trước dữ liệu ({Math.min(4, parsedPuzzles.length)}/{totalPuzzles} bài đầu tiên):</span>
              <button
                type="button"
                onClick={() => setShowCodePreview(!showCodePreview)}
                className="text-xs text-ds-brand hover:underline font-medium"
              >
                {showCodePreview ? 'Ẩn mã Typst' : 'Xem trước mã Typst'}
              </button>
            </div>

            {showCodePreview ? (
              <pre className="text-[11px] font-mono p-2.5 rounded-lg border border-ds-line bg-ds-surface max-h-40 overflow-y-auto text-ds-text">
                {generatedTypst}
              </pre>
            ) : (
              <div className="border border-ds-line rounded-lg overflow-hidden max-h-40 overflow-y-auto">
                <table className="w-full text-left text-xs border-collapse">
                  <thead className="bg-ds-surface sticky top-0 border-b border-ds-line text-ds-muted">
                    <tr>
                      <th className="py-1.5 px-2">#</th>
                      <th className="py-1.5 px-2">Tiêu đề</th>
                      <th className="py-1.5 px-2">Lượt</th>
                      <th className="py-1.5 px-2">Sao</th>
                      <th className="py-1.5 px-2">Đáp án</th>
                    </tr>
                  </thead>
                  <tbody>
                    {parsedPuzzles.slice(0, 4).map((p, idx) => (
                      <tr key={idx} className="border-b border-ds-line/50 hover:bg-ds-surface/40">
                        <td className="py-1.5 px-2 font-bold text-ds-brand">{idx + 1}</td>
                        <td className="py-1.5 px-2 font-medium text-ds-text truncate max-w-[120px]">
                          {p.title}
                        </td>
                        <td className="py-1.5 px-2 text-ds-muted">{p.turn === 'b' ? 'Đen' : 'Trắng'}</td>
                        <td className="py-1.5 px-2 text-ds-gold">{'★'.repeat(p.difficulty)}</td>
                        <td className="py-1.5 px-2 text-ds-muted truncate max-w-[150px]">
                          {p.solution || '—'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* Configuration Options */}
        <div className="grid grid-cols-2 gap-3 pt-2 border-t border-ds-line">
          <div>
            <label className="text-xs font-semibold text-ds-text mb-1 block">Khổ giấy & Bố cục</label>
            <div className="grid grid-cols-2 gap-2">
              <button
                type="button"
                onClick={() => setLayout('a4-3x4')}
                className={`py-2 px-2.5 rounded-lg border text-xs font-medium flex flex-col items-center gap-1 transition-all ${
                  layout === 'a4-3x4'
                    ? 'border-ds-brand bg-ds-brand-soft/40 text-ds-brand font-bold ring-1 ring-ds-brand'
                    : 'border-ds-line bg-ds-surface text-ds-muted hover:border-ds-brand/40'
                }`}
              >
                <Icon name="layout-grid" size={16} />
                <span>A4 (12 bài 3x4)</span>
              </button>
              <button
                type="button"
                onClick={() => setLayout('16x24-2x3')}
                className={`py-2 px-2.5 rounded-lg border text-xs font-medium flex flex-col items-center gap-1 transition-all ${
                  layout === '16x24-2x3'
                    ? 'border-ds-brand bg-ds-brand-soft/40 text-ds-brand font-bold ring-1 ring-ds-brand'
                    : 'border-ds-line bg-ds-surface text-ds-muted hover:border-ds-brand/40'
                }`}
              >
                <Icon name="book-open" size={16} />
                <span>16x24cm (6 bài 2x3)</span>
              </button>
            </div>
          </div>

          <div>
            <label className="text-xs font-semibold text-ds-text mb-1 block">Tùy chọn xuất bản</label>
            <div className="space-y-1.5 pt-0.5">
              <label className="flex items-center gap-2 text-xs text-ds-text cursor-pointer">
                <input
                  type="checkbox"
                  checked={upsideDown}
                  onChange={(e) => setUpsideDown(e.target.checked)}
                  className="rounded text-ds-brand focus:ring-ds-brand"
                />
                In đáp án lật ngược ở chân mỗi trang
              </label>
              <label className="flex items-center gap-2 text-xs text-ds-text cursor-pointer">
                <input
                  type="checkbox"
                  checked={appendix}
                  onChange={(e) => setAppendix(e.target.checked)}
                  className="rounded text-ds-brand focus:ring-ds-brand"
                />
                Gom toàn bộ đáp án ra phụ lục cuối sách
              </label>
            </div>
          </div>
        </div>

        {/* Metadata Inputs */}
        <div className="grid grid-cols-3 gap-3 pt-2 border-t border-ds-line">
          <div>
            <label className="text-xs font-semibold text-ds-text mb-1 block">Tiêu đề tài liệu</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full text-xs p-2 rounded-lg border border-ds-line bg-ds-surface text-ds-text focus:outline-none focus:border-ds-brand"
            />
          </div>
          <div>
            <label className="text-xs font-semibold text-ds-text mb-1 block">Phụ đề / Chủ đề</label>
            <input
              type="text"
              value={subtitle}
              onChange={(e) => setSubtitle(e.target.value)}
              className="w-full text-xs p-2 rounded-lg border border-ds-line bg-ds-surface text-ds-text focus:outline-none focus:border-ds-brand"
            />
          </div>
          <div>
            <label className="text-xs font-semibold text-ds-text mb-1 block">Tác giả / CLB</label>
            <input
              type="text"
              value={author}
              onChange={(e) => setAuthor(e.target.value)}
              className="w-full text-xs p-2 rounded-lg border border-ds-line bg-ds-surface text-ds-text focus:outline-none focus:border-ds-brand"
            />
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center justify-end gap-2 pt-3 border-t border-ds-line">
          <button
            type="button"
            onClick={onClose}
            className="px-3.5 py-1.5 rounded-lg border border-ds-line text-xs font-medium text-ds-muted hover:bg-ds-surface"
          >
            Hủy
          </button>
          {onCreateNewDoc && (
            <button
              type="button"
              onClick={handleCreateFile}
              disabled={totalPuzzles === 0}
              className="px-3.5 py-1.5 rounded-lg border border-ds-brand text-xs font-medium text-ds-brand hover:bg-ds-brand-soft/40 disabled:opacity-50"
            >
              Tạo file mới
            </button>
          )}
          <button
            type="button"
            onClick={handleInsert}
            disabled={totalPuzzles === 0}
            className="px-4 py-1.5 rounded-lg bg-ds-brand text-xs font-semibold text-white hover:bg-ds-brand/90 disabled:opacity-50 shadow-sm flex items-center gap-1.5"
          >
            <Icon name="check" size={14} />
            Chèn vào tài liệu hiện tại
          </button>
        </div>
      </div>
    </Modal>
  )
}
