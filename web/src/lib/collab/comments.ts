// Comment threads of a co-edited file. They live in their own Y.Doc (the
// "comments" channel), so the server can let commenters write them while
// only editors write the text. A thread is anchored with Yjs relative
// positions into the text document: the anchor follows the text it was
// made on as people edit around it.
import * as Y from 'yjs'

export interface CommentMessage {
  id: string
  author: string
  authorName: string
  text: string
  at: number
}

export interface CommentThread {
  id: string
  author: string
  authorName: string
  createdAt: number
  resolved: boolean
  /** The commented text when the thread was opened. */
  quote: string
  messages: CommentMessage[]
  /** Current range in the text, or null if that text was deleted. */
  from: number | null
  to: number | null
}

export interface Author {
  user: string
  name: string
}

const threadsOf = (doc: Y.Doc) => doc.getMap<Y.Map<unknown>>('threads')

const newId = () =>
  typeof crypto !== 'undefined' && 'randomUUID' in crypto
    ? crypto.randomUUID()
    : `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`

export function createThread(
  comments: Y.Doc,
  text: Y.Text,
  from: number,
  to: number,
  author: Author,
  body: string,
): string {
  const id = newId()
  const start = Y.encodeRelativePosition(Y.createRelativePositionFromTypeIndex(text, from, 0))
  // The end sticks to the character before it, so typing right after the
  // commented span does not grow it.
  const end = Y.encodeRelativePosition(Y.createRelativePositionFromTypeIndex(text, to, -1))
  comments.transact(() => {
    const t = new Y.Map<unknown>()
    t.set('id', id)
    t.set('author', author.user)
    t.set('authorName', author.name)
    t.set('createdAt', Date.now())
    t.set('resolved', false)
    t.set('quote', text.toString().slice(from, to).slice(0, 280))
    t.set('start', start)
    t.set('end', end)
    const messages = new Y.Array<CommentMessage>()
    messages.push([{ id: newId(), author: author.user, authorName: author.name, text: body, at: Date.now() }])
    t.set('messages', messages)
    threadsOf(comments).set(id, t)
  })
  return id
}

export function reply(comments: Y.Doc, threadId: string, author: Author, body: string) {
  const messages = threadsOf(comments).get(threadId)?.get('messages') as Y.Array<CommentMessage> | undefined
  messages?.push([{ id: newId(), author: author.user, authorName: author.name, text: body, at: Date.now() }])
}

export function setResolved(comments: Y.Doc, threadId: string, resolved: boolean) {
  threadsOf(comments).get(threadId)?.set('resolved', resolved)
}

export function deleteThread(comments: Y.Doc, threadId: string) {
  threadsOf(comments).delete(threadId)
}

function resolve(bytes: unknown, textDoc: Y.Doc): number | null {
  if (!(bytes instanceof Uint8Array)) return null
  const abs = Y.createAbsolutePositionFromRelativePosition(Y.decodeRelativePosition(bytes), textDoc)
  return abs ? abs.index : null
}

/** All threads, oldest first, with their current position in the text. */
export function listThreads(comments: Y.Doc, textDoc: Y.Doc): CommentThread[] {
  const out: CommentThread[] = []
  threadsOf(comments).forEach((t) => {
    const from = resolve(t.get('start'), textDoc)
    const to = resolve(t.get('end'), textDoc)
    // Deleting the whole span collapses both ends onto one point.
    const orphan = from === null || to === null || to <= from
    const messages = (t.get('messages') as Y.Array<CommentMessage> | undefined)?.toArray() ?? []
    out.push({
      id: String(t.get('id')),
      author: String(t.get('author') ?? ''),
      authorName: String(t.get('authorName') ?? ''),
      createdAt: Number(t.get('createdAt') ?? 0),
      resolved: t.get('resolved') === true,
      quote: String(t.get('quote') ?? ''),
      messages,
      from: orphan ? null : from,
      to: orphan ? null : to,
    })
  })
  return out.sort((a, b) => a.createdAt - b.createdAt)
}

/** `@user` mentions in a comment, for highlighting. */
export function mentions(body: string): string[] {
  return [...body.matchAll(/(^|\s)@([\p{L}\p{N}_.-]+)/gu)].map((m) => m[2])
}
