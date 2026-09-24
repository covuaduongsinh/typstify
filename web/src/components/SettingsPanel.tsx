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

      <Section<LspSettings> title="Nâng cao: LSP" path="/api/settings/lsp" fields={[]} description="Chưa có tùy chọn nào cho bản web." />
    </div>
  )
}
