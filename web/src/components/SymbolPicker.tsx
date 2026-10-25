import { useEffect, useMemo, useState } from 'react'
import { ApiError, api } from '../api/client'
import { filterSymbols, type TypstSymbol } from '../lib/symbols'
import { Modal } from './Modal'

let cache: TypstSymbol[] | null = null

/** Searchable table of Typst's sym.* symbols. Click inserts `#sym.name`,
 * Shift+click the character itself. */
export function SymbolPicker({ onInsert, onClose }: { onInsert: (text: string) => void; onClose: () => void }) {
  const [list, setList] = useState<TypstSymbol[] | null>(cache)
  const [error, setError] = useState<string | null>(null)
  const [query, setQuery] = useState('')

  useEffect(() => {
    if (cache) return
    let alive = true
    api
      .get<TypstSymbol[]>('/api/symbols')
      .then((l) => {
        cache = l
        if (alive) setList(l)
      })
      .catch((err: unknown) => alive && setError(err instanceof ApiError ? err.message : 'Không tải được bảng ký hiệu'))
    return () => {
      alive = false
    }
  }, [])

  const shown = useMemo(() => (list ? filterSymbols(list, query) : []), [list, query])

  return (
    <Modal title="Ký hiệu Typst" onClose={onClose} className="symbol-picker-modal">
      <input
        className="symbol-search"
        autoFocus
        placeholder="Tìm theo tên (arrow, alpha…) hoặc dán một ký tự"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      <p className="settings-hint">Bấm để chèn #sym.tên; giữ Shift khi bấm để chèn thẳng ký tự.</p>
      {error && (
        <div className="error" role="alert">
          {error}
        </div>
      )}
      {!list && !error && <span className="spinner" />}
      <div className="symbol-grid" role="list">
        {shown.map((s) => (
          <button
            key={s.name}
            role="listitem"
            className="symbol-cell"
            title={`sym.${s.name}`}
            onClick={(e) => onInsert(e.shiftKey ? s.char : `#sym.${s.name}`)}
          >
            <span className="symbol-char">{s.char}</span>
            <span className="symbol-name">{s.name}</span>
          </button>
        ))}
      </div>
    </Modal>
  )
}
