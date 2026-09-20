// Mirrors service/settings.AgentRegistry / AgentEntry (server/agent_api.go).
export interface AgentRegistryEntry {
  id: string
  name: string
  version: string
  description: string
  repository?: string
  website?: string
  authors: string[]
  license: string
  icon: string
}

export interface AgentRegistry {
  version: string
  agents: AgentRegistryEntry[]
}
