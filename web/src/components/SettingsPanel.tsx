import { useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { AgentSettings, GeneralSettings, LspSettings, TypstSettings } from '../api/types'
import { AgentRegistryPicker } from './AgentRegistryPicker'

function Section<T extends object>({
  title,
  path,
  fields,
}: {
  title: string
  path: string
  fields: Array<{ key: keyof T; label: string }>
}) {
  const [value, setValue] = useState<T | null>(null)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api.get<T>(path).then(setValue)
  }, [path])

  if (!value) return <section className="settings-section">Đang tải {title}…</section>

  const save = async () => {
    setError(null)
    try {
      const updated = await api.putJson<T>(path, value)
      setValue(updated)
      setSaved(true)
      setTimeout(() => setSaved(false), 1500)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Không lưu được cài đặt')
    }
  }

  return (
    <section className="settings-section">
      <h2>{title}</h2>
      {fields.map(({ key, label }) => (
        <label key={String(key)} className="settings-field">
          <span>{label}</span>
          <input
            value={String(value[key] ?? '')}
            onChange={(e) => setValue({ ...value, [key]: e.target.value } as T)}
          />
        </label>
      ))}
      <button onClick={save}>Lưu</button>
      {saved && <span className="settings-saved">Đã lưu</span>}
      {error && <div className="error">{error}</div>}
    </section>
  )
}

export function SettingsPanel() {
  const [agentSettingsVersion, setAgentSettingsVersion] = useState(0)

  return (
    <div className="settings-panel">
      <Section<GeneralSettings>
        title="Chung"
        path="/api/settings/general"
        fields={[
          { key: 'language', label: 'Ngôn ngữ' },
          { key: 'theme', label: 'Giao diện' },
          { key: 'externalTypst', label: 'Đường dẫn typst (tùy chọn)' },
          { key: 'externalTinymist', label: 'Đường dẫn tinymist (tùy chọn)' },
        ]}
      />
      <Section<TypstSettings>
        title="Typst"
        path="/api/settings/typst"
        fields={[
          { key: 'cacheDir', label: 'Thư mục cache gói' },
          { key: 'localPkgDir', label: 'Thư mục gói cục bộ' },
          { key: 'extraFontPath', label: 'Thư mục font bổ sung' },
        ]}
      />
      <Section<LspSettings>
        title="LSP"
        path="/api/settings/lsp"
        fields={[]}
      />
      <AgentRegistryPicker onSelected={() => setAgentSettingsVersion((v) => v + 1)} />
      <Section<AgentSettings>
        key={agentSettingsVersion}
        title="Trợ lý AI (thủ công / nâng cao)"
        path="/api/settings/agent"
        fields={[
          { key: 'agentName', label: 'Tên' },
          { key: 'cmd', label: 'Lệnh' },
          { key: 'args', label: 'Tham số (cách nhau bởi dấu cách)' },
          { key: 'env', label: 'Biến môi trường (KEY=value, cách nhau bởi dấu cách)' },
        ]}
      />
    </div>
  )
}
