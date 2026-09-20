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

export function contentBlockText(block: ContentBlock | undefined): string {
  if (!block) return ''
  if (block.type === 'text') return block.text ?? ''
  return `[${block.type} content]`
}
