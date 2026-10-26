import { useCallback, useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'
import { ROLE_LABELS, type ProjectMember, type ProjectRole, type ShareLinkInfo } from '../api/types'
import { Icon } from './Icon'
import { Modal } from './Modal'

const GRANTABLE: ProjectRole[] = ['editor', 'commenter', 'viewer']

/** ShareModal manages a registered project's members and share links
 * (server/projects_api.go). Only owners and admins can open it. */
export function ShareModal({ projectId, projectName, onClose }: { projectId: string; projectName: string; onClose: () => void }) {
  const [members, setMembers] = useState<ProjectMember[]>([])
  const [links, setLinks] = useState<ShareLinkInfo[]>([])
  const [newUser, setNewUser] = useState('')
  const [newUserRole, setNewUserRole] = useState<ProjectRole>('editor')
  const [linkRole, setLinkRole] = useState<ProjectRole>('viewer')
  const [linkDays, setLinkDays] = useState(7)
  const [createdLink, setCreatedLink] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const base = `/api/projects/${encodeURIComponent(projectId)}`

  const reload = useCallback(async () => {
    try {
      const [m, l] = await Promise.all([
        api.get<ProjectMember[]>(`${base}/members`),
        api.get<ShareLinkInfo[]>(`${base}/shares`),
      ])
      setMembers(m ?? [])
      setLinks(l ?? [])
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Không tải được thông tin chia sẻ')
    }
  }, [base])

  useEffect(() => {
    void reload()
  }, [reload])

  const act = async (fn: () => Promise<unknown>) => {
    setError(null)
    setBusy(true)
    try {
      await fn()
      await reload()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Thao tác thất bại')
    } finally {
      setBusy(false)
    }
  }

  const setRole = (user: string, role: ProjectRole) =>
    act(() => api.putJson(`${base}/members/${encodeURIComponent(user)}`, { role }))

  const createLink = () =>
    act(async () => {
      const res = await api.post<{ token: string }>(`${base}/shares`, { role: linkRole, expiresInDays: linkDays })
      setCreatedLink(`${window.location.origin}/?join=${res.token}`)
      setCopied(false)
    })

  const copy = async () => {
    if (!createdLink) return
    try {
      await navigator.clipboard.writeText(createdLink)
      setCopied(true)
    } catch {
      setCopied(false)
    }
  }

  return (
    <Modal title={`Chia sẻ “${projectName}”`} onClose={onClose} className="share-modal">
      {error && (
        <div className="error" role="alert">
          {error}
        </div>
      )}

      <section aria-labelledby="share-members-h">
        <h3 id="share-members-h">Thành viên</h3>
        <ul className="share-list">
          {members.map((m) => (
            <li key={m.username} className="share-row">
              <span className="share-user">{m.username}</span>
              {m.role === 'owner' ? (
                <span className="share-role">{ROLE_LABELS.owner}</span>
              ) : (
                <>
                  <select
                    aria-label={`Vai trò của ${m.username}`}
                    value={m.role}
                    disabled={busy}
                    onChange={(e) => void setRole(m.username, e.target.value as ProjectRole)}
                  >
                    {GRANTABLE.map((r) => (
                      <option key={r} value={r}>
                        {ROLE_LABELS[r]}
                      </option>
                    ))}
                  </select>
                  <button
                    className="btn-ghost"
                    disabled={busy}
                    title={`Xoá ${m.username} khỏi dự án`}
                    aria-label={`Xoá ${m.username} khỏi dự án`}
                    onClick={() => void act(() => api.del(`${base}/members/${encodeURIComponent(m.username)}`))}
                  >
                    <Icon name="x" size={14} />
                  </button>
                </>
              )}
            </li>
          ))}
        </ul>
        <form
          className="share-add"
          onSubmit={(e) => {
            e.preventDefault()
            const u = newUser.trim().toLowerCase()
            if (u) void setRole(u, newUserRole).then(() => setNewUser(''))
          }}
        >
          <input
            aria-label="Tên tài khoản cần thêm"
            placeholder="Tên tài khoản trên máy chủ này"
            value={newUser}
            onChange={(e) => setNewUser(e.target.value)}
          />
          <select aria-label="Vai trò" value={newUserRole} onChange={(e) => setNewUserRole(e.target.value as ProjectRole)}>
            {GRANTABLE.map((r) => (
              <option key={r} value={r}>
                {ROLE_LABELS[r]}
              </option>
            ))}
          </select>
          <button type="submit" disabled={busy || !newUser.trim()}>
            Thêm
          </button>
        </form>
      </section>

      <section aria-labelledby="share-links-h">
        <h3 id="share-links-h">Liên kết chia sẻ</h3>
        <p className="entry-hint">Người nhận cần đăng nhập (hoặc được tạo tài khoản) trên máy chủ này rồi mở liên kết.</p>
        <div className="share-add">
          <select aria-label="Vai trò qua liên kết" value={linkRole} onChange={(e) => setLinkRole(e.target.value as ProjectRole)}>
            {GRANTABLE.map((r) => (
              <option key={r} value={r}>
                {ROLE_LABELS[r]}
              </option>
            ))}
          </select>
          <select aria-label="Thời hạn" value={linkDays} onChange={(e) => setLinkDays(Number(e.target.value))}>
            <option value={1}>1 ngày</option>
            <option value={7}>7 ngày</option>
            <option value={30}>30 ngày</option>
            <option value={0}>Không hết hạn</option>
          </select>
          <button className="btn-primary" disabled={busy} onClick={() => void createLink()}>
            Tạo liên kết
          </button>
        </div>
        {createdLink && (
          <div className="share-created">
            <input readOnly value={createdLink} aria-label="Liên kết vừa tạo" onFocus={(e) => e.currentTarget.select()} />
            <button onClick={() => void copy()}>{copied ? 'Đã chép' : 'Chép'}</button>
            <p className="entry-hint">Liên kết chỉ hiện một lần — hãy chép ngay.</p>
          </div>
        )}
        {links.length > 0 && (
          <ul className="share-list">
            {links.map((l) => (
              <li key={l.id} className="share-row">
                <span className="share-user">{ROLE_LABELS[l.role]}</span>
                <span className="share-meta">
                  {l.expiresAt ? `hết hạn ${new Date(l.expiresAt).toLocaleDateString('vi')}` : 'không hết hạn'} ·{' '}
                  {l.uses} lượt dùng
                </span>
                <button
                  className="btn-ghost"
                  disabled={busy}
                  onClick={() => void act(() => api.del(`${base}/shares/${encodeURIComponent(l.id)}`))}
                >
                  Thu hồi
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>
    </Modal>
  )
}
