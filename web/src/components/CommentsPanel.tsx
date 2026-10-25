import { useState } from 'react'
import { createThread, deleteThread, mentions, reply, setResolved } from '../lib/collab/comments'
import type { CollabProvider } from '../lib/collab/provider'
import { useThreads } from '../lib/collab/useCollab'
import { Icon } from './Icon'

interface CommentsPanelProps {
  provider: CollabProvider | null
  me: { user: string; name: string } | null
  canComment: boolean
  canModerate: boolean
  /** The editor's current selection, to anchor a new thread. */
  getSelection: () => { from: number; to: number; text: string } | null
  onJump: (from: number, to: number) => void
}

function Body({ text }: { text: string }) {
  const names = new Set(mentions(text))
  if (!names.size) return <>{text}</>
  return (
    <>
      {text.split(/(@[\p{L}\p{N}_.-]+)/u).map((part, i) =>
        part.startsWith('@') && names.has(part.slice(1)) ? (
          <span key={i} className="comment-mention">
            {part}
          </span>
        ) : (
          part
        ),
      )}
    </>
  )
}

const timeLabel = (at: number) =>
  new Date(at).toLocaleString('vi-VN', { hour: '2-digit', minute: '2-digit', day: '2-digit', month: '2-digit' })

/** CommentsPanel shows and edits the comment threads of the open file
 * (lib/collab/comments). Commenters and editors may comment; editors and
 * owners may also delete any thread. */
export function CommentsPanel({ provider, me, canComment, canModerate, getSelection, onJump }: CommentsPanelProps) {
  const threads = useThreads(provider)
  const [draft, setDraft] = useState('')
  const [replies, setReplies] = useState<Record<string, string>>({})
  const [showResolved, setShowResolved] = useState(false)
  const [hint, setHint] = useState<string | null>(null)

  if (!provider || !me) {
    return (
      <div className="comments-panel">
        <p className="settings-hint">Bình luận có khi mở một tệp .typ đang được đồng bộ cộng tác.</p>
      </div>
    )
  }

  const add = () => {
    const sel = getSelection()
    if (!sel || sel.to <= sel.from) {
      setHint('Hãy bôi đen đoạn văn bản cần bình luận trong trình soạn thảo trước.')
      return
    }
    createThread(provider.commentsDoc, provider.ytext, sel.from, sel.to, me, draft.trim())
    setDraft('')
    setHint(null)
  }

  const visible = threads.filter((t) => showResolved || !t.resolved)
  const resolvedCount = threads.filter((t) => t.resolved).length

  return (
    <div className="comments-panel">
      {canComment ? (
        <form
          className="comment-new"
          onSubmit={(e) => {
            e.preventDefault()
            if (draft.trim()) add()
          }}
        >
          <textarea
            placeholder="Bôi đen đoạn văn rồi viết bình luận… (gõ @tên để nhắc ai đó)"
            value={draft}
            rows={3}
            onChange={(e) => setDraft(e.target.value)}
          />
          {hint && <p className="settings-hint">{hint}</p>}
          <button type="submit" className="btn-primary" disabled={!draft.trim()}>
            <Icon name="message" size={14} /> Bình luận đoạn đã chọn
          </button>
        </form>
      ) : (
        <p className="settings-hint">Bạn chỉ có quyền xem bình luận.</p>
      )}

      {resolvedCount > 0 && (
        <label className="comment-filter">
          <input type="checkbox" checked={showResolved} onChange={(e) => setShowResolved(e.target.checked)} /> Hiện{' '}
          {resolvedCount} luồng đã xong
        </label>
      )}

      {visible.length === 0 && <p className="settings-hint">Chưa có bình luận nào.</p>}

      <ul className="comment-threads">
        {visible.map((t) => (
          <li key={t.id} className={`comment-thread${t.resolved ? ' resolved' : ''}${t.from === null ? ' orphan' : ''}`}>
            <button
              className="comment-quote"
              disabled={t.from === null}
              title={t.from === null ? 'Đoạn văn này đã bị xoá' : 'Nhảy tới đoạn văn'}
              onClick={() => t.from !== null && t.to !== null && onJump(t.from, t.to)}
            >
              “{t.quote || '…'}”
            </button>
            {t.messages.map((m) => (
              <div key={m.id} className="comment-message">
                <div className="comment-meta">
                  <strong>{m.authorName || m.author}</strong> · {timeLabel(m.at)}
                </div>
                <div className="comment-body">
                  <Body text={m.text} />
                </div>
              </div>
            ))}
            {canComment && (
              <form
                className="comment-reply"
                onSubmit={(e) => {
                  e.preventDefault()
                  const body = replies[t.id]?.trim()
                  if (!body) return
                  reply(provider.commentsDoc, t.id, me, body)
                  setReplies((r) => ({ ...r, [t.id]: '' }))
                }}
              >
                <input
                  placeholder="Trả lời…"
                  value={replies[t.id] ?? ''}
                  onChange={(e) => setReplies((r) => ({ ...r, [t.id]: e.target.value }))}
                />
              </form>
            )}
            {canComment && (
              <div className="comment-actions">
                <button className="btn-ghost" onClick={() => setResolved(provider.commentsDoc, t.id, !t.resolved)}>
                  <Icon name={t.resolved ? 'refresh' : 'check'} size={13} /> {t.resolved ? 'Mở lại' : 'Xong'}
                </button>
                {(canModerate || t.author === me.user) && (
                  <button className="btn-ghost" onClick={() => deleteThread(provider.commentsDoc, t.id)}>
                    <Icon name="x" size={13} /> Xoá
                  </button>
                )}
              </div>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}
