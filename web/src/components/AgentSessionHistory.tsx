import { useEffect, useMemo, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { AcpSessionSummary } from '../lib/acpTypes'
import { Icon } from './Icon'

function formatUpdatedAt(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleString('vi-VN', { dateStyle: 'medium', timeStyle: 'short' })
}

/** AgentSessionHistory lists past ACP sessions for the open project (GET
 * /api/agent/sessions) and lets the user pick one to load back into the chat
 * -- the web counterpart of the desktop app's session history panel
 * (ui/assistant/sessions.go). */
export function AgentSessionHistory({ onSelect, onClose }: { onSelect: (sessionId: string) => void; onClose: () => void }) {
  const [sessions, setSessions] = useState<AcpSessionSummary[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [query, setQuery] = useState('')

  useEffect(() => {
    let cancelled = false
    api
      .get<AcpSessionSummary[]>('/api/agent/sessions')
      .then((res) => {
        if (cancelled) return
        setSessions(res ?? [])
      })
      .catch((err) => {
        if (cancelled) return
        setError(err instanceof ApiError ? err.message : 'Không tải được lịch sử phiên')
      })
    return () => {
      cancelled = true
    }
  }, [])

  const filtered = useMemo(() => {
    const list = sessions ?? []
    // Newest first -- sessions missing updatedAt (agent didn't report one)
    // sort after ones that have it, keeping their relative order.
    const sorted = [...list].sort((a, b) => {
      if (!a.updatedAt && !b.updatedAt) return 0
      if (!a.updatedAt) return 1
      if (!b.updatedAt) return -1
      return b.updatedAt.localeCompare(a.updatedAt)
    })
    const q = query.trim().toLowerCase()
    if (!q) return sorted
    return sorted.filter((s) => (s.title || s.sessionId).toLowerCase().includes(q))
  }, [sessions, query])

  return (
    <div className="agent-session-history">
      <div className="agent-session-history-header">
        <div className="agent-session-history-search">
          <input
            autoFocus
            placeholder="Tìm cuộc trò chuyện cũ…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
        <button className="btn-ghost preview-tool" title="Đóng lịch sử" onClick={onClose}>
          <Icon name="x" size={14} />
        </button>
      </div>

      <div className="agent-session-history-list">
        {error && <div className="error">{error}</div>}
        {!sessions && !error && <p className="settings-hint">Đang tải…</p>}
        {sessions && filtered.length === 0 && !error && (
          <p className="settings-hint">
            {sessions.length === 0 ? 'Chưa có cuộc trò chuyện nào trước đây.' : 'Không tìm thấy cuộc trò chuyện phù hợp.'}
          </p>
        )}
        {filtered.map((s) => (
          <button key={s.sessionId} className="agent-session-history-item" onClick={() => onSelect(s.sessionId)}>
            <span className="agent-session-history-item-title">{s.title || s.sessionId}</span>
            {s.updatedAt && <span className="agent-session-history-item-time">{formatUpdatedAt(s.updatedAt)}</span>}
          </button>
        ))}
      </div>
    </div>
  )
}
