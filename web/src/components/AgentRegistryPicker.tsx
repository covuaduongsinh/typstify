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
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Không tải được danh sách trợ lý AI'))
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
      setError(err instanceof ApiError ? err.message : 'Không chọn được trợ lý AI')
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="settings-section agent-registry-picker">
      <h2>Chọn trợ lý AI</h2>
      <p className="settings-hint">
        Chọn một trợ lý từ danh sách ACP chính thức. Bấm "Dùng" để ghi đè cấu hình thủ công
        bên dưới.
      </p>

      <div className="agent-registry-row">
        <select value={selectedId} onChange={(e) => setSelectedId(e.target.value)}>
          <option value="">Chọn trợ lý…</option>
          {registry?.agents.map((a) => (
            <option key={a.id} value={a.id}>
              {a.name}
            </option>
          ))}
        </select>
        <button onClick={use} disabled={!selectedId || busy}>
          {busy ? 'Đang chọn…' : 'Dùng'}
        </button>
      </div>

      {entry && (
        <div className="agent-registry-detail">
          <div className="agent-registry-detail-header">
            {entry.name} <span className="agent-registry-version">{entry.version}</span>
          </div>
          <p>{entry.description}</p>
          {entry.license && <div>Giấy phép: {entry.license}</div>}
          {entry.authors?.length > 0 && <div>Tác giả: {entry.authors.join(', ')}</div>}
        </div>
      )}

      {error && <div className="error">{error}</div>}

      <p className="terms-warning">
        Các gói thuê bao (Claude Pro/Max, Google AI, ChatGPT Plus/Pro…) dành cho việc dùng cá
        nhân qua CLI của chính nhà cung cấp. Chạy tự động liên tục trong nền, hoặc dùng chung một
        tài khoản cho nhiều người, nằm ngoài phạm vi đó và nhà cung cấp có thể khóa tài khoản. Nếu
        cần chạy nền tự động, hãy cấu hình trợ lý bằng API key riêng (ô Env bên dưới) thay vì đăng
        nhập bằng gói thuê bao.
      </p>
    </section>
  )
}
