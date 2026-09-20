import { useEffect, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import { AgentClient } from '../lib/agentClient'
import { contentBlockText, type AuthRequiredData, type MessageChunk, type PermissionRequest, type ToolCall } from '../lib/acpTypes'
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
          append({ kind: 'plan', id: `e${nextEntryId++}`, text: 'Plan updated' })
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
          append({ kind: 'error', id: `e${nextEntryId++}`, text: msg.message ?? 'Unknown agent error' })
          break
        case 'disconnected':
          setConnected(false)
          setWaiting(false)
          setHasDisconnectedOnce(true)
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
              text: 'Connection to the agent was lost. Use "Retry connection" below to start a new session.',
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

  const sendPrompt = (text: string) => {
    const trimmed = text.trim()
    if (!trimmed || !clientRef.current) return
    append({ kind: 'user', id: `e${nextEntryId++}`, text: trimmed })
    setWaiting(true)
    clientRef.current.prompt(trimmed)
  }

  const submit = () => {
    const text = input.trim()
    if (!text) return
    sendPrompt(text)
    setInput('')
  }

  const respond = (optionId: string) => {
    clientRef.current?.respondPermission(optionId)
    setPendingPermission(null)
  }

  const cancelTurn = () => {
    clientRef.current?.cancel()
    setWaiting(false)
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
        {connected ? 'Agent connected' : 'Connecting to agent…'}
        {!connected && hasDisconnectedOnce && (
          <button className="chat-retry-btn" onClick={() => setRetryToken((n) => n + 1)}>
            Retry connection
          </button>
        )}
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
                ? 'Still waiting — this is taking longer than usual. The agent may be working on something complex, or stuck.'
                : 'Agent is thinking…'}
            </div>
            <button className="chat-cancel-btn" onClick={cancelTurn}>
              Cancel
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
          placeholder="Ask the AI agent…"
        />
        <button onClick={submit} disabled={!connected || waiting}>
          Send
        </button>
      </div>
    </div>
  )
}
