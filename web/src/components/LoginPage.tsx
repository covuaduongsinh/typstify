import { useState } from 'react'
import { ApiError, api } from '../api/client'
import { useTranslations } from '../lib/i18n'

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
      setError(err instanceof ApiError ? err.message : 'Login failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="login-page">
      <form className="login-form" onSubmit={submit}>
        <h1>Typstify</h1>
        <input
          type="password"
          autoFocus
          placeholder="Server password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <button type="submit" disabled={busy}>
          {t('Sign In')}
        </button>
        {error && <div className="login-error">{error}</div>}
      </form>
    </div>
  )
}
