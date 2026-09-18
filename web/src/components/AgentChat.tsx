import { useEffect, useRef, useState } from 'react'
import { AgentClient } from '../lib/agentClient'
import { contentBlockText, type MessageChunk, type PermissionRequest, type ToolCall } from '../lib/acpTypes'

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
  const [pendingPermission, setPendingPermission] = useState<PermissionRequest | null>(null)
  const clientRef = useRef<AgentClient | null>(null)
  const streamingAgentId = useRef<string | null>(null)
  const streamingThoughtId = useRef<string | null>(null)
  const scrollRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    setEntries([])
    const client = new AgentClient()
    clientRef.current = client

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

    const unsubscribe = client.on((msg) => {
      switch (msg.type) {
        case 'ready':
          setConnected(true)
          break
        case 'userMessage':
          streamingAgentId.current = null
          streamingThoughtId.current = null
          append({ kind: 'user', id: `e${nextEntryId++}`, text: contentBlockText((msg.data as MessageChunk).content) })
          break
        case 'agentMessage':
          streamingThoughtId.current = null
          appendOrGrowText('agent', contentBlockText((msg.data as MessageChunk).content), streamingAgentId)
          break
        case 'agentThought':
          streamingAgentId.current = null
          appendOrGrowText('thought', contentBlockText((msg.data as MessageChunk).content), streamingThoughtId)
          break
        case 'toolCall': {
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
          append({ kind: 'plan', id: `e${nextEntryId++}`, text: 'Plan updated' })
          break
        case 'permissionRequest':
          setPendingPermission(msg.data as PermissionRequest)
          break
        case 'turnEnd':
          streamingAgentId.current = null
          streamingThoughtId.current = null
          break
        case 'error':
          append({ kind: 'error', id: `e${nextEntryId++}`, text: msg.message ?? 'Unknown agent error' })
          break
      }
    })

    return () => {
      unsubscribe()
      client.close()
      setConnected(false)
    }
  }, [projectPath])

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight })
  }, [entries])

  const submit = () => {
    const text = input.trim()
    if (!text || !clientRef.current) return
    clientRef.current.prompt(text)
    setInput('')
  }

  const respond = (optionId: string) => {
    clientRef.current?.respondPermission(optionId)
    setPendingPermission(null)
  }

  return (
    <div className="agent-chat">
      <div className="agent-chat-status">{connected ? 'Agent connected' : 'Connecting to agent…'}</div>
      <div className="agent-chat-log" ref={scrollRef}>
        {entries.map((e) => (
          <div key={e.id} className={`chat-entry chat-entry-${e.kind}`}>
            {e.kind === 'tool' ? (
              <div className="tool-card">
                <span className="tool-status">{e.status}</span> {e.title}
              </div>
            ) : (
              <div className="chat-text">{e.text}</div>
            )}
          </div>
        ))}
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

      <div className="agent-chat-input">
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              submit()
            }
          }}
          placeholder="Ask the AI agent…"
        />
        <button onClick={submit} disabled={!connected}>
          Send
        </button>
      </div>
    </div>
  )
}
