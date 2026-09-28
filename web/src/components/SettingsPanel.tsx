import { useEffect, useRef, useState, type ReactNode } from 'react'
import { ApiError, api } from '../api/client'
import type { AgentSettings, FontFileInfo, GeneralSettings, LspSettings, TypstSettings } from '../api/types'
import { describeCombo, effectiveKey, keyComboFromEvent, SHORTCUT_ACTIONS, useShortcuts } from '../lib/shortcuts'
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

      <AccountSettingsSection />

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

      <FontsSection />

      <ShortcutsSection />

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

      <LspSettingsSection />
    </div>
  )
}

// LspSettings mixes int-as-bool (EnableLSPLogs/EnablePowerSaving, kept as 0/1
// for compatibility with the desktop app's Gio widget.Bool save path) and a
// real bool (EnablePartialRenderPreview) -- the generic string-keyed
// Section<T> above can't express either without corrupting the PUT payload
// (server/settings_api.go decodes strictly into the typed Go struct), so
// this section is hand-written instead.
function LspSettingsSection() {
  const [value, setValue] = useState<LspSettings | null>(null)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    api
      .get<LspSettings>('/api/settings/lsp')
      .then(setValue)
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Không tải được cài đặt'))
  }, [])

  const save = async () => {
    if (!value) return
    setError(null)
    setBusy(true)
    try {
      const updated = await api.putJson<LspSettings>('/api/settings/lsp', value)
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
    <details className="settings-section">
      <summary>
        <Icon name="chevron-right" size={14} className="settings-caret" />
        Nâng cao: LSP
      </summary>
      <div className="settings-section-body">
        {!value && !error && <p className="settings-hint">Đang tải…</p>}
        {value && (
          <>
            <div className="checkbox-row" style={{ flexDirection: 'column', alignItems: 'flex-start', gap: 2 }}>
              <label className="checkbox-label">
                <input
                  type="checkbox"
                  checked={value.enablePowerSaving !== 0}
                  onChange={(e) => setValue({ ...value, enablePowerSaving: e.target.checked ? 1 : 0 })}
                />
                <span>Chế độ tiết kiệm tài nguyên</span>
              </label>
              <small className="settings-hint">
                Khi bật, LSP chỉ kiểm tra cú pháp và gợi ý cơ bản; chẩn đoán lỗi và xem trước ở chế độ "Đồng bộ chính
                xác" sẽ không hoạt động. Cần mở lại tệp để áp dụng.
              </small>
            </div>

            <div className="checkbox-row" style={{ flexDirection: 'column', alignItems: 'flex-start', gap: 2, marginTop: 10 }}>
              <label className="checkbox-label">
                <input
                  type="checkbox"
                  checked={value.enablePartialRenderPreview}
                  onChange={(e) => setValue({ ...value, enablePartialRenderPreview: e.target.checked })}
                />
                <span>Xem trước từng phần (chế độ "Đồng bộ chính xác")</span>
              </label>
              <small className="settings-hint">
                Chỉ dựng các trang trong khung nhìn thay vì toàn bộ tài liệu — cải thiện tốc độ với tài liệu dài.
              </small>
            </div>

            <div className="checkbox-row" style={{ flexDirection: 'column', alignItems: 'flex-start', gap: 2, marginTop: 10 }}>
              <label className="checkbox-label">
                <input
                  type="checkbox"
                  checked={value.enableLspLogs !== 0}
                  onChange={(e) => setValue({ ...value, enableLspLogs: e.target.checked ? 1 : 0 })}
                />
                <span>Ghi log gỡ lỗi LSP</span>
              </label>
              <small className="settings-hint">
                Ghi log chi tiết của LSP (tinymist) ra log máy chủ — hữu ích khi cần báo lỗi. Cần khởi động lại máy
                chủ để áp dụng.
              </small>
            </div>

            <div className="settings-actions" style={{ marginTop: 12 }}>
              <button className="btn-primary" onClick={save} disabled={busy}>
                {busy ? 'Đang lưu…' : 'Lưu'}
              </button>
              {saved && (
                <span className="settings-saved">
                  <Icon name="check" size={13} /> Đã lưu
                </span>
              )}
            </div>
          </>
        )}
        {error && <div className="error">{error}</div>}
      </div>
    </details>
  )
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

// FontsSection uploads/lists/deletes font files (server/fonts_api.go),
// which land in whatever directory the "Thư mục font bổ sung" field above
// already points to (or a managed default the server creates the first
// time you upload one) -- so this is a drag-and-drop-free alternative to
// manually copying files into that folder, not a separate mechanism.
function FontsSection() {
  const [fonts, setFonts] = useState<FontFileInfo[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement | null>(null)

  const load = () => {
    api
      .get<FontFileInfo[]>('/api/settings/fonts')
      .then(setFonts)
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Không tải được danh sách font'))
  }

  useEffect(() => {
    load()
  }, [])

  const upload = async (files: FileList | null) => {
    if (!files || files.length === 0) return
    setError(null)
    setUploading(true)
    try {
      for (const file of Array.from(files)) {
        await api.postBinary(`/api/settings/fonts?filename=${encodeURIComponent(file.name)}`, file)
      }
      load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Tải lên font thất bại')
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const remove = async (name: string) => {
    setError(null)
    try {
      await api.del(`/api/settings/fonts/${encodeURIComponent(name)}`)
      load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Xóa font thất bại')
    }
  }

  return (
    <details className="settings-section">
      <summary>
        <Icon name="chevron-right" size={14} className="settings-caret" />
        Font đã tải lên
      </summary>
      <div className="settings-section-body">
        <p className="settings-hint">
          Tải lên font .ttf/.otf/.ttc (ví dụ font tiếng Việt hoặc ký hiệu quân cờ). Mở lại tài liệu hoặc bấm biên dịch
          lại để bản xem trước dùng font mới.
        </p>

        {!fonts && !error && <p className="settings-hint">Đang tải…</p>}
        {fonts && fonts.length === 0 && <p className="settings-hint">Chưa có font nào được tải lên.</p>}
        {fonts && fonts.length > 0 && (
          <ul className="fonts-list">
            {fonts.map((f) => (
              <li key={f.name} className="fonts-list-item">
                <span className="fonts-list-item-name">{f.name}</span>
                <span className="fonts-list-item-size">{formatFileSize(f.size)}</span>
                <button className="btn-ghost btn-sm" title="Xóa font" onClick={() => remove(f.name)}>
                  <Icon name="x" size={13} />
                </button>
              </li>
            ))}
          </ul>
        )}

        <div className="settings-actions">
          <input
            ref={fileInputRef}
            type="file"
            accept=".ttf,.otf,.ttc"
            multiple
            style={{ display: 'none' }}
            onChange={(e) => upload(e.target.files)}
          />
          <button className="btn-primary" onClick={() => fileInputRef.current?.click()} disabled={uploading}>
            {uploading ? 'Đang tải lên…' : 'Tải font lên'}
          </button>
        </div>
        {error && <div className="error">{error}</div>}
      </div>
    </details>
  )
}

// ShortcutsSection lets the user remap the handful of app-level chess
// shortcuts (lib/shortcuts.ts) -- everything else in the app is either
// CodeMirror's own bundled keymap or a local widget convention (Escape
// closes a modal) that isn't meaningfully "app-customizable", so this list
// is deliberately short rather than a generic keymap editor.
function ShortcutsSection() {
  const { overrides, setShortcut, resetShortcut } = useShortcuts()
  const [listeningFor, setListeningFor] = useState<string | null>(null)
  const [conflict, setConflict] = useState<string | null>(null)

  const captureNext = (e: React.KeyboardEvent, actionId: string) => {
    e.preventDefault()
    if (e.key === 'Escape') {
      setListeningFor(null)
      return
    }
    const combo = keyComboFromEvent(e)
    if (!combo) return // bare modifier, or missing Ctrl/Cmd -- keep listening

    const takenBy = SHORTCUT_ACTIONS.find(
      (a) => a.id !== actionId && effectiveKey(a, overrides) === combo,
    )
    if (takenBy) {
      setConflict(`"${describeCombo(combo)}" đang dùng cho "${takenBy.label}". Chọn tổ hợp khác.`)
      return
    }

    setConflict(null)
    setShortcut(actionId, combo)
    setListeningFor(null)
  }

  return (
    <details className="settings-section">
      <summary>
        <Icon name="chevron-right" size={14} className="settings-caret" />
        Phím tắt
      </summary>
      <div className="settings-section-body">
        <p className="settings-hint">
          Phím tắt cho các thao tác cờ vua hay dùng. Luôn cần giữ Ctrl (⌘ trên Mac) để không xung đột với gõ văn bản
          bình thường.
        </p>

        <ul className="shortcuts-list">
          {SHORTCUT_ACTIONS.map((action) => {
            const listening = listeningFor === action.id
            const isDefault = !(action.id in overrides)
            return (
              <li key={action.id} className="shortcuts-list-item">
                <span className="shortcuts-list-item-label">{action.label}</span>
                {listening ? (
                  <input
                    autoFocus
                    className="shortcuts-capture-input"
                    readOnly
                    value="Nhấn tổ hợp phím mới… (Esc để hủy)"
                    onKeyDown={(e) => captureNext(e, action.id)}
                    onBlur={() => setListeningFor(null)}
                  />
                ) : (
                  <>
                    <kbd className="shortcuts-combo">{describeCombo(effectiveKey(action, overrides))}</kbd>
                    <button
                      className="btn-ghost btn-sm"
                      onClick={() => {
                        setConflict(null)
                        setListeningFor(action.id)
                      }}
                    >
                      Đổi
                    </button>
                    {!isDefault && (
                      <button className="btn-ghost btn-sm" title="Khôi phục mặc định" onClick={() => resetShortcut(action.id)}>
                        <Icon name="refresh" size={12} />
                      </button>
                    )}
                  </>
                )}
              </li>
            )
          })}
        </ul>
        {conflict && <div className="error">{conflict}</div>}
      </div>
    </details>
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

function AccountSettingsSection() {
  const [authStatus, setAuthStatus] = useState<{
    authRequired: boolean
    authenticated: boolean
    username?: string
    displayName?: string
  } | null>(null)
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadStatus = () => {
    api
      .get<{
        authRequired: boolean
        authenticated: boolean
        username?: string
        displayName?: string
      }>('/api/auth/status')
      .then(setAuthStatus)
      .catch(() => {})
  }

  useEffect(() => {
    loadStatus()
  }, [])

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    if (newPassword !== confirmPassword) {
      setError('Mật khẩu mới và mật khẩu xác nhận không khớp')
      return
    }

    if (newPassword.length < 4) {
      setError('Mật khẩu mới phải có ít nhất 4 ký tự')
      return
    }

    setBusy(true)
    try {
      await api.post('/api/auth/change-password', {
        currentPassword,
        newPassword,
      })
      setSaved(true)
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
      setTimeout(() => setSaved(false), 2500)
    } catch (err: any) {
      setError(err instanceof ApiError ? err.message : 'Không đổi được mật khẩu')
    } finally {
      setBusy(false)
    }
  }

  const handleLogout = async () => {
    if (!confirm('Bạn có chắc chắn muốn đăng xuất?')) return
    try {
      await api.post('/api/auth/logout', {})
    } catch {
    } finally {
      window.location.reload()
    }
  }

  return (
    <details className="settings-section" open>
      <summary>
        <Icon name="chevron-right" size={14} className="settings-caret" />
        Tài khoản & Bảo mật
      </summary>
      <div className="settings-section-body">
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
          <div>
            <div style={{ fontWeight: 600, fontSize: 14 }}>
              {authStatus?.displayName || authStatus?.username || 'Người dùng'}
            </div>
            {authStatus?.username && (
              <div style={{ fontSize: 12, color: 'var(--text-dim)' }}>
                Tài khoản: <code>{authStatus.username}</code>
              </div>
            )}
          </div>
          <button className="btn-secondary btn-sm" onClick={handleLogout} style={{ color: 'var(--error)' }}>
            Đăng xuất
          </button>
        </div>

        <form onSubmit={handleChangePassword} style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <div style={{ fontWeight: 500, fontSize: 13, marginTop: 4 }}>Đổi mật khẩu:</div>
          <label className="settings-field">
            <span>Mật khẩu hiện tại</span>
            <input
              type="password"
              autoComplete="current-password"
              placeholder="Nhập mật khẩu đang dùng"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              required
            />
          </label>

          <div className="grid-2col">
            <label className="settings-field">
              <span>Mật khẩu mới</span>
              <input
                type="password"
                autoComplete="new-password"
                placeholder="Tối thiểu 4 ký tự"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                required
              />
            </label>
            <label className="settings-field">
              <span>Xác nhận mật khẩu mới</span>
              <input
                type="password"
                autoComplete="new-password"
                placeholder="Nhập lại mật khẩu mới"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
              />
            </label>
          </div>

          <div className="settings-actions" style={{ marginTop: 8 }}>
            <button
              type="submit"
              className="btn-primary"
              disabled={busy || !currentPassword || !newPassword || !confirmPassword}
            >
              {busy ? 'Đang cập nhật…' : 'Cập nhật mật khẩu'}
            </button>
            {saved && (
              <span className="settings-saved">
                <Icon name="check" size={13} /> Đã đổi mật khẩu thành công
              </span>
            )}
          </div>
          {error && <div className="error">{error}</div>}
        </form>
      </div>
    </details>
  )
}
