import { wsUrl } from '../api/client'

// Wire format mirrors server/lsp_ws.go's lspClientMessage/lspServerMessage.

interface OutgoingMessage {
  type: string
  id?: string
  path?: string
  content?: string
  line?: number
  character?: number
}

interface IncomingMessage {
  type: string
  id?: string
  path?: string
  items?: unknown
  hover?: { contents?: string } | null
  symbols?: unknown
  diagnostics?: LspDiagnostic[]
  message?: string
}

export interface LspDiagnostic {
  range: {
    start: { line: number; character: number }
    end: { line: number; character: number }
  }
  severity?: number
  source?: string
  message: string
}

export interface CompletionItem {
  label: string
  kind?: number
  detail?: string
  insertText?: string
}

type DiagnosticsListener = (path: string, diagnostics: LspDiagnostic[]) => void

let nextId = 1

const REQUEST_TIMEOUT_MS = 10_000
const RECONNECT_MIN_MS = 1_000
const RECONNECT_MAX_MS = 15_000

/** LspClient bridges one CodeMirror editor session to /ws/lsp. If the
 * socket drops (server restart, proxy idle timeout, network), it
 * reconnects with backoff and notifies onReconnect listeners so the editor
 * can re-open its document on the new server-side session. */
export class LspClient {
  private ws: WebSocket | null = null
  private pending = new Map<
    string,
    { resolve: (v: unknown) => void; reject: (e: Error) => void; timer: number }
  >()
  private diagnosticsListeners = new Set<DiagnosticsListener>()
  private reconnectListeners = new Set<() => void>()
  private openPromise!: Promise<void>
  private closed = false
  private everOpened = false
  private reconnectDelay = RECONNECT_MIN_MS
  private reconnectTimer: number | undefined

  constructor() {
    this.connect()
  }

  private connect() {
    this.openPromise = new Promise<void>((resolve, reject) => {
      const ws = new WebSocket(wsUrl('/ws/lsp'))
      this.ws = ws
      ws.addEventListener('open', () => {
        const isReconnect = this.everOpened
        this.everOpened = true
        this.reconnectDelay = RECONNECT_MIN_MS
        resolve()
        if (isReconnect) for (const l of this.reconnectListeners) l()
      })
      ws.addEventListener('error', () => reject(new Error('LSP WebSocket connection failed')))
      ws.addEventListener('message', (ev) => this.onMessage(ev))
      ws.addEventListener('close', () => {
        reject(new Error('LSP connection closed'))
        this.rejectAllPending('LSP connection closed')
        this.scheduleReconnect()
      })
    })
    // Failures surface through request()/send(); never as an unhandled rejection.
    this.openPromise.catch(() => {})
  }

  private scheduleReconnect() {
    if (this.closed) return
    window.clearTimeout(this.reconnectTimer)
    this.reconnectTimer = window.setTimeout(() => this.connect(), this.reconnectDelay)
    this.reconnectDelay = Math.min(this.reconnectDelay * 2, RECONNECT_MAX_MS)
  }

  private onMessage(ev: MessageEvent) {
    let msg: IncomingMessage
    try {
      msg = JSON.parse(ev.data as string)
    } catch {
      return
    }

    if (msg.type === 'diagnostics' && msg.path) {
      for (const listener of this.diagnosticsListeners) listener(msg.path, msg.diagnostics ?? [])
      return
    }

    if (msg.type === 'error' && !msg.id) {
      console.error('lsp error:', msg.message)
      return
    }

    if (!msg.id) return
    const pending = this.pending.get(msg.id)
    if (!pending) return
    this.pending.delete(msg.id)
    window.clearTimeout(pending.timer)

    if (msg.type === 'error') {
      pending.reject(new Error(msg.message ?? 'LSP request failed'))
    } else {
      pending.resolve(msg)
    }
  }

  private rejectAllPending(reason: string) {
    for (const p of this.pending.values()) {
      window.clearTimeout(p.timer)
      p.reject(new Error(reason))
    }
    this.pending.clear()
  }

  /** Sends once the socket is open. Notifications sent while disconnected
   * are dropped: on reconnect the editor re-sends the full document. */
  private async send(msg: OutgoingMessage): Promise<boolean> {
    try {
      await this.openPromise
    } catch {
      return false
    }
    if (this.ws?.readyState !== WebSocket.OPEN) return false
    this.ws.send(JSON.stringify(msg))
    return true
  }

  private async request<T>(msg: OutgoingMessage): Promise<T> {
    const id = String(nextId++)
    if (!(await this.send({ ...msg, id }))) throw new Error('LSP not connected')
    return new Promise<T>((resolve, reject) => {
      const timer = window.setTimeout(() => {
        this.pending.delete(id)
        reject(new Error('LSP request timed out'))
      }, REQUEST_TIMEOUT_MS)
      this.pending.set(id, { resolve: resolve as (v: unknown) => void, reject, timer })
    })
  }

  onDiagnostics(listener: DiagnosticsListener): () => void {
    this.diagnosticsListeners.add(listener)
    return () => this.diagnosticsListeners.delete(listener)
  }

  /** Called after the socket reconnects (not on the first connect). */
  onReconnect(listener: () => void): () => void {
    this.reconnectListeners.add(listener)
    return () => this.reconnectListeners.delete(listener)
  }

  didOpen(path: string, content: string) {
    void this.send({ type: 'didOpen', path, content })
  }

  didChange(path: string, content: string) {
    void this.send({ type: 'didChange', path, content })
  }

  didClose(path: string) {
    void this.send({ type: 'didClose', path })
  }

  didSave(path: string) {
    void this.send({ type: 'didSave', path })
  }

  async complete(path: string, line: number, character: number): Promise<CompletionItem[]> {
    const res = await this.request<IncomingMessage>({ type: 'complete', path, line, character })
    const items = (res.items as { items?: CompletionItem[] } | CompletionItem[] | undefined) ?? []
    return Array.isArray(items) ? items : (items.items ?? [])
  }

  async hover(path: string, line: number, character: number): Promise<string | null> {
    const res = await this.request<IncomingMessage>({ type: 'hover', path, line, character })
    return res.hover?.contents ?? null
  }

  close() {
    this.closed = true
    window.clearTimeout(this.reconnectTimer)
    this.rejectAllPending('LSP client closed')
    this.ws?.close()
  }
}
