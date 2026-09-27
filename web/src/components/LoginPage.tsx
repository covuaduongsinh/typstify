import { useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'
import { BrandMark } from './BrandMark'

export function LoginPage({ onLoggedIn }: { onLoggedIn: () => void }) {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [remember, setRemember] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    const savedUser = localStorage.getItem('typstify_remember_username')
    if (savedUser) {
      setUsername(savedUser)
    }
  }, [])

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    if (mode === 'register') {
      if (password !== confirmPassword) {
        setError('Mật khẩu xác nhận không khớp')
        return
      }
      if (password.length < 4) {
        setError('Mật khẩu phải có ít nhất 4 ký tự')
        return
      }
    }

    setBusy(true)
    try {
      if (mode === 'login') {
        await api.post('/api/auth/login', {
          username: username.trim(),
          password,
        })
      } else {
        await api.post('/api/auth/register', {
          username: username.trim(),
          password,
          displayName: displayName.trim() || undefined,
        })
      }

      if (remember && username.trim()) {
        localStorage.setItem('typstify_remember_username', username.trim())
      } else if (!remember) {
        localStorage.removeItem('typstify_remember_username')
      }

      onLoggedIn()
    } catch (err) {
      setError(
        err instanceof ApiError
          ? err.message
          : mode === 'login'
            ? 'Đăng nhập thất bại'
            : 'Đăng ký tài khoản thất bại',
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="entry-page">
      <div className="entry-card login-form">
        <BrandMark size={56} />
        <h1 className="entry-title">Dương Sinh Chess Studio</h1>
        <p className="entry-slogan">Vui trí tuệ</p>

        <div className="auth-tab-group" role="tablist">
          <button
            type="button"
            className={`auth-tab ${mode === 'login' ? 'active' : ''}`}
            onClick={() => {
              setMode('login')
              setError(null)
            }}
          >
            Đăng nhập
          </button>
          <button
            type="button"
            className={`auth-tab ${mode === 'register' ? 'active' : ''}`}
            onClick={() => {
              setMode('register')
              setError(null)
            }}
          >
            Tạo tài khoản
          </button>
        </div>

        <form onSubmit={submit} className="auth-fields-container" autoComplete="on">
          <label className="entry-field">
            <span>Tên đăng nhập</span>
            <input
              type="text"
              name="username"
              id="username"
              autoComplete="username"
              autoFocus={!username}
              placeholder={mode === 'login' ? 'Tên đăng nhập (hoặc để trống)' : 'Ví dụ: duongsinh'}
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required={mode === 'register'}
            />
          </label>

          {mode === 'register' && (
            <label className="entry-field">
              <span>Tên hiển thị (tùy chọn)</span>
              <input
                type="text"
                name="displayName"
                id="displayName"
                autoComplete="name"
                placeholder="Ví dụ: Thầy Dương Sinh"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
              />
            </label>
          )}

          <label className="entry-field">
            <span>Mật khẩu</span>
            <input
              type="password"
              name="password"
              id="password"
              autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              autoFocus={!!username}
              placeholder={mode === 'login' ? 'Nhập mật khẩu' : 'Tối thiểu 4 ký tự'}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </label>

          {mode === 'register' && (
            <label className="entry-field">
              <span>Xác nhận mật khẩu</span>
              <input
                type="password"
                name="confirmPassword"
                id="confirmPassword"
                autoComplete="new-password"
                placeholder="Nhập lại mật khẩu"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
              />
            </label>
          )}

          <div className="auth-options-row">
            <label className="checkbox-label" style={{ fontSize: 13, cursor: 'pointer' }}>
              <input
                type="checkbox"
                checked={remember}
                onChange={(e) => setRemember(e.target.checked)}
              />
              <span>Ghi nhớ đăng nhập trên máy này</span>
            </label>
          </div>

          <button
            type="submit"
            className="btn-primary entry-submit"
            disabled={busy || !password || (mode === 'register' && !username)}
          >
            {busy
              ? 'Đang xử lý…'
              : mode === 'login'
                ? 'Đăng nhập'
                : 'Tạo tài khoản & Bắt đầu'}
          </button>
        </form>

        {error && (
          <div className="login-error" role="alert">
            {error}
          </div>
        )}

        <div className="auth-footer-switch">
          {mode === 'login' ? (
            <button
              type="button"
              className="btn-link"
              onClick={() => {
                setMode('register')
                setError(null)
              }}
            >
              Chưa có tài khoản? <strong>Tạo tài khoản mới</strong>
            </button>
          ) : (
            <button
              type="button"
              className="btn-link"
              onClick={() => {
                setMode('login')
                setError(null)
              }}
            >
              Đã có tài khoản? <strong>Đăng nhập ngay</strong>
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
