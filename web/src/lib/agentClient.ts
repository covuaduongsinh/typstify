import { wsUrl } from '../api/client'

// Wire format mirrors server/agent_ws.go's agentClientMessage/
// agentServerMessage. `data` carries the underlying ACP payload as-is
// (agent.UserMessageChunk, agent.ToolCall, acp.RequestPermissionRequest, ...)
// since package agent already exchanges these as JSON over the Agent Client
// Protocol -- see https://agentclientprotocol.com for the field shapes.

export interface AgentServerMessage {
  type: string
  sessionId?: string
  data?: unknown
  message?: string
}

type Listener = (msg: AgentServerMessage) => void

/** AgentClient bridges one AI agent chat session to /ws/agent. */
export class AgentClient {
  private ws: WebSocket
  private listeners = new Set<Listener>()
  sessionId: string | null = null

  constructor() {
    this.ws = new WebSocket(wsUrl('/ws/agent'))
    this.ws.addEventListener('message', (ev) => {
      let msg: AgentServerMessage
      try {
        msg = JSON.parse(ev.data as string)
      } catch {
        return
      }
      if (msg.type === 'ready' && msg.sessionId) this.sessionId = msg.sessionId
      for (const l of this.listeners) l(msg)
    })
  }

  on(listener: Listener): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private send(msg: Record<string, unknown>) {
    if (this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg))
    } else {
      this.ws.addEventListener('open', () => this.ws.send(JSON.stringify(msg)), { once: true })
    }
  }

  prompt(text: string) {
    this.send({ type: 'prompt', text })
  }

  cancel() {
    this.send({ type: 'cancel' })
  }

  respondPermission(optionId: string) {
    this.send({ type: 'permissionResponse', optionId })
  }

  close() {
    this.ws.close()
  }
}
