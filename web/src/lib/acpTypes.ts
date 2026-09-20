// Mirrors the JSON wire shapes of github.com/coder/acp-go-sdk's ContentBlock
// (discriminated union) and the session update/permission types Typstify's
// agent package re-exports (agent/session.go). Only the "text" content
// variant is rendered for v1; image/audio/resource content falls back to a
// placeholder.

export interface ContentBlock {
  type: string
  text?: string
}

export interface MessageChunk {
  content: ContentBlock
}

export type ToolCallStatus = 'pending' | 'in_progress' | 'completed' | 'failed'

export interface ToolCallContentItem {
  type: string
  content?: ContentBlock
}

export interface ToolCall {
  toolCallId: string
  title: string
  status?: ToolCallStatus
  kind?: string
  content?: ToolCallContentItem[]
}

export interface PermissionOption {
  optionId: string
  name: string
  kind: string
}

export interface PermissionRequest {
  sessionId: string
  options: PermissionOption[]
  toolCall: ToolCall
}

// Mirrors acp.AuthMethod's flat wire shape (server/agent_ws.go's
// agentAuthRequiredData). type is absent for agent-managed methods (the
// agent handles login itself, typically by printing a URL/code to stderr --
// see GET /api/console), "env_var", or "terminal".
export interface AuthMethod {
  id: string
  name: string
  description?: string
  type?: 'env_var' | 'terminal'
  link?: string
  vars?: Array<{ name: string; label?: string }>
}

export interface AuthRequiredData {
  agentName: string
  authMethods: AuthMethod[]
}

// Mirrors acp.SessionConfigOption's flat wire shape (server/agent_ws.go
// relays the ACP-SDK type as-is, its custom MarshalJSON flattens the
// Select/Boolean variant with a "type" discriminator). Only "select" is
// rendered (e.g. the model picker); other types/categories are ignored.
export interface SessionConfigSelectOption {
  name: string
  value: string
  description?: string
}

export interface SessionConfigOption {
  type: 'select' | 'boolean'
  id: string
  name: string
  category?: string
  currentValue?: string | boolean
  // Only the "ungrouped" shape (a flat array) is supported; a "grouped"
  // Options payload would serialize as a different (object) shape and is
  // just ignored by contentBlockText-style callers filtering on Array.isArray.
  options?: SessionConfigSelectOption[]
  description?: string
}

// Outgoing image attachment for AgentClient.prompt() -- base64 data (no
// "data:...;base64," prefix) plus its MIME type, matching
// server/agent_ws.go's imageAttachment and acp.ImageBlock.
export interface ImageAttachment {
  data: string
  mimeType: string
}

export function contentBlockText(block: ContentBlock | undefined): string {
  if (!block) return ''
  if (block.type === 'text') return block.text ?? ''
  return `[${block.type} content]`
}
