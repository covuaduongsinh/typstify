import { useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { AgentRegistry, AgentRegistryEntry } from '../api/agentRegistry'
import type { AgentSettings } from '../api/types'

/** AgentRegistryPicker lets the user pick an AI agent from the official ACP
 * registry (https://agentclientprotocol.com/get-started/registry) --
 * GET /api/agent/registry / POST /api/agent/select (server/agent_api.go),
 * mirroring the desktop app's ui/settings/agent.go picker. */
export function AgentRegistryPicker({ onSelected }: { onSelected: (settings: AgentSettings) => void }) {
  const [registry, setRegistry] = useState<AgentRegistry | null>(null)
  const [selectedId, setSelectedId] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    api
      .get<AgentRegistry>('/api/agent/registry')
      .then(setRegistry)
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Failed to load agent registry'))
  }, [])

  const entry: AgentRegistryEntry | undefined = registry?.agents.find((a) => a.id === selectedId)

  const use = async () => {
    if (!selectedId) return
    setBusy(true)
    setError(null)
    try {
      const settings = await api.post<AgentSettings>('/api/agent/select', { agentId: selectedId })
      onSelected(settings)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to select agent')
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="settings-section agent-registry-picker">
      <h2>Agent Registry</h2>
      <p className="settings-hint">
        Select an agent from the official ACP registry. Click "Use" to overwrite the manual
        configuration below.
      </p>

      <div className="agent-registry-row">
        <select value={selectedId} onChange={(e) => setSelectedId(e.target.value)}>
          <option value="">Select an agent…</option>
          {registry?.agents.map((a) => (
            <option key={a.id} value={a.id}>
              {a.name}
            </option>
          ))}
        </select>
        <button onClick={use} disabled={!selectedId || busy}>
          {busy ? 'Selecting…' : 'Use'}
        </button>
      </div>

      {entry && (
        <div className="agent-registry-detail">
          <div className="agent-registry-detail-header">
            {entry.name} <span className="agent-registry-version">{entry.version}</span>
          </div>
          <p>{entry.description}</p>
          {entry.license && <div>License: {entry.license}</div>}
          {entry.authors?.length > 0 && <div>Authors: {entry.authors.join(', ')}</div>}
        </div>
      )}

      {error && <div className="error">{error}</div>}

      <p className="terms-warning">
        Subscription plans (Claude Pro/Max, Google AI, ChatGPT Plus/Pro…) are meant for normal
        personal use of that provider's own CLI. Running it unattended in a background loop, or
        sharing one account across multiple people, falls outside that and providers have
        suspended accounts for it. If you need unattended background use, configure that agent
        with its own API key instead (Env field below) rather than a subscription login.
      </p>
    </section>
  )
}
