import { useEffect, useState, useSyncExternalStore } from 'react'
import { listThreads, type CommentThread } from './comments'
import { CollabProvider, type CollabStatus, type CollabUser, type ReadyMode } from './provider'

const OFF: CollabStatus = {
  mode: 'unavailable',
  role: null,
  leader: false,
  peers: [],
  savedAt: null,
  message: null,
  conflict: null,
}
const noSubscribe = () => () => undefined

export interface CollabSession {
  provider: CollabProvider | null
  /** null while the provider is still deciding how the file opens. */
  mode: ReadyMode | null
  status: CollabStatus
}

/**
 * useCollab opens the co-editing room of `path` (a provider per open file)
 * and closes it when the file, the project or the user changes. The
 * browser's local copy (IndexedDB) is scoped to this server and project.
 */
export function useCollab(project: string, path: string | null, me: CollabUser | null): CollabSession {
  const [session, setSession] = useState<{ provider: CollabProvider | null; mode: ReadyMode | null }>({
    provider: null,
    mode: null,
  })
  const [generation, setGeneration] = useState(0)
  const user = me?.user
  const name = me?.name

  useEffect(() => {
    if (!path || !me) return
    // Scoped per user too: on a shared computer the next person must not
    // see (or send) the previous one's offline copy.
    const room = `typstify:${window.location.host}:${me.user}:${project}:${path}`
    const provider = new CollabProvider(project, path, room, me)
    let alive = true
    // A new provider for a new file/user: nothing is decided yet.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setSession({ provider, mode: null })
    void provider.ready.then((mode) => {
      if (alive) setSession({ provider, mode })
    })
    provider.onReset = () => {
      if (alive) setGeneration((g) => g + 1)
    }
    return () => {
      alive = false
      provider.destroy()
    }
    // me is identified by user/name; a new object with the same values
    // must not reopen the room.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [project, path, user, name, generation])

  const p = path && me ? session.provider : null
  const status = useSyncExternalStore(p?.subscribe ?? noSubscribe, p?.getStatus ?? (() => OFF))
  return { provider: p, mode: p ? session.mode : null, status }
}

/** Live list of a file's threads; re-read on every comment or text change
 * (the text moves anchors). */
export function useThreads(provider: CollabProvider | null): CommentThread[] {
  const [threads, setThreads] = useState<CommentThread[]>([])
  useEffect(() => {
    if (!provider) return
    const refresh = () => setThreads(listThreads(provider.commentsDoc, provider.textDoc))
    refresh()
    provider.commentsDoc.on('update', refresh)
    provider.textDoc.on('update', refresh)
    return () => {
      provider.commentsDoc.off('update', refresh)
      provider.textDoc.off('update', refresh)
    }
  }, [provider])
  return provider ? threads : []
}
