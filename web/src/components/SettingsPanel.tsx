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

  if (!value) return <section className="settings-section">Loading {title}…</section>

  const save = async () => {
    setError(null)
    try {
      const updated = await api.putJson<T>(path, value)
      setValue(updated)
      setSaved(true)
      setTimeout(() => setSaved(false), 1500)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to save settings')
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
      <button onClick={save}>Save</button>
      {saved && <span className="settings-saved">Saved</span>}
      {error && <div className="error">{error}</div>}
    </section>
  )
}

export function SettingsPanel() {
  const [agentSettingsVersion, setAgentSettingsVersion] = useState(0)

  return (
    <div className="settings-panel">
      <Section<GeneralSettings>
        title="General"
        path="/api/settings/general"
        fields={[
          { key: 'language', label: 'Language' },
          { key: 'theme', label: 'Theme' },
          { key: 'externalTypst', label: 'typst executable path (optional)' },
          { key: 'externalTinymist', label: 'tinymist executable path (optional)' },
        ]}
      />
      <Section<TypstSettings>
        title="Typst"
        path="/api/settings/typst"
        fields={[
          { key: 'cacheDir', label: 'Package cache dir' },
          { key: 'localPkgDir', label: 'Local package dir' },
          { key: 'extraFontPath', label: 'Extra font path' },
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
        title="AI Agent (manual/advanced)"
        path="/api/settings/agent"
        fields={[
          { key: 'agentName', label: 'Name' },
          { key: 'cmd', label: 'Command' },
          { key: 'args', label: 'Args (space-separated)' },
          { key: 'env', label: 'Env (KEY=value, space-separated)' },
        ]}
      />
    </div>
  )
}
