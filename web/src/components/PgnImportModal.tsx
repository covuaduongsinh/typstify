import { useState } from 'react'

interface PgnImportModalProps {
  isOpen: boolean
  onClose: () => void
  onInsertCode: (code: string) => void
}

interface PgnParsed {
  headers: Record<string, string>
  moves: string
}

function parsePgn(pgnText: string): PgnParsed {
  const headers: Record<string, string> = {}
  const lines = pgnText.split('\n')
  const moveLines: string[] = []
  for (const line of lines) {
    const trimmed = line.trim()
    if (!trimmed) {
      continue
    }

    if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
      const match = trimmed.match(/\[([A-Za-z0-9_]+)\s+"(.*)"\]/)
      if (match) {
        headers[match[1]] = match[2]
      }
    } else {
      moveLines.push(trimmed)
    }
  }

  return {
    headers,
    moves: moveLines.join(' '),
  }
}

function convertPgnToTypst(parsed: PgnParsed): string {
  const h = parsed.headers
  const white = h['White'] || 'Trắng'
  const black = h['Black'] || 'Đen'
  const whiteElo = h['WhiteElo'] || '2700'
  const blackElo = h['BlackElo'] || '2700'
  const whiteFed = h['WhiteFed'] || ''
  const blackFed = h['BlackFed'] || ''
  const whiteTitle = h['WhiteTitle'] || 'GM'
  const blackTitle = h['BlackTitle'] || 'GM'
  const event = h['Event'] || 'Giải Đấu Cờ Vua'
  const site = h['Site'] || 'Hà Nội'
  const date = h['Date'] || new Date().toISOString().slice(0, 10)
  const round = h['Round'] || '1'
  const result = h['Result'] || '1 - 0'
  const eco = h['ECO'] || 'B90'
  const opening = h['Opening'] || 'Khai Cuộc'

  // Clean moves and replace some notation
  let moves = parsed.moves
    .replace(/\r?\n/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()

  return `// ============================================================================
// VÁN ĐẤU: ${white} vs ${black} (${event})
// ============================================================================

#game-header(
  white: "${white}",
  white-title: "${whiteTitle}",
  white-elo: "${whiteElo}",
  white-fed: "${whiteFed}",
  black: "${black}",
  black-title: "${blackTitle}",
  black-elo: "${blackElo}",
  black-fed: "${blackFed}",
  event: "${event}",
  site: "${site}",
  date: "${date}",
  round: "${round}",
  result: "${result}",
  eco: "${eco}",
  opening: "${opening}"
)

#v(6pt)

${moves}
\n`
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

  if (!isOpen) return null

  const handleImport = () => {
    const parsed = parsePgn(pgnInput)
    const typstCode = convertPgnToTypst(parsed)
    onInsertCode(typstCode)
    onClose()
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
    <div className="chess-modal-overlay">
      <div className="chess-modal-container pgn-modal">
        <div className="chess-modal-header">
          <h3>📜 Nhập Ván Cờ Từ PGN & Chuyển Sang Typst</h3>
          <button className="chess-modal-close" onClick={onClose}>
            ✕
          </button>
        </div>

        <div className="chess-modal-body">
          <div className="pgn-input-section">
            <div className="pgn-input-header">
              <label>Dán nội dung PGN hoặc chọn file .pgn từ máy tính:</label>
              <label className="pgn-file-upload-btn">
                📂 Tải file PGN
                <input
                  type="file"
                  accept=".pgn,.txt"
                  onChange={handleFileUpload}
                  style={{ display: 'none' }}
                />
              </label>
            </div>
            <textarea
              className="pgn-textarea"
              value={pgnInput}
              onChange={(e) => setPgnInput(e.target.value)}
              placeholder="[Event &quot;...&quot;]&#10;1. e4 e5..."
              rows={12}
            />
          </div>
        </div>

        <div className="chess-modal-footer">
          <button className="small-action-btn" onClick={() => setPgnInput(SAMPLE_PGN)}>
            🔄 Nạp ván mẫu
          </button>
          <button className="insert-code-btn" onClick={handleImport}>
            ✨ Chuyển đổi & Chèn vào Tài Liệu
          </button>
        </div>
      </div>
    </div>
  )
}
