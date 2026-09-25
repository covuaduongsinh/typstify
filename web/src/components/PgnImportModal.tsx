import { useMemo, useState } from 'react'
import { parsePgn, pgnToTypst, type PgnLayout } from '../lib/pgn'
import { Icon } from './Icon'
import { Modal } from './Modal'

interface PgnImportModalProps {
  isOpen: boolean
  onClose: () => void
  onInsertCode: (code: string) => void
}

const SAMPLE_PGN = `[Event "FIDE World Championship 2024"]
[Site "Singapore"]
[Date "2024.11.25"]
[Round "1"]
[White "Ding Liren"]
[Black "Gukesh D"]
[Result "1-0"]
[WhiteElo "2728"]
[BlackElo "2794"]
[ECO "C58"]
[Opening "Two Knights Defense"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6 4. Ng5 d5 5. exd5 Na5 6. Bb5+ c6 7. dxc6 bxc6 8. Bd3! h6 9. Ne4 Nd5 10. O-O Nf4 11. Be2 Bf5 12. d3! Nxe2+ 13. Qxe2 Be7 14. Nbc3 O-O 15. f4! exf4 16. Bxf4 Bg6 17. Rae1 Nb7 18. Qf3 Nc5 19. Kh1 Ne6 20. Be3 f5 21. Qh3! 1-0`

export function PgnImportModal({ isOpen, onClose, onInsertCode }: PgnImportModalProps) {
  const [pgnInput, setPgnInput] = useState(SAMPLE_PGN)
  const [layout, setLayout] = useState<PgnLayout>('magazine')
  const [showPreviewCode, setShowPreviewCode] = useState(false)

  const parsedGames = useMemo(() => parsePgn(pgnInput), [pgnInput])
  const gameCount = parsedGames.length

  const generatedCode = useMemo(() => {
    if (gameCount === 0) return ''
    return pgnToTypst(pgnInput, { layout })
  }, [pgnInput, layout, gameCount])

  if (!isOpen) return null

  const handleImport = () => {
    if (generatedCode) {
      onInsertCode(generatedCode)
      onClose()
    }
  }

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      const reader = new FileReader()
      reader.onload = (ev) => {
        const text = ev.target?.result as string
        if (text) setPgnInput(text)
      }
      reader.readAsText(file)
    }
  }

  return (
    <Modal
      className="pgn-modal"
      title={
        <>
          <Icon name="scroll" /> Nhập ván cờ từ PGN
        </>
      }
      onClose={onClose}
      footer={
        <>
          <button className="small-action-btn" onClick={() => setPgnInput(SAMPLE_PGN)}>
            <Icon name="refresh" size={13} /> Nạp ván mẫu
          </button>
          <button
            className="small-action-btn"
            onClick={() => setShowPreviewCode(!showPreviewCode)}
          >
            <Icon name={showPreviewCode ? 'eye-off' : 'eye'} size={13} />{' '}
            {showPreviewCode ? 'Ẩn mã Typst' : 'Xem trước mã'}
          </button>
          <button className="insert-code-btn btn-primary" onClick={handleImport} disabled={gameCount === 0}>
            <Icon name="download" size={14} /> Chèn {gameCount > 1 ? `${gameCount} ván` : 'ván cờ'} vào tài liệu
          </button>
        </>
      }
    >
      <div className="chess-modal-body">
        <div className="pgn-input-section">
          <div className="pgn-input-header">
            <label>Dán nội dung PGN hoặc chọn file .pgn từ máy tính:</label>
            <label className="pgn-file-upload-btn">
              <Icon name="folder-open" size={13} /> Chọn file .pgn
              <input type="file" accept=".pgn,.txt" onChange={handleFileUpload} style={{ display: 'none' }} />
            </label>
          </div>
          <textarea
            className="pgn-textarea"
            value={pgnInput}
            onChange={(e) => setPgnInput(e.target.value)}
            placeholder="[Event &quot;...&quot;]&#10;1. e4 e5..."
            rows={8}
          />
        </div>

        <div className="pgn-options-section" style={{ marginTop: '12px' }}>
          <div className="config-group">
            <label style={{ fontWeight: 600, display: 'block', marginBottom: '6px' }}>Định dạng Xuất bản:</label>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '8px' }}>
              <button
                type="button"
                className={`palette-btn ${layout === 'magazine' ? 'selected' : ''}`}
                style={{ height: 'auto', padding: '8px', textAlign: 'left', display: 'flex', flexDirection: 'column' }}
                onClick={() => setLayout('magazine')}
              >
                <span style={{ fontWeight: 'bold' }}>📰 Tạp chí (2 cột Trắng/Đen)</span>
                <span style={{ fontSize: '11px', opacity: 0.8 }}>Bảng nước đi 2 cột sang trọng, chuẩn xuất bản</span>
              </button>

              <button
                type="button"
                className={`palette-btn ${layout === 'inline' ? 'selected' : ''}`}
                style={{ height: 'auto', padding: '8px', textAlign: 'left', display: 'flex', flexDirection: 'column' }}
                onClick={() => setLayout('inline')}
              >
                <span style={{ fontWeight: 'bold' }}>📝 Biên bản ván cờ liền mạch</span>
                <span style={{ fontSize: '11px', opacity: 0.8 }}>Đoạn văn nước đi liên tục kèm bình luận nghiêng</span>
              </button>

              <button
                type="button"
                className={`palette-btn ${layout === 'header_only' ? 'selected' : ''}`}
                style={{ height: 'auto', padding: '8px', textAlign: 'left', display: 'flex', flexDirection: 'column' }}
                onClick={() => setLayout('header_only')}
              >
                <span style={{ fontWeight: 'bold' }}>📇 Thẻ thông tin ván đấu</span>
                <span style={{ fontSize: '11px', opacity: 0.8 }}>Chỉ lấy bảng tên kỳ thủ, Elo, sự kiện, kết quả</span>
              </button>
            </div>
          </div>

          {gameCount > 0 && (
            <div
              style={{
                marginTop: '12px',
                padding: '8px 12px',
                background: 'var(--color-bg-subtle, #f1f5f9)',
                borderRadius: '6px',
                fontSize: '12px',
              }}
            >
              <b>Tìm thấy {gameCount} ván cờ:</b> {parsedGames[0]?.headers['White'] || 'Trắng'} vs{' '}
              {parsedGames[0]?.headers['Black'] || 'Đen'}{' '}
              {parsedGames[0]?.result ? `(${parsedGames[0].result})` : ''}
              {parsedGames[0]?.moves && ` • ${parsedGames[0].moves.length} nước đi`}
            </div>
          )}

          {showPreviewCode && (
            <div className="config-group" style={{ marginTop: '12px' }}>
              <label>Mã Typst được tạo tự động:</label>
              <pre className="chess-code-preview" style={{ maxHeight: '180px', overflowY: 'auto' }}>
                {generatedCode}
              </pre>
            </div>
          )}
        </div>
      </div>
    </Modal>
  )
}
