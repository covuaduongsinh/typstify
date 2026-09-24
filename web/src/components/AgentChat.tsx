import { useEffect, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import { api } from '../api/client'
import { AgentClient } from '../lib/agentClient'
import {
  contentBlockText,
  type AuthRequiredData,
  type ImageAttachment,
  type MessageChunk,
  type PermissionRequest,
  type SessionConfigOption,
  type ToolCall,
} from '../lib/acpTypes'
import { AuthCard } from './AuthCard'
import { QuickActions } from './QuickActions'

// How close to the bottom (px) the user has to be for new messages to
// auto-scroll the log. Prevents yanking someone back down mid-stream while
// they've scrolled up to reread earlier output.
const AUTO_SCROLL_THRESHOLD = 80

type TextEntryKind = 'user' | 'agent' | 'thought' | 'plan' | 'error'

type ChatEntry =
  | { kind: TextEntryKind; id: string; text: string }
  | { kind: 'tool'; id: string; toolCallId: string; title: string; status: string }

function isTextEntry(e: ChatEntry): e is { kind: TextEntryKind; id: string; text: string } {
  return e.kind !== 'tool'
}

let nextEntryId = 1

export function AgentChat({ projectPath }: { projectPath: string }) {
  const [entries, setEntries] = useState<ChatEntry[]>([])
  const [input, setInput] = useState('')
  const [connected, setConnected] = useState(false)
  const [waiting, setWaiting] = useState(false)
  const [waitingLong, setWaitingLong] = useState(false)
  const [pendingPermission, setPendingPermission] = useState<PermissionRequest | null>(null)
  const [authRequired, setAuthRequired] = useState<AuthRequiredData | null>(null)
  const [configOptions, setConfigOptions] = useState<SessionConfigOption[]>([])
  const [pendingImages, setPendingImages] = useState<ImageAttachment[]>([])
  const [retryToken, setRetryToken] = useState(0)
  const [hasDisconnectedOnce, setHasDisconnectedOnce] = useState(false)
  const clientRef = useRef<AgentClient | null>(null)
  const streamingAgentId = useRef<string | null>(null)
  const streamingThoughtId = useRef<string | null>(null)
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const prevProjectPathRef = useRef<string | null>(null)

  const append = (entry: ChatEntry) => setEntries((prev) => [...prev, entry])

  const appendOrGrowText = (kind: 'agent' | 'thought', text: string, ref: React.MutableRefObject<string | null>) => {
    setEntries((prev) => {
      if (ref.current) {
        const idx = prev.findIndex((e) => e.id === ref.current)
        const existing = idx >= 0 ? prev[idx] : undefined
        if (existing && isTextEntry(existing)) {
          const copy = [...prev]
          copy[idx] = { ...existing, text: existing.text + text }
          return copy
        }
      }
      const id = `e${nextEntryId++}`
      ref.current = id
      return [...prev, { kind, id, text }]
    })
  }

  useEffect(() => {
    // Only wipe the transcript when the open project actually changed --
    // not when this effect re-ran because the user clicked "Retry
    // connection" (retryToken), so a manual retry keeps the conversation
    // visible instead of silently discarding it.
    if (prevProjectPathRef.current !== projectPath) {
      setEntries([])
      setHasDisconnectedOnce(false)
      prevProjectPathRef.current = projectPath
    }
    setWaiting(false)
    const client = new AgentClient()
    clientRef.current = client

    const unsubscribe = client.on((msg) => {
      switch (msg.type) {
        case 'ready':
          setConnected(true)
          setAuthRequired(null)
          break
        case 'authRequired':
          setAuthRequired(msg.data as AuthRequiredData)
          break
        case 'configOptions':
          setConfigOptions(msg.data as SessionConfigOption[])
          break
        case 'userMessage':
          // Most agents don't echo the user's own prompt back (observed:
          // Antigravity never sends this, Claude Code does) -- the client
          // already renders its own copy optimistically in submit()/
          // sendPrompt() below, so this is just a streaming-state reset,
          // never a second render of the same text.
          streamingAgentId.current = null
          streamingThoughtId.current = null
          break
        case 'agentMessage':
          setWaiting(false)
          streamingThoughtId.current = null
          appendOrGrowText('agent', contentBlockText((msg.data as MessageChunk).content), streamingAgentId)
          break
        case 'agentThought':
          setWaiting(false)
          streamingAgentId.current = null
          appendOrGrowText('thought', contentBlockText((msg.data as MessageChunk).content), streamingThoughtId)
          break
        case 'toolCall': {
          setWaiting(false)
          const tc = msg.data as ToolCall
          streamingAgentId.current = null
          streamingThoughtId.current = null
          append({ kind: 'tool', id: `e${nextEntryId++}`, toolCallId: tc.toolCallId, title: tc.title, status: tc.status ?? 'pending' })
          break
        }
        case 'toolCallUpdate': {
          const tc = msg.data as Partial<ToolCall> & { toolCallId: string }
          setEntries((prev) =>
            prev.map((e) =>
              e.kind === 'tool' && e.toolCallId === tc.toolCallId
                ? { ...e, title: tc.title ?? e.title, status: tc.status ?? e.status }
                : e,
            ),
          )
          break
        }
        case 'plan':
          setWaiting(false)
          append({ kind: 'plan', id: `e${nextEntryId++}`, text: 'Đã cập nhật kế hoạch' })
          break
        case 'permissionRequest':
          setWaiting(false)
          setPendingPermission(msg.data as PermissionRequest)
          break
        case 'turnEnd':
          setWaiting(false)
          streamingAgentId.current = null
          streamingThoughtId.current = null
          break
        case 'error':
          setWaiting(false)
          append({ kind: 'error', id: `e${nextEntryId++}`, text: msg.message ?? 'Lỗi không xác định từ trợ lý AI' })
          break
        case 'disconnected':
          setConnected(false)
          setWaiting(false)
          setHasDisconnectedOnce(true)
          setConfigOptions([])
          streamingAgentId.current = null
          streamingThoughtId.current = null
          // If the server already sent a specific {type:'error'} message just
          // before the socket closed, that error is already its own chat
          // bubble (the 'error' case above) -- don't stack a second, generic
          // one under it. Only fall back to the generic text when the socket
          // died with no explanation at all (e.g. a pure network drop).
          if (!msg.message) {
            append({
              kind: 'error',
              id: `e${nextEntryId++}`,
              text: 'Mất kết nối với trợ lý AI. Bấm "Kết nối lại" để mở phiên mới.',
            })
          }
          break
      }
    })

    return () => {
      unsubscribe()
      client.close()
      setConnected(false)
    }
  }, [projectPath, retryToken])

  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    const distanceFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    if (distanceFromBottom < AUTO_SCROLL_THRESHOLD) {
      el.scrollTo({ top: el.scrollHeight })
    }
  }, [entries])

  // A simple text reply streams back within ~15s (measured against a real
  // agent); a real editing task can legitimately take longer. Past 30s,
  // stop implying "any second now" and surface that this is unusual so the
  // user can decide to keep waiting or Cancel, instead of guessing forever.
  useEffect(() => {
    if (!waiting) {
      setWaitingLong(false)
      return
    }
    const timer = window.setTimeout(() => setWaitingLong(true), 30000)
    return () => window.clearTimeout(timer)
  }, [waiting])

  const sendPrompt = (text: string, images: ImageAttachment[] = []) => {
    const trimmed = text.trim()
    if ((!trimmed && images.length === 0) || !clientRef.current) return
    append({
      kind: 'user',
      id: `e${nextEntryId++}`,
      text: trimmed || `[${images.length} ảnh]`,
    })
    setWaiting(true)
    clientRef.current.prompt(trimmed, images)
  }

  const submit = () => {
    const text = input.trim()
    if (!text && pendingImages.length === 0) return
    sendPrompt(text, pendingImages)
    setInput('')
    setPendingImages([])
  }

  // Reads image items out of a paste event's clipboard data (screenshots,
  // copied images) as base64 -- text paste is left to the textarea's own
  // default behavior. Mirrors the desktop client's clipboard.FmtImage path
  // (agent/view/inputbox.go) so both surfaces support the same workflow.
  const handlePaste = (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
    const items = e.clipboardData?.items
    if (!items) return
    const imageItems = Array.from(items).filter((item) => item.type.startsWith('image/'))
    if (imageItems.length === 0) return
    e.preventDefault()
    for (const item of imageItems) {
      const file = item.getAsFile()
      if (!file) continue
      const mimeType = item.type
      const reader = new FileReader()
      reader.onload = () => {
        const dataUrl = reader.result as string
        const base64 = dataUrl.slice(dataUrl.indexOf(',') + 1)
        setPendingImages((prev) => [...prev, { data: base64, mimeType }])
      }
      reader.readAsDataURL(file)
    }
  }

  const respond = (optionId: string) => {
    clientRef.current?.respondPermission(optionId)
    setPendingPermission(null)
  }

  const cancelTurn = () => {
    clientRef.current?.cancel()
    setWaiting(false)
  }

  // Applies a config change (model/mode/etc.) to the live session AND saves
  // it as the default for future sessions, so picking "Sonnet 5" once
  // doesn't need repeating every time a new session spawns (server side:
  // agent_ws.go's applyPreferredConfig, driven by this same endpoint).
  const changeConfigOption = (configId: string, value: string) => {
    clientRef.current?.setConfigOption(configId, value)
    void api.post('/api/agent/preferred-config', { configId, value }).catch(() => {
      // Best-effort -- the live change above already applied; only the
      // "remember this for next time" part failed, nothing to surface here.
    })
  }

  if (authRequired) {
    return (
      <div className="agent-chat">
        <AuthCard
          data={authRequired}
          onDone={() => {
            setAuthRequired(null)
            clientRef.current?.retryAuth()
          }}
        />
      </div>
    )
  }

  return (
    <div className="agent-chat">
      <div className="agent-chat-status">
        {connected ? 'Trợ lý AI đã sẵn sàng' : 'Đang kết nối trợ lý AI…'}
        {!connected && hasDisconnectedOnce && (
          <button className="chat-retry-btn" onClick={() => setRetryToken((n) => n + 1)}>
            Kết nối lại
          </button>
        )}
        {connected &&
          configOptions
            .filter((opt) => opt.type === 'select' && Array.isArray(opt.options))
            .map((opt) => (
              <select
                key={opt.id}
                className="chat-model-select"
                title={opt.name}
                value={typeof opt.currentValue === 'string' ? opt.currentValue : ''}
                onChange={(e) => changeConfigOption(opt.id, e.target.value)}
              >
                {opt.options!.map((choice) => (
                  <option key={choice.value} value={choice.value}>
                    {choice.name}
                  </option>
                ))}
              </select>
            ))}
      </div>
      <div className="agent-chat-log" ref={scrollRef} role="log" aria-live="polite">
        {entries.map((e) => (
          <div key={e.id} className={`chat-entry chat-entry-${e.kind}`}>
            {e.kind === 'tool' ? (
              <div className="tool-card">
                <span className="tool-status">{e.status}</span> {e.title}
              </div>
            ) : e.kind === 'agent' || e.kind === 'thought' ? (
              <div className="chat-text chat-text-markdown">
                <ReactMarkdown>{e.text}</ReactMarkdown>
              </div>
            ) : (
              <div className="chat-text">{e.text}</div>
            )}
          </div>
        ))}
        {waiting && (
          <div className="chat-entry chat-entry-waiting">
            <div className="chat-text">
              {waitingLong
                ? 'Vẫn đang chờ — lâu hơn bình thường. Trợ lý có thể đang xử lý việc phức tạp, hoặc bị treo.'
                : 'Trợ lý đang suy nghĩ…'}
            </div>
            <button className="chat-cancel-btn" onClick={cancelTurn}>
              Dừng
            </button>
          </div>
        )}
      </div>

      {pendingPermission && (
        <div className="permission-card">
          <div className="permission-title">{pendingPermission.toolCall.title}</div>
          <div className="permission-options">
            {pendingPermission.options.map((opt) => (
              <button key={opt.optionId} onClick={() => respond(opt.optionId)}>
                {opt.name}
              </button>
            ))}
          </div>
        </div>
      )}

      <QuickActions onPrompt={sendPrompt} disabled={waiting} />

      {pendingImages.length > 0 && (
        <div className="chat-pending-images">
          {pendingImages.map((img, i) => (
            <div key={i} className="chat-pending-image">
              <img src={`data:${img.mimeType};base64,${img.data}`} alt={`ảnh dán ${i + 1}`} />
              <button
                className="chat-pending-image-remove"
                title="Bỏ ảnh"
                onClick={() => setPendingImages((prev) => prev.filter((_, idx) => idx !== i))}
              >
                ×
              </button>
            </div>
          ))}
        </div>
      )}

      <div className="agent-chat-input">
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              if (connected && !waiting) submit()
            }
          }}
          onPaste={handlePaste}
          placeholder="Hỏi trợ lý AI… (Enter để gửi, Shift+Enter xuống dòng, dán ảnh để đính kèm)"
        />
        <button onClick={submit} disabled={!connected || waiting}>
          Gửi
        </button>
      </div>
    </div>
  )
}
