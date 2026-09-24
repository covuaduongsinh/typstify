import { useState } from 'react'
import { ApiError, api } from '../api/client'
import { useTranslations } from '../lib/i18n'
import { BrandMark } from './BrandMark'

export function LoginPage({ onLoggedIn }: { onLoggedIn: () => void }) {
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const t = useTranslations(['Sign In'])

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await api.post('/api/auth/login', { password })
      onLoggedIn()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Đăng nhập thất bại')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="entry-page">
      <form className="entry-card login-form" onSubmit={submit}>
        <BrandMark size={56} />
        <h1 className="entry-title">Dương Sinh Chess Studio</h1>
        <p className="entry-slogan">Vui trí tuệ</p>
        <label className="entry-field">
          <span>Mật khẩu máy chủ</span>
          <input
            type="password"
            autoFocus
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            aria-invalid={!!error}
          />
        </label>
        <button type="submit" className="btn-primary entry-submit" disabled={busy || !password}>
          {busy ? 'Đang đăng nhập…' : t('Sign In')}
        </button>
        {error && (
          <div className="login-error" role="alert">
            {error}
          </div>
        )}
      </form>
    </div>
  )
}
