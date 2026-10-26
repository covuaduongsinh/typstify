import { useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { AuthStatus } from '../api/types'
import { Icon } from './Icon'

type Setup = { secret: string; otpauthUri: string }

/** SecuritySection turns the account's TOTP second factor on and off
 * (server/totp.go). The secret is shown for manual entry in any
 * authenticator app (Google Authenticator, Aegis, 1Password, ...). */
export function SecuritySection() {
  const [status, setStatus] = useState<AuthStatus | null>(null)
  const [setup, setSetup] = useState<Setup | null>(null)
  const [code, setCode] = useState('')
  const [password, setPassword] = useState('')
  const [recovery, setRecovery] = useState<string[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const refresh = () =>
    api
      .get<AuthStatus>('/api/auth/status')
      .then(setStatus)
      .catch(() => setStatus(null))

  useEffect(() => {
    void refresh()
  }, [])

  const act = async (fn: () => Promise<void>) => {
    setError(null)
    setBusy(true)
    try {
      await fn()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Thao tác thất bại')
    } finally {
      setBusy(false)
    }
  }

  const start = () => act(async () => setSetup(await api.post<Setup>('/api/auth/mfa/setup')))

  const enable = () =>
    act(async () => {
      const res = await api.post<{ recoveryCodes: string[] }>('/api/auth/mfa/enable', { code: code.trim(), password })
      setRecovery(res.recoveryCodes)
      setSetup(null)
      setCode('')
      setPassword('')
      await refresh()
    })

  const disable = () =>
    act(async () => {
      await api.post('/api/auth/mfa/disable', { password, code: code.trim() })
      setPassword('')
      setCode('')
      setRecovery(null)
      await refresh()
    })

  // Accounts only exist when the server has users; the legacy single
  // password login has nothing to protect with a second factor here.
  if (!status?.username || !status.hasUsers) return null

  return (
    <details className="settings-section">
      <summary>
        <Icon name="chevron-right" size={14} className="settings-caret" />
        Bảo mật · Xác thực hai bước
      </summary>
      <div className="settings-section-body">
        {error && (
          <div className="error" role="alert">
            {error}
          </div>
        )}

        {recovery && (
          <div className="mfa-recovery">
            <p className="settings-hint">
              Lưu các mã khôi phục này ở nơi an toàn. Mỗi mã dùng được một lần khi bạn mất ứng dụng xác thực. Chúng
              sẽ không hiện lại.
            </p>
            <pre className="mfa-codes">{recovery.join('\n')}</pre>
          </div>
        )}

        {status.mfaEnabled ? (
          <form
            onSubmit={(e) => {
              e.preventDefault()
              void disable()
            }}
          >
            <p className="settings-hint">
              <Icon name="check" size={12} /> Đang bật. Muốn tắt, nhập mật khẩu và một mã xác thực.
            </p>
            <label className="settings-field">
              <span>Mật khẩu</span>
              <input type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} />
            </label>
            <label className="settings-field">
              <span>Mã xác thực / mã khôi phục</span>
              <input inputMode="numeric" autoComplete="one-time-code" value={code} onChange={(e) => setCode(e.target.value)} />
            </label>
            <div className="settings-actions">
              <button type="submit" disabled={busy || !password || !code.trim()}>
                Tắt xác thực hai bước
              </button>
            </div>
          </form>
        ) : setup ? (
          <form
            onSubmit={(e) => {
              e.preventDefault()
              void enable()
            }}
          >
            <p className="settings-hint">
              Mở ứng dụng xác thực, chọn thêm tài khoản bằng khoá thủ công (TOTP, 6 số, 30 giây) và nhập khoá dưới
              đây. Sau đó nhập mã 6 số ứng dụng hiển thị.
            </p>
            <label className="settings-field">
              <span>Khoá bí mật</span>
              <input readOnly value={setup.secret} onFocus={(e) => e.currentTarget.select()} />
            </label>
            <p className="settings-hint">
              Hoặc mở trên điện thoại: <a href={setup.otpauthUri}>liên kết otpauth</a>
            </p>
            <label className="settings-field">
              <span>Mã 6 số</span>
              <input
                inputMode="numeric"
                autoComplete="one-time-code"
                maxLength={6}
                value={code}
                onChange={(e) => setCode(e.target.value)}
              />
            </label>
            <label className="settings-field">
              <span>Mật khẩu tài khoản</span>
              <input type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} />
            </label>
            <div className="settings-actions">
              <button type="submit" className="btn-primary" disabled={busy || code.trim().length !== 6 || !password}>
                Xác nhận và bật
              </button>
            </div>
          </form>
        ) : (
          <>
            <p className="settings-hint">
              Thêm một lớp bảo vệ: khi đăng nhập, ngoài mật khẩu cần nhập mã từ ứng dụng xác thực trên điện thoại.
            </p>
            <div className="settings-actions">
              <button className="btn-primary" disabled={busy} onClick={() => void start()}>
                Bật xác thực hai bước
              </button>
            </div>
          </>
        )}
      </div>
    </details>
  )
}
