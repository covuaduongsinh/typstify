// CollabProvider connects one open file to its co-editing room on the
// server (/ws/collab, service/collab). The server only relays and stores
// Yjs updates; everything that needs to understand the document happens
// here, and the leader the server picks (the earliest editor in the room)
// also writes the file to disk, compacts the room's log and folds in edits
// made outside the room.
import * as Y from 'yjs'
import { IndexeddbPersistence } from 'y-indexeddb'
import { Awareness, applyAwarenessUpdate, encodeAwarenessUpdate } from 'y-protocols/awareness'
import { wsUrl } from '../../api/client'
import type { ProjectRole } from '../../api/types'
import { Channel, MsgType, frame, parseControl, parseFrame, type ControlMessage, type PeerInfo } from './protocol'
import { applyTextDiff, rebaseChange } from './textDiff'

export type CollabMode = 'connecting' | 'live' | 'offline' | 'unavailable'

export interface CollabStatus {
  mode: CollabMode
  role: ProjectRole | null
  leader: boolean
  peers: PeerInfo[]
  /** When the leader last sent the text to be written to disk. */
  savedAt: number | null
  message: string | null
  /** Another version of the text the user must choose about: the file
   * changed outside the room where this text was being edited too, or
   * offline edits could not be merged into a replaced room history. */
  conflict: { kind: 'external' | 'offline'; text: string; hash: string } | null
}

export interface CollabUser {
  user: string
  name: string
  color: string
  colorLight: string
}

/** How the editor should open the file once the provider settles. */
export type ReadyMode = 'live' | 'offline' | 'unavailable'

const SAVE_DELAY = 1500
const OFFLINE_AFTER = 5000
const MAX_BACKOFF = 15000

const canEdit = (r: ProjectRole | null) => r === 'owner' || r === 'editor'
const canComment = (r: ProjectRole | null) => canEdit(r) || r === 'commenter'

/** A stable color per user, used for their cursor and avatar. */
export function userColor(user: string): { color: string; colorLight: string } {
  let h = 0
  for (const c of user) h = (h * 31 + c.charCodeAt(0)) >>> 0
  const hue = h % 360
  return { color: `hsl(${hue} 70% 45%)`, colorLight: `hsl(${hue} 70% 45% / 0.2)` }
}

const epochKey = (room: string) => `typstify.collab.epoch:${room}`

function readEpoch(room: string): string | null {
  try {
    return localStorage.getItem(epochKey(room))
  } catch {
    return null
  }
}

function writeEpoch(room: string, epoch: string) {
  try {
    localStorage.setItem(epochKey(room), epoch)
  } catch {
    // without storage the next visit simply starts from the server state
  }
}

export class CollabProvider {
  textDoc!: Y.Doc
  commentsDoc!: Y.Doc
  ytext!: Y.Text
  awareness!: Awareness
  readonly ready: Promise<ReadyMode>
  /** Called when the room's history was replaced after the editor opened
   * (the server lost its log): the caller must reopen the file. */
  onReset: (() => void) | null = null

  private readonly path: string
  private readonly project: string
  private readonly room: string
  private readonly me: CollabUser
  private ws: WebSocket | null = null
  private replay: Y.Doc[] | null = null
  private synced = false
  private closed = false
  private retries = 0
  private settled = false
  private settle!: (m: ReadyMode) => void
  private timers: Record<string, number | undefined> = {}
  private persistence: IndexeddbPersistence[] = []
  private hadLocalCopy = false
  /** IndexedDB name prefix of the local copy in use. */
  private base = ''
  /** A local copy kept under another epoch, checked for edits the server
   * never got once the replay finished (see recoverOldCopy). */
  private oldCopy: string | null = null
  /** Hash of the disk version the shared text includes; sent with every
   * save so the server never overwrites a newer file (service/collab). */
  private diskSeen = ''
  private status: CollabStatus = {
    mode: 'connecting',
    role: null,
    leader: false,
    peers: [],
    savedAt: null,
    message: null,
    conflict: null,
  }
  private listeners = new Set<() => void>()

  /** project is the server path of the open project (the server refuses
   * the room if it now has another project open); room names this file's
   * local copy (IndexedDB). */
  constructor(project: string, path: string, room: string, me: CollabUser) {
    this.project = project
    this.path = path
    this.room = room
    this.me = me
    this.ready = new Promise((resolve) => {
      this.settle = (m) => {
        if (this.settled) return
        this.settled = true
        resolve(m)
      }
    })
    this.makeDocs(readEpoch(room) ?? 'new')
    this.connect()
    this.timers.offline = window.setTimeout(() => {
      // No server: edit the local copy if there is one for this room.
      if (this.synced) return
      if (this.hadLocalCopy) this.settle('offline')
      else this.giveUp()
    }, OFFLINE_AFTER)
  }

  // --- status for React (useSyncExternalStore) ---
  getStatus = () => this.status
  subscribe = (fn: () => void) => {
    this.listeners.add(fn)
    return () => this.listeners.delete(fn)
  }
  private setStatus(patch: Partial<CollabStatus>) {
    this.status = { ...this.status, ...patch }
    for (const fn of this.listeners) fn()
  }

  get canEdit() {
    return canEdit(this.status.role)
  }
  get canComment() {
    return canComment(this.status.role)
  }

  /** The local copy is stored per epoch: a replaced history gets a new
   * database instead of reusing one whose deletion is still pending. */
  private makeDocs(epoch: string) {
    this.textDoc = new Y.Doc()
    this.commentsDoc = new Y.Doc()
    this.ytext = this.textDoc.getText('content')
    this.awareness = new Awareness(this.textDoc)
    this.awareness.setLocalStateField('user', { name: this.me.name, color: this.me.color, colorLight: this.me.colorLight })

    const base = `${this.room}@${epoch}`
    this.base = base
    const text = new IndexeddbPersistence(`${base}:text`, this.textDoc)
    this.persistence = [text, new IndexeddbPersistence(`${base}:comments`, this.commentsDoc)]
    void text.whenSynced.then(() => {
      this.hadLocalCopy = epoch !== 'new' && this.ytext.length > 0
    })

    this.textDoc.on('update', (update: Uint8Array, origin: unknown) => {
      if (origin !== this) this.sendUpdate(Channel.Text, update)
      if (this.status.leader) this.scheduleSave()
    })
    this.commentsDoc.on('update', (update: Uint8Array, origin: unknown) => {
      if (origin !== this) this.sendUpdate(Channel.Comments, update)
    })
    this.awareness.on('update', ({ added, updated, removed }: Record<string, number[]>, origin: unknown) => {
      if (origin === this || !this.isOpen()) return
      const changed = [...added, ...updated, ...removed]
      this.send(frame(MsgType.Awareness, 0, encodeAwarenessUpdate(this.awareness, changed)))
    })
  }

  /** Closes the current docs, keeping their local copy for recovery. */
  private closeDocs() {
    for (const p of this.persistence) void p.destroy()
    this.awareness.destroy()
    this.textDoc.destroy()
    this.commentsDoc.destroy()
  }

  private isOpen() {
    return this.ws?.readyState === WebSocket.OPEN && this.synced
  }

  private send(data: Uint8Array<ArrayBuffer>) {
    if (this.ws?.readyState === WebSocket.OPEN) this.ws.send(data)
  }

  private sendUpdate(channel: number, update: Uint8Array) {
    // Before the replay finished, local changes wait: the diff sent at
    // "synced" includes them.
    if (!this.isOpen()) return
    if (channel === Channel.Text ? this.canEdit : this.canComment) this.send(frame(MsgType.Update, channel, update))
  }

  private connect() {
    if (this.closed) return
    const q = `path=${encodeURIComponent(this.path)}&project=${encodeURIComponent(this.project)}`
    const ws = new WebSocket(wsUrl(`/ws/collab?${q}`))
    ws.binaryType = 'arraybuffer'
    this.ws = ws
    ws.onmessage = (ev) => this.receive(new Uint8Array(ev.data as ArrayBuffer))
    ws.onclose = (ev) => {
      if (this.ws !== ws) return
      this.ws = null
      this.synced = false
      this.replay = null
      if (this.closed) return
      // 1008: the server refused us for good (access revoked, file too
      // large). Anything else, including 4000 (fell behind), is retried.
      if (ev.code === 1008) {
        this.setStatus({ mode: 'unavailable', message: ev.reason || 'Không thể cộng tác trên tệp này' })
        this.giveUp()
        return
      }
      this.setStatus({ mode: 'offline', leader: false })
      const delay = Math.min(1000 * 2 ** this.retries++, MAX_BACKOFF)
      this.timers.retry = window.setTimeout(() => this.connect(), delay)
    }
  }

  private docFor(channel: number): Y.Doc | null {
    return channel === Channel.Text ? this.textDoc : channel === Channel.Comments ? this.commentsDoc : null
  }

  private receive(data: Uint8Array) {
    const f = parseFrame(data)
    if (!f) return
    switch (f.type) {
      case MsgType.Update: {
        const doc = this.docFor(f.channel)
        if (!doc) return
        if (this.replay) Y.applyUpdate(this.replay[f.channel], f.payload)
        Y.applyUpdate(doc, f.payload, this)
        break
      }
      case MsgType.Awareness:
        applyAwarenessUpdate(this.awareness, f.payload, this)
        break
      case MsgType.Synced:
        this.finishReplay()
        break
      case MsgType.Control: {
        const msg = parseControl(f.payload)
        if (msg) this.control(msg)
        break
      }
    }
  }

  private control(msg: ControlMessage) {
    switch (msg.type) {
      case 'hello': {
        const known = readEpoch(this.room)
        if (known !== msg.epoch) {
          if (this.settled) {
            // The file is already open on the old history. The reopened
            // provider starts from the stored epoch again and takes the
            // branch below, which keeps this local copy for recovery. No
            // final save: this text belongs to the replaced history.
            this.destroy(false)
            this.onReset?.()
            return
          }
          // A different history than the local copy: merging the two would
          // duplicate the text. Start from the server's, and look at the
          // old copy for edits the server never got once synced.
          this.oldCopy = this.base
          this.closeDocs()
          this.makeDocs(msg.epoch)
          writeEpoch(this.room, msg.epoch)
        }
        this.replay = [new Y.Doc(), new Y.Doc()]
        this.diskSeen = msg.diskHash
        this.setStatus({ role: msg.role, leader: msg.leader, peers: msg.peers, message: null })
        break
      }
      case 'peers':
        this.setStatus({ peers: msg.peers })
        break
      case 'leader':
        this.setStatus({ leader: msg.leader !== false })
        if (msg.leader !== false) this.scheduleSave()
        break
      case 'compact': {
        const doc = this.docFor(msg.channel)
        if (doc) this.send(frame(MsgType.Compact, msg.channel, Y.encodeStateAsUpdate(doc)))
        break
      }
      case 'external':
        if (this.status.leader && this.canEdit) this.foldExternal(msg.base, msg.text, msg.hash)
        break
      case 'error':
        this.setStatus({ message: msg.message })
        break
    }
  }

  private finishReplay() {
    const replay = this.replay
    this.replay = null
    this.synced = true
    this.retries = 0
    // Send what the server lacks: edits made offline or before the replay.
    if (replay) {
      const channels = [
        [Channel.Text, this.textDoc, this.canEdit],
        [Channel.Comments, this.commentsDoc, this.canComment],
      ] as const
      for (const [ch, doc, allowed] of channels) {
        const missing = Y.encodeStateAsUpdate(doc, Y.encodeStateVector(replay[ch]))
        // An empty update is 2 bytes (no structs, no deletions).
        if (allowed && missing.length > 2) this.send(frame(MsgType.Update, ch, missing))
        replay[ch].destroy()
      }
    }
    this.send(frame(MsgType.Awareness, 0, encodeAwarenessUpdate(this.awareness, [this.textDoc.clientID])))
    this.setStatus({ mode: 'live' })
    if (this.status.leader) this.scheduleSave()
    if (this.oldCopy) void this.recoverOldCopy(this.oldCopy)
    this.oldCopy = null
    this.settle('live')
  }

  /** Reads the local copy of a replaced history; if it holds text the new
   * history lacks (offline edits), offers it back as a conflict. The old
   * databases are deleted afterwards either way. */
  private async recoverOldCopy(base: string) {
    const tmp = new Y.Doc()
    const text = new IndexeddbPersistence(`${base}:text`, tmp)
    try {
      await text.whenSynced
      const old = tmp.getText('content').toString()
      if (!this.closed && this.canEdit && old && old !== this.ytext.toString()) {
        this.setStatus({ conflict: { kind: 'offline', text: old, hash: this.diskSeen } })
      }
    } finally {
      await text.clearData()
      await new IndexeddbPersistence(`${base}:comments`, new Y.Doc()).clearData()
      tmp.destroy()
    }
  }

  /** The file changed outside the room (agent, Dropbox, desktop). Only
   * that change (base -> text) is replayed on the shared text, so edits
   * not yet saved survive; overlapping edits become a conflict. */
  private foldExternal(base: string, text: string, hash: string) {
    const r = rebaseChange(base, text, this.ytext.toString())
    if (r.kind === 'conflict') {
      // Saves stop until the user decides; the server keeps the file.
      this.setStatus({ conflict: { kind: 'external', text, hash } })
      return
    }
    if (r.kind === 'apply') {
      this.textDoc.transact(() => {
        if (r.to > r.from) this.ytext.delete(r.from, r.to - r.from)
        if (r.insert) this.ytext.insert(r.from, r.insert)
      })
    }
    this.diskSeen = hash
    // Also when nothing changed: the save records that text and file agree.
    this.scheduleSave()
  }

  /** Settles a conflict: take the other version, or keep the shared text
   * (which is then written over the file). */
  resolveConflict(takeOther: boolean) {
    const c = this.status.conflict
    if (!c) return
    if (takeOther && this.canEdit) applyTextDiff(this.ytext, c.text)
    // Either way the user has now seen that disk version.
    if (c.hash) this.diskSeen = c.hash
    this.setStatus({ conflict: null })
    this.flush()
  }

  private scheduleSave() {
    window.clearTimeout(this.timers.save)
    this.timers.save = window.setTimeout(() => this.flush(), SAVE_DELAY)
  }

  /** Writes the text to disk now (leader only; others' saves are the
   * leader's job, and none happen while a conflict waits for the user).
   * Returns whether this client asked for a save. */
  flush(): boolean {
    window.clearTimeout(this.timers.save)
    if (!this.status.leader || !this.isOpen() || !this.canEdit || this.status.conflict) return false
    const seen = this.diskSeen.padEnd(64, '0').slice(0, 64)
    this.send(frame(MsgType.Save, 0, new TextEncoder().encode(seen + this.ytext.toString())))
    this.setStatus({ savedAt: Date.now() })
    return true
  }

  /** The file opens without co-editing. The provider must stop for good:
   * reconnecting later as leader would save its stale copy over the edits
   * made meanwhile in the plain editor. */
  private giveUp() {
    this.setStatus({ mode: 'unavailable' })
    this.settle('unavailable')
    this.destroy()
  }

  /** Closes the room. save: write the text one last time (leader only). */
  destroy(save = true) {
    if (this.closed) return
    if (save) this.flush()
    this.closed = true
    for (const id of Object.values(this.timers)) window.clearTimeout(id)
    // Tells the others this cursor is gone, if still connected.
    this.awareness.setLocalState(null)
    const ws = this.ws
    this.ws = null
    ws?.close()
    for (const p of this.persistence) void p.destroy()
    this.awareness.destroy()
    this.textDoc.destroy()
    this.commentsDoc.destroy()
    this.settle('unavailable')
    this.listeners.clear()
  }
}
