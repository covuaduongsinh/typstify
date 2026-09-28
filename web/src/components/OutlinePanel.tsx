import { useEffect, useState } from 'react'
import type { LspDocumentSymbol } from '../lib/lspClient'
import { Icon } from './Icon'

interface OutlinePanelProps {
  /** Path of the currently open file; the panel resets whenever it changes. */
  path: string | null
  /** Fetches the outline for the currently open file (see EditorHandle.getOutline). */
  fetchOutline: () => Promise<LspDocumentSymbol[]>
  onSelect: (line: number, character: number) => void
}

// SymbolKind values per the LSP spec (textDocument/documentSymbol).
const KIND_LABEL: Record<number, string> = {
  5: 'class',
  6: 'method',
  8: 'field',
  12: 'function',
  13: 'variable',
  15: 'string',
}

function SymbolRow({
  symbol,
  depth,
  onSelect,
}: {
  symbol: LspDocumentSymbol
  depth: number
  onSelect: (line: number, character: number) => void
}) {
  return (
    <>
      <button
        className="outline-item"
        style={{ paddingLeft: 10 + depth * 14 }}
        title={symbol.detail || symbol.name}
        onClick={() => onSelect(symbol.selectionRange.start.line, symbol.selectionRange.start.character)}
      >
        <span className="outline-item-kind">{KIND_LABEL[symbol.kind] ?? '•'}</span>
        <span className="outline-item-name">{symbol.name}</span>
      </button>
      {symbol.children?.map((child, i) => (
        <SymbolRow key={`${child.name}-${i}`} symbol={child} depth={depth + 1} onSelect={onSelect} />
      ))}
    </>
  )
}

export function OutlinePanel({ path, fetchOutline, onSelect }: OutlinePanelProps) {
  const [symbols, setSymbols] = useState<LspDocumentSymbol[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const reload = () => {
    setBusy(true)
    setError(null)
    fetchOutline()
      .then((res) => setSymbols(res))
      .catch((err) => setError(err instanceof Error ? err.message : 'Không tải được outline'))
      .finally(() => setBusy(false))
  }

  useEffect(() => {
    if (!path) {
      setSymbols([])
      return
    }
    reload()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [path])

  return (
    <div className="outline-panel">
      <div className="outline-panel-header">
        <span>Dàn ý tài liệu</span>
        <button className="outline-refresh-btn" onClick={reload} disabled={!path || busy} title="Làm mới">
          <Icon name="refresh" size={14} />
        </button>
      </div>
      {!path ? (
        <div className="outline-empty">Chưa mở tệp nào.</div>
      ) : busy ? (
        <div className="outline-empty">Đang tải…</div>
      ) : error ? (
        <div className="outline-empty error">{error}</div>
      ) : symbols.length === 0 ? (
        <div className="outline-empty">Tài liệu chưa có heading/section nào.</div>
      ) : (
        <div className="outline-list">
          {symbols.map((s, i) => (
            <SymbolRow key={`${s.name}-${i}`} symbol={s} depth={0} onSelect={onSelect} />
          ))}
        </div>
      )}
    </div>
  )
}
