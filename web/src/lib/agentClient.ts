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
  // Remembers the last application-level {type:'error'} message the server
  // sent (if any) so the synthesized 'disconnected' message below can carry
  // it -- letting AgentChat.tsx tell "we know why this failed" apart from
  // "the socket just died with no explanation", instead of always showing
  // the same generic text on top of whatever specific error already showed.
  private lastErrorMessage: string | undefined
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
      if (msg.type === 'error' && msg.message) this.lastErrorMessage = msg.message
      for (const l of this.listeners) l(msg)
    })
    // Without this, a connection that dies for any reason (server
    // restart, network drop, something between browser and server closing
    // an idle/long-lived socket) leaves the UI showing stale "connected"/
    // "waiting" state forever -- observed live: the server had already
    // logged the session as closed while the chat panel kept showing
    // "Agent is thinking...". Surface it as a message like any other server
    // event so AgentChat.tsx can react (stop waiting, show reconnect UI)
    // instead of every caller needing its own ws.onclose listener.
    const notifyDisconnected = () => {
      for (const l of this.listeners) l({ type: 'disconnected', message: this.lastErrorMessage })
    }
    this.ws.addEventListener('close', notifyDisconnected)
    this.ws.addEventListener('error', notifyDisconnected)
  }

  on(listener: Listener): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private send(msg: Record<string, unknown>) {
    if (this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg))
    } else if (this.ws.readyState === WebSocket.CONNECTING) {
      this.ws.addEventListener('open', () => this.ws.send(JSON.stringify(msg)), { once: true })
    }
    // CLOSING/CLOSED: dropped. The 'close'/'error' listeners in the
    // constructor already notified callers via a "disconnected" message,
    // so there is nothing new to report here.
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

  /** Tell the server to retry starting the ACP session, after the browser
   * has driven POST /api/agent/auth/{methodId} to completion following an
   * "authRequired" message (see server/agent_ws.go's
   * startSessionOrRequireAuth). */
  retryAuth() {
    this.send({ type: 'retryAuth' })
  }

  close() {
    this.ws.close()
  }
}
