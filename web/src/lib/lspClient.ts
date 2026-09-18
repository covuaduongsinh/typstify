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

/** LspClient bridges one CodeMirror editor session to /ws/lsp. */
export class LspClient {
  private ws: WebSocket | null = null
  private pending = new Map<string, { resolve: (v: unknown) => void; reject: (e: Error) => void }>()
  private diagnosticsListeners = new Set<DiagnosticsListener>()
  private openPromise: Promise<void>

  constructor() {
    this.openPromise = new Promise((resolve, reject) => {
      const ws = new WebSocket(wsUrl('/ws/lsp'))
      this.ws = ws
      ws.addEventListener('open', () => resolve())
      ws.addEventListener('error', () => reject(new Error('LSP WebSocket connection failed')))
      ws.addEventListener('message', (ev) => this.onMessage(ev))
      ws.addEventListener('close', () => this.rejectAllPending('LSP connection closed'))
    })
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

    if (msg.type === 'error') {
      pending.reject(new Error(msg.message ?? 'LSP request failed'))
    } else {
      pending.resolve(msg)
    }
  }

  private rejectAllPending(reason: string) {
    for (const p of this.pending.values()) p.reject(new Error(reason))
    this.pending.clear()
  }

  private async send(msg: OutgoingMessage) {
    await this.openPromise
    this.ws?.send(JSON.stringify(msg))
  }

  private async request<T>(msg: OutgoingMessage): Promise<T> {
    const id = String(nextId++)
    await this.send({ ...msg, id })
    return new Promise<T>((resolve, reject) => {
      this.pending.set(id, { resolve: resolve as (v: unknown) => void, reject })
    })
  }

  onDiagnostics(listener: DiagnosticsListener): () => void {
    this.diagnosticsListeners.add(listener)
    return () => this.diagnosticsListeners.delete(listener)
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
    this.ws?.close()
  }
}
