import { ROLE_LABELS } from '../api/types'
import { userColor, type CollabStatus } from '../lib/collab/provider'
import { Icon } from './Icon'

const initials = (name: string) =>
  name
    .split(/\s+/)
    .filter(Boolean)
    .slice(-2)
    .map((w) => w[0]?.toUpperCase())
    .join('') || '?'

const MODE_LABEL: Record<CollabStatus['mode'], string> = {
  connecting: 'Đang kết nối…',
  live: 'Đang đồng bộ',
  offline: 'Ngoại tuyến: thay đổi được giữ và gửi khi có mạng',
  unavailable: 'Không cộng tác',
}

/** Who else has the file open, and whether edits are reaching the server. */
export function CollabPresence({ status, self }: { status: CollabStatus; self: string | null }) {
  // One avatar per user, even with the file open in several tabs.
  const people = [...new Map(status.peers.map((p) => [p.user || p.name, p])).values()]
  return (
    <div className="collab-presence" title={status.message ?? MODE_LABEL[status.mode]}>
      <span className={`collab-dot collab-${status.mode}`} aria-label={MODE_LABEL[status.mode]} />
      {status.mode === 'offline' && <Icon name="cloud" size={14} />}
      {people.map((p) => (
        <span
          key={p.id}
          className={`collab-avatar${p.user === self ? ' me' : ''}`}
          style={{ background: userColor(p.user || p.name).color }}
          title={`${p.name}${p.user === self ? ' (bạn)' : ''} · ${ROLE_LABELS[p.role] ?? p.role}${p.leader ? ' · đang lưu tệp' : ''}`}
        >
          {initials(p.name)}
        </span>
      ))}
    </div>
  )
}
