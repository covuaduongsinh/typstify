import { useCallback, useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'
import { Icon } from './Icon'

// Mirrors service/gitsync (Status, CommitInfo, FileChange).
interface CommitInfo {
  hash: string
  message: string
  author: string
  when: string
}

interface GitStatus {
  initialized: boolean
  branch: string
  remote: string
  hasToken: boolean
  changes: { path: string; status: string }[]
  head?: CommitInfo
  remoteHead?: CommitInfo
  ahead: number
  behind: number
  diverged: boolean
  fetchError?: string
}

interface GitInfo {
  status: GitStatus
  canConfigure: boolean
  config: { remote: string; branch: string; username: string; authorName: string; authorEmail: string }
}

const CHANGE_LABEL: Record<string, string> = {
  added: 'thêm',
  modified: 'sửa',
  deleted: 'xoá',
  untracked: 'mới',
  renamed: 'đổi tên',
}

const commitLine = (c?: CommitInfo) =>
  c ? `${c.message || '(không có nội dung)'} · ${c.author} · ${new Date(c.when).toLocaleString('vi-VN')}` : 'chưa có'

/**
 * GitPanel syncs the project with a Git remote (service/gitsync). The
 * owner sets the remote and an access token (kept on the server); editors
 * commit, push and pull. When both sides have new commits nothing is
 * merged: the user picks a side, and pulling always saves the current
 * files as a history version first.
 */
export function GitPanel({ canEdit, onFilesChanged }: { canEdit: boolean; onFilesChanged: () => void }) {
  const [info, setInfo] = useState<GitInfo | null>(null)
  const [form, setForm] = useState({ remote: '', branch: '', username: '', token: '', authorName: '', authorEmail: '' })
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [diverged, setDiverged] = useState(false)
  const [showConfig, setShowConfig] = useState(false)

  const load = useCallback(async (fetchRemote: boolean) => {
    const res = await api.get<GitInfo>(`/api/git${fetchRemote ? '?fetch=1' : ''}`)
    setInfo(res)
    setForm((f) => ({ ...f, ...res.config, token: '' }))
    return res
  }, [])

  useEffect(() => {
    let alive = true
    api
      .get<GitInfo>('/api/git')
      .then((res) => {
        if (!alive) return
        setInfo(res)
        setForm((f) => ({ ...f, ...res.config, token: '' }))
        setShowConfig(!res.status.remote && res.canConfigure)
      })
      .catch(
        (err: unknown) => alive && setError(err instanceof ApiError ? err.message : 'Không đọc được trạng thái Git'),
      )
    return () => {
      alive = false
    }
  }, [])

  const run = async (fn: () => Promise<string | void>) => {
    setBusy(true)
    setError(null)
    setNotice(null)
    try {
      const done = await fn()
      if (done) setNotice(done)
    } catch (err) {
      if (err instanceof ApiError && err.body?.code === 'diverged') setDiverged(true)
      setError(err instanceof ApiError ? err.message : 'Thao tác thất bại')
    } finally {
      setBusy(false)
    }
  }

  const saveConfig = () =>
    run(async () => {
      await api.putJson('/api/git/config', form)
      setShowConfig(false)
      await load(false)
      return 'Đã lưu cấu hình Git.'
    })

  const commit = () =>
    run(async () => {
      await api.post('/api/git/commit', { message: message.trim() })
      setMessage('')
      await load(false)
      return 'Đã commit.'
    })

  const push = (force = false) =>
    run(async () => {
      await api.post('/api/git/push', { force })
      setDiverged(false)
      await load(true)
      return force ? 'Đã ghi đè nhánh trên kho bằng bản của bạn.' : 'Đã đẩy lên kho.'
    })

  const pull = (takeRemote = false) =>
    run(async () => {
      const res = await api.post<{ updated: boolean }>('/api/git/pull', { takeRemote })
      setDiverged(false)
      await load(true)
      if (res.updated) onFilesChanged()
      return res.updated
        ? 'Đã lấy bản từ kho. Bản trước đó được lưu trong Lịch sử ("Trước khi kéo từ Git").'
        : 'Đã là bản mới nhất.'
    })

  if (!info) {
    return error ? (
      <div className="error" role="alert">
        {error}
      </div>
    ) : (
      <span className="spinner" />
    )
  }
  const st = { ...info.status, changes: info.status.changes ?? [] }

  return (
    <div className="git-panel">
      {error && (
        <div className="error" role="alert">
          {error}
        </div>
      )}
      {notice && <p className="history-notice">{notice}</p>}

      {info.canConfigure && (
        <button className="btn-ghost" onClick={() => setShowConfig((v) => !v)}>
          <Icon name="sliders" size={14} /> Cấu hình kho
        </button>
      )}
      {showConfig && info.canConfigure && (
        <form
          className="git-config"
          onSubmit={(e) => {
            e.preventDefault()
            void saveConfig()
          }}
        >
          <label className="settings-field">
            <span>Địa chỉ kho (https://…)</span>
            <input
              value={form.remote}
              placeholder="https://github.com/ten/sach-co-vua.git"
              onChange={(e) => setForm({ ...form, remote: e.target.value })}
            />
          </label>
          <label className="settings-field">
            <span>Nhánh</span>
            <input
              value={form.branch}
              placeholder="main"
              onChange={(e) => setForm({ ...form, branch: e.target.value })}
            />
          </label>
          <label className="settings-field">
            <span>Tên đăng nhập kho</span>
            <input value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} />
          </label>
          <label className="settings-field">
            <span>Token truy cập</span>
            <input
              type="password"
              autoComplete="off"
              value={form.token}
              placeholder={st.hasToken ? 'Đã lưu — để trống để giữ nguyên' : 'GitHub: Personal access token'}
              onChange={(e) => setForm({ ...form, token: e.target.value })}
            />
          </label>
          <label className="settings-field">
            <span>Tên tác giả commit (tuỳ chọn)</span>
            <input value={form.authorName} onChange={(e) => setForm({ ...form, authorName: e.target.value })} />
          </label>
          <label className="settings-field">
            <span>Email tác giả (tuỳ chọn)</span>
            <input value={form.authorEmail} onChange={(e) => setForm({ ...form, authorEmail: e.target.value })} />
          </label>
          <p className="settings-hint">Token chỉ lưu trên máy chủ và không bao giờ được gửi lại trình duyệt.</p>
          <button type="submit" className="btn-primary" disabled={busy || !form.remote.trim()}>
            Lưu cấu hình
          </button>
        </form>
      )}

      <div className="git-status">
        <div>
          <strong>Kho:</strong> {st.remote || 'chưa cấu hình'}
        </div>
        <div>
          <strong>Nhánh:</strong> {st.branch || 'main'}
          {!st.initialized && ' (sẽ tạo kho Git ở lần commit đầu)'}
        </div>
        <div>
          <strong>Commit gần nhất:</strong> {commitLine(st.head)}
        </div>
        {st.remoteHead && (
          <div>
            <strong>Trên kho:</strong> {commitLine(st.remoteHead)}
          </div>
        )}
        {(st.ahead > 0 || st.behind > 0) && (
          <div className={st.diverged ? 'git-diverged' : undefined}>
            {st.ahead > 0 && `${st.ahead} commit chưa đẩy lên. `}
            {st.behind > 0 && `${st.behind} commit mới trên kho.`}
          </div>
        )}
        {st.fetchError && <div className="error">Không kết nối được kho: {st.fetchError}</div>}
      </div>

      <button
        className="btn-ghost"
        disabled={busy || !st.remote}
        onClick={() => void run(async () => void (await load(true)))}
      >
        <Icon name="refresh" size={14} /> Kiểm tra kho
      </button>

      {canEdit && (
        <>
          <h4 className="git-heading">
            {st.initialized
              ? `Thay đổi chưa commit (${st.changes.length})`
              : 'Commit đầu tiên sẽ đưa mọi tệp của dự án vào kho'}
          </h4>
          {st.changes.length > 0 && (
            <ul className="git-changes">
              {st.changes.slice(0, 50).map((c) => (
                <li key={c.path}>
                  <span className="history-status">{CHANGE_LABEL[c.status] ?? c.status}</span> {c.path}
                </li>
              ))}
              {st.changes.length > 50 && <li>… và {st.changes.length - 50} tệp khác</li>}
            </ul>
          )}
          <form
            className="history-save"
            onSubmit={(e) => {
              e.preventDefault()
              if (message.trim()) void commit()
            }}
          >
            <input
              placeholder="Nội dung commit, ví dụ: Thêm chương 3"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
            />
            <button
              type="submit"
              className="btn-primary"
              disabled={busy || !message.trim() || (st.initialized && st.changes.length === 0)}
            >
              Commit tất cả
            </button>
          </form>
          <div className="git-actions">
            <button className="btn-ghost" disabled={busy || !st.remote} onClick={() => void push()}>
              <Icon name="cloud-upload" size={14} /> Đẩy lên kho
            </button>
            <button className="btn-ghost" disabled={busy || !st.remote} onClick={() => void pull()}>
              <Icon name="cloud-download" size={14} /> Lấy về từ kho
            </button>
          </div>
          {diverged && (
            <div className="history-confirm">
              <p>Cả kho và máy chủ đều có commit mới. Typstify không tự gộp; chọn bên giữ lại:</p>
              <button className="btn-primary" disabled={busy} onClick={() => void pull(true)}>
                Lấy bản trên kho (bản hiện tại được lưu trong Lịch sử)
              </button>
              {info.canConfigure && (
                <button className="btn-ghost" disabled={busy} onClick={() => void push(true)}>
                  Ghi đè kho bằng bản của tôi
                </button>
              )}
            </div>
          )}
        </>
      )}
    </div>
  )
}
