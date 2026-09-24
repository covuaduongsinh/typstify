import { useEffect, useRef, useState } from 'react'
import { api } from '../api/client'
import type { AuthMethod, AuthRequiredData } from '../lib/acpTypes'

const URL_RE = /https?:\/\/\S+/

/** AuthCard renders when /ws/agent reports "authRequired" (see
 * server/agent_ws.go's startSessionOrRequireAuth). Driving
 * POST /api/agent/auth/{methodId} (server/agent_api.go's handleAgentAuth)
 * blocks server-side until the agent finishes its own login flow -- for an
 * agent-managed OAuth method that can be as long as the user needs to open
 * a login link and complete it, so this polls GET /api/console (the same
 * stderr stream the desktop app's Console panel shows) for a URL the agent
 * printed, the same "pipe + regex" trick already proven for this app in
 * scratch/auth_antigravity.go. */
export function AuthCard({ data, onDone }: { data: AuthRequiredData; onDone: () => void }) {
  const [runningId, setRunningId] = useState<string | null>(null)
  const [foundUrl, setFoundUrl] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const pollRef = useRef<number | undefined>(undefined)

  useEffect(() => () => window.clearInterval(pollRef.current), [])

  const start = async (method: AuthMethod) => {
    setRunningId(method.id)
    setFoundUrl(null)
    setError(null)

    pollRef.current = window.setInterval(async () => {
      try {
        const text = await api.get<string>('/api/console')
        const match = text.match(URL_RE)
        if (match) setFoundUrl(match[0])
      } catch {
        // ignore transient poll failures
      }
    }, 1500)

    try {
      await api.post(`/api/agent/auth/${encodeURIComponent(method.id)}`)
      window.clearInterval(pollRef.current)
      setRunningId(null)
      onDone()
    } catch (err) {
      window.clearInterval(pollRef.current)
      setRunningId(null)
      setError(err instanceof Error ? err.message : 'Xác thực thất bại')
    }
  }

  return (
    <div className="auth-card">
      <div className="auth-card-title">{data.agentName || 'Trợ lý AI'} cần bạn đăng nhập</div>
      <div className="auth-methods">
        {data.authMethods.map((m) => (
          <div key={m.id} className="auth-method">
            <div className="auth-method-name">{m.name}</div>
            {m.description && <div className="auth-method-desc">{m.description}</div>}
            {runningId === m.id ? (
              <div className="auth-in-progress">
                <span>Đang chờ bạn hoàn tất đăng nhập…</span>
                {foundUrl && (
                  <a href={foundUrl} target="_blank" rel="noreferrer" className="auth-open-link">
                    Mở liên kết đăng nhập
                  </a>
                )}
              </div>
            ) : (
              <button disabled={runningId !== null} onClick={() => start(m)}>
                Đăng nhập
              </button>
            )}
          </div>
        ))}
      </div>
      {error && <div className="error">{error}</div>}
    </div>
  )
}
