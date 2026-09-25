import { useEffect, useRef, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { AuthMethod, AuthRequiredData } from '../lib/acpTypes'

const URL_RE = /https?:\/\/[^\s"'<>]+/g

/** latestLoginUrl picks the most recent non-loopback URL the agent printed
 * (older attempts stay in the console log, so the first match can be a
 * stale link whose login would no longer be awaited). */
function latestLoginUrl(consoleText: string): string | null {
  const urls = (consoleText.match(URL_RE) ?? []).filter((u) => !/^https?:\/\/(127\.0\.0\.1|localhost)[:/]/.test(u))
  return urls.length > 0 ? urls[urls.length - 1] : null
}

/** looksLikeCallback is a quick client-side check of a pasted address,
 * so an obvious mistake is reported before a round trip. */
function looksLikeCallback(url: string): string | null {
  const u = url.trim()
  if (!/^http:\/\/(127\.0\.0\.1|localhost|\[::1\]):\d+/.test(u)) {
    return 'Địa chỉ cần bắt đầu bằng http://127.0.0.1:… (địa chỉ của trang lỗi sau khi đăng nhập).'
  }
  if (!/[?&](code|error)=/.test(u)) {
    return 'Địa chỉ thiếu phần code=… — hãy sao chép toàn bộ địa chỉ trên thanh địa chỉ.'
  }
  return null
}

/** AuthCard renders when /ws/agent reports "authRequired" (see
 * server/agent_ws.go's startSessionOrRequireAuth). Driving
 * POST /api/agent/auth/{methodId} (server/agent_api.go's handleAgentAuth)
 * blocks server-side until the agent finishes its own login flow -- for an
 * agent-managed OAuth method that can be as long as the user needs to open
 * a login link and complete it, so this polls GET /api/console (the same
 * stderr stream the desktop app's Console panel shows) for a URL the agent
 * printed, the same "pipe + regex" trick already proven for this app in
 * scratch/auth_antigravity.go.
 *
 * OAuth "loopback" logins (e.g. Log in with Google) end by redirecting the
 * browser to http://127.0.0.1:<port> -- the agent's port inside the server,
 * not on the user's machine -- so the browser shows a connection error.
 * The user pastes that address here and the server relays it to the agent
 * (POST /api/agent/auth/callback, server/agent_auth_callback.go). */
export function AuthCard({ data, onDone }: { data: AuthRequiredData; onDone: () => void }) {
  const [runningId, setRunningId] = useState<string | null>(null)
  const [foundUrl, setFoundUrl] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [callbackUrl, setCallbackUrl] = useState('')
  const [callbackError, setCallbackError] = useState<string | null>(null)
  const [callbackBusy, setCallbackBusy] = useState(false)
  const [callbackSent, setCallbackSent] = useState(false)
  const pollRef = useRef<number | undefined>(undefined)

  useEffect(() => () => window.clearInterval(pollRef.current), [])

  const start = async (method: AuthMethod) => {
    setRunningId(method.id)
    setFoundUrl(null)
    setError(null)
    setCallbackUrl('')
    setCallbackError(null)
    setCallbackSent(false)

    pollRef.current = window.setInterval(async () => {
      try {
        const text = await api.get<string>('/api/console')
        const url = latestLoginUrl(text)
        if (url) setFoundUrl(url)
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

  const sendCallback = async () => {
    const problem = looksLikeCallback(callbackUrl)
    if (problem) {
      setCallbackError(problem)
      return
    }
    setCallbackError(null)
    setCallbackBusy(true)
    try {
      await api.post('/api/agent/auth/callback', { url: callbackUrl.trim() })
      // The pending /api/agent/auth/{method} call resolves once the agent
      // has exchanged the code, which then calls onDone().
      setCallbackSent(true)
    } catch (err) {
      setCallbackError(err instanceof ApiError || err instanceof Error ? err.message : 'Không gửi được mã đăng nhập')
    } finally {
      setCallbackBusy(false)
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
                {!foundUrl ? (
                  <span>
                    <span className="spinner" /> Đang lấy liên kết đăng nhập…
                  </span>
                ) : (
                  <ol className="auth-steps">
                    <li>
                      <a href={foundUrl} target="_blank" rel="noreferrer" className="auth-open-link">
                        Mở liên kết đăng nhập
                      </a>{' '}
                      và đăng nhập tài khoản của bạn.
                    </li>
                    <li>
                      Sau đó trình duyệt mở trang <b>không kết nối được</b> (địa chỉ <code>127.0.0.1:…</code>) — đó là bình
                      thường. Sao chép <b>toàn bộ địa chỉ</b> trên thanh địa chỉ của trang đó.
                    </li>
                    <li>
                      Dán vào đây rồi bấm <b>Hoàn tất đăng nhập</b>:
                      <div className="auth-callback-row">
                        <input
                          value={callbackUrl}
                          onChange={(e) => {
                            setCallbackUrl(e.target.value)
                            setCallbackError(null)
                          }}
                          onKeyDown={(e) => e.key === 'Enter' && void sendCallback()}
                          placeholder="http://127.0.0.1:…/?state=…&code=…"
                          aria-label="Địa chỉ trang sau khi đăng nhập"
                          disabled={callbackBusy || callbackSent}
                        />
                        <button
                          className="btn-primary"
                          onClick={() => void sendCallback()}
                          disabled={!callbackUrl.trim() || callbackBusy || callbackSent}
                        >
                          {callbackBusy ? 'Đang gửi…' : 'Hoàn tất đăng nhập'}
                        </button>
                      </div>
                      {callbackError && (
                        <div className="error" role="alert">
                          {callbackError}
                        </div>
                      )}
                      {callbackSent && (
                        <div className="auth-callback-ok" role="status">
                          <span className="spinner" /> Đã gửi mã đăng nhập, đang chờ trợ lý AI xác nhận…
                        </div>
                      )}
                    </li>
                  </ol>
                )}
                <small className="auth-note">
                  Mã đăng nhập chỉ dùng được trong vài phút. Nếu báo lỗi, bấm tải lại trang rồi đăng nhập lại.
                </small>
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
