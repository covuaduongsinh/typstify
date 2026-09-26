import { useEffect, useState, type ReactNode } from 'react'
import { ApiError, api } from '../api/client'
import type { AgentSettings, GeneralSettings, LspSettings, TypstSettings } from '../api/types'
import { AgentRegistryPicker } from './AgentRegistryPicker'
import { Icon } from './Icon'

interface Field<T> {
  key: keyof T
  label: string
  hint?: string
  /** Renders a <select> instead of a text input. */
  options?: Array<{ value: string; label: string }>
  placeholder?: string
}

// Values accepted by the desktop app (i18n/localizer.go, ui/palette/palette.go).
const DESKTOP_LANGUAGES = [
  { value: 'en-us', label: 'English' },
  { value: 'zh-cn', label: '中文' },
  { value: 'de', label: 'Deutsch' },
]
const DESKTOP_THEMES = [
  'Default Light',
  'Default Dark',
  'Solarized Light',
  'Solarized Dark',
  'Nord Light',
  'Nord Dark',
  'Dracula',
  'Gruvbox Light',
  'Gruvbox Dark',
].map((v) => ({ value: v, label: v }))

function Section<T extends object>({
  title,
  path,
  fields,
  description,
  defaultOpen = false,
}: {
  title: string
  path: string
  fields: Array<Field<T>>
  description?: ReactNode
  defaultOpen?: boolean
}) {
  const [value, setValue] = useState<T | null>(null)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    api
      .get<T>(path)
      .then(setValue)
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Không tải được cài đặt'))
  }, [path])

  const save = async () => {
    if (!value) return
    setError(null)
    setBusy(true)
    try {
      const updated = await api.putJson<T>(path, value)
      setValue(updated)
      setSaved(true)
      setTimeout(() => setSaved(false), 1500)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Không lưu được cài đặt')
    } finally {
      setBusy(false)
    }
  }

  return (
    <details className="settings-section" open={defaultOpen}>
      <summary>
        <Icon name="chevron-right" size={14} className="settings-caret" />
        {title}
      </summary>
      <div className="settings-section-body">
        {description && <p className="settings-hint">{description}</p>}
        {!value && !error && <p className="settings-hint">Đang tải…</p>}
        {value &&
          fields.map(({ key, label, hint, options, placeholder }) => {
            const current = String(value[key] ?? '')
            const set = (v: string) => setValue({ ...value, [key]: v } as T)
            return (
              <label key={String(key)} className="settings-field">
                <span>{label}</span>
                {options ? (
                  <select value={current} onChange={(e) => set(e.target.value)}>
                    {/* keep an unknown stored value selectable rather than silently changing it */}
                    {!options.some((o) => o.value === current) && <option value={current}>{current || '—'}</option>}
                    {options.map((o) => (
                      <option key={o.value} value={o.value}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                ) : (
                  <input value={current} placeholder={placeholder} onChange={(e) => set(e.target.value)} />
                )}
                {hint && <small>{hint}</small>}
              </label>
            )
          })}
        {value && fields.length > 0 && (
          <div className="settings-actions">
            <button className="btn-primary" onClick={save} disabled={busy}>
              {busy ? 'Đang lưu…' : 'Lưu'}
            </button>
            {saved && (
              <span className="settings-saved">
                <Icon name="check" size={13} /> Đã lưu
              </span>
            )}
          </div>
        )}
        {error && <div className="error">{error}</div>}
      </div>
    </details>
  )
}

export function SettingsPanel() {
  const [agentSettingsVersion, setAgentSettingsVersion] = useState(0)

  return (
    <div className="settings-panel">
      <h2 className="settings-title">Cài đặt</h2>
      <p className="settings-hint">
        Giao diện Sáng/Tối của bản web đổi bằng nút <Icon name="moon" size={12} /> trên thanh tiêu đề và được nhớ
        trên trình duyệt này.
      </p>

      <Section<TypstSettings>
        title="Typst & font"
        path="/api/settings/typst"
        defaultOpen
        fields={[
          { key: 'extraFontPath', label: 'Thư mục font bổ sung', hint: 'Nơi đặt font tiếng Việt / font ký hiệu cờ.' },
          { key: 'localPkgDir', label: 'Thư mục gói cục bộ' },
          { key: 'cacheDir', label: 'Thư mục cache gói' },
        ]}
      />

      <details className="settings-section" open>
        <summary>
          <Icon name="chevron-right" size={14} className="settings-caret" />
          Trợ lý AI
        </summary>
        <div className="settings-section-body">
          <AgentRegistryPicker onSelected={() => setAgentSettingsVersion((v) => v + 1)} />
        </div>
      </details>

      <Section<GeneralSettings>
        title="Ứng dụng desktop"
        path="/api/settings/general"
        description="Các tùy chọn này áp dụng cho ứng dụng Typstify desktop dùng chung cấu hình."
        fields={[
          { key: 'language', label: 'Ngôn ngữ', options: DESKTOP_LANGUAGES },
          { key: 'theme', label: 'Giao diện', options: DESKTOP_THEMES },
          { key: 'externalTypst', label: 'Đường dẫn typst (tùy chọn)', placeholder: 'Để trống để dùng bản đi kèm' },
          { key: 'externalTinymist', label: 'Đường dẫn tinymist (tùy chọn)', placeholder: 'Để trống để dùng bản đi kèm' },
        ]}
      />

      <Section<AgentSettings>
        key={agentSettingsVersion}
        title="Nâng cao: cấu hình trợ lý AI thủ công"
        path="/api/settings/agent"
        fields={[
          { key: 'agentName', label: 'Tên' },
          { key: 'cmd', label: 'Lệnh' },
          { key: 'args', label: 'Tham số (cách nhau bởi dấu cách)' },
          { key: 'env', label: 'Biến môi trường (KEY=value, cách nhau bởi dấu cách)' },
        ]}
      />

      <DropboxSettingsSection />

      <Section<LspSettings> title="Nâng cao: LSP" path="/api/settings/lsp" fields={[]} description="Chưa có tùy chọn nào cho bản web." />
    </div>
  )
}

function DropboxSettingsSection() {
  const [status, setStatus] = useState<{
    connected: boolean
    account?: { display_name: string; email: string }
    syncFolder: string
    autoSync: boolean
    autoSyncInterval: number
    syncOnSave: boolean
    appKey?: string
  } | null>(null)
  const [accessToken, setAccessToken] = useState('')
  const [refreshToken, setRefreshToken] = useState('')
  const [appKey, setAppKey] = useState('')
  const [appSecret, setAppSecret] = useState('')
  const [syncFolder, setSyncFolder] = useState('/Typstify')
  const [autoSync, setAutoSync] = useState(false)
  const [autoSyncInterval, setAutoSyncInterval] = useState(10)
  const [syncOnSave, setSyncOnSave] = useState(false)
  const [saved, setSaved] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = () => {
    api
      .get<any>('/api/dropbox/status')
      .then((res) => {
        setStatus(res)
        setSyncFolder(res.syncFolder || '/Typstify')
        setAutoSync(res.autoSync || false)
        setAutoSyncInterval(res.autoSyncInterval || 10)
        setSyncOnSave(res.syncOnSave || false)
        if (res.appKey) setAppKey(res.appKey)
      })
      .catch(() => {})
  }

  useEffect(() => {
    load()
  }, [])

  const save = async () => {
    setBusy(true)
    setError(null)
    try {
      await api.post('/api/dropbox/auth/token', {
        accessToken: accessToken.trim(),
        refreshToken: refreshToken.trim(),
        appKey: appKey.trim(),
        appSecret: appSecret.trim(),
        syncFolder: syncFolder.trim() || '/Typstify',
        autoSync,
        autoSyncInterval: Number(autoSyncInterval) || 10,
        syncOnSave,
      })
      setSaved(true)
      setAccessToken('')
      setRefreshToken('')
      load()
      setTimeout(() => setSaved(false), 1500)
    } catch (err: any) {
      setError(err instanceof ApiError ? err.message : 'Không lưu được cài đặt Dropbox')
    } finally {
      setBusy(false)
    }
  }

  const disconnect = async () => {
    if (!confirm('Ngắt kết nối tài khoản Dropbox?')) return
    setBusy(true)
    try {
      await api.post('/api/dropbox/auth/disconnect', {})
      load()
    } catch {
    } finally {
      setBusy(false)
    }
  }

  return (
    <details className="settings-section">
      <summary>
        <Icon name="chevron-right" size={14} className="settings-caret" />
        Đồng bộ Dropbox
      </summary>
      <div className="settings-section-body">
        <p className="settings-hint">
          Tự động đồng bộ tài liệu, giáo trình và bài tập với tài khoản Dropbox cá nhân.
        </p>

        {status?.connected ? (
          <div className="settings-field" style={{ marginBottom: 12 }}>
            <span style={{ color: 'var(--color-success, #22c55e)' }}>
              <Icon name="check" size={14} /> Đã kết nối: {status.account?.display_name} ({status.account?.email})
            </span>
            <button className="btn-secondary btn-sm" onClick={disconnect} disabled={busy} style={{ width: 'fit-content' }}>
              Ngắt kết nối
            </button>
          </div>
        ) : (
          <p className="settings-hint" style={{ color: 'var(--color-warning, #eab308)' }}>
            Chưa kết nối tài khoản Dropbox. Nhập Access Token dưới đây để kích hoạt.
          </p>
        )}

        <label className="settings-field">
          <span>Access Token / Refresh Token</span>
          <input
            type="password"
            placeholder={status?.connected ? '••••••••••••••••' : 'Nhập token từ Dropbox Console'}
            value={accessToken}
            onChange={(e) => setAccessToken(e.target.value)}
          />
        </label>

        <div className="grid-2col">
          <label className="settings-field">
            <span>App Key (tùy chọn)</span>
            <input
              placeholder="App key"
              value={appKey}
              onChange={(e) => setAppKey(e.target.value)}
            />
          </label>
          <label className="settings-field">
            <span>App Secret (tùy chọn)</span>
            <input
              type="password"
              placeholder="••••••••"
              value={appSecret}
              onChange={(e) => setAppSecret(e.target.value)}
            />
          </label>
        </div>

        <label className="settings-field">
          <span>Thư mục đồng bộ trên Dropbox</span>
          <input value={syncFolder} onChange={(e) => setSyncFolder(e.target.value)} placeholder="/Typstify" />
        </label>

        <div className="checkbox-row" style={{ marginTop: 8 }}>
          <label className="checkbox-label">
            <input type="checkbox" checked={autoSync} onChange={(e) => setAutoSync(e.target.checked)} />
            <span>Tự động đồng bộ định kỳ</span>
          </label>
          {autoSync && (
            <label className="inline-input">
              <span>Mỗi</span>
              <input
                type="number"
                min={1}
                max={120}
                value={autoSyncInterval}
                onChange={(e) => setAutoSyncInterval(Number(e.target.value))}
                style={{ width: 60 }}
              />
              <span>phút</span>
            </label>
          )}
        </div>

        <div className="checkbox-row" style={{ marginTop: 8 }}>
          <label className="checkbox-label">
            <input type="checkbox" checked={syncOnSave} onChange={(e) => setSyncOnSave(e.target.checked)} />
            <span>Tự động đẩy lên Dropbox khi lưu (Sync on save)</span>
          </label>
        </div>

        <div className="settings-actions" style={{ marginTop: 12 }}>
          <button className="btn-primary" onClick={save} disabled={busy}>
            {busy ? 'Đang lưu…' : 'Lưu cấu hình Dropbox'}
          </button>
          {saved && (
            <span className="settings-saved">
              <Icon name="check" size={13} /> Đã lưu
            </span>
          )}
        </div>
        {error && <div className="error">{error}</div>}
      </div>
    </details>
  )
}
