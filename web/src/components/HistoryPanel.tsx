import { lazy, Suspense, useCallback, useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { HistoryChange, HistorySnapshot } from '../api/types'
import { Icon } from './Icon'
import { Modal } from './Modal'

const HistoryDiff = lazy(() => import('./HistoryDiff').then((m) => ({ default: m.HistoryDiff })))
const GitPanel = lazy(() => import('./GitPanel').then((m) => ({ default: m.GitPanel })))

// Files shown as text in the comparison; anything else is binary.
const TEXT_EXT = /\.(typ|md|txt|bib|ya?ml|json|csv|toml|xml|svg|tex|html|css|js|ts)$/i

const STATUS_LABEL: Record<HistoryChange['status'], string> = {
  modified: 'đã sửa',
  deleted: 'đã xoá',
  added: 'mới thêm',
  unchanged: 'như cũ',
}

const timeLabel = (iso: string) =>
  new Date(iso).toLocaleString('vi-VN', {
    hour: '2-digit',
    minute: '2-digit',
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })

function title(s: HistorySnapshot) {
  if (s.label) return s.label
  if (s.kind === 'restore') return 'Trước khi khôi phục'
  return 'Tự động'
}

const errorText = (err: unknown) => (err instanceof ApiError ? err.message : 'Thao tác thất bại')

interface DiffState {
  path: string
  before: string
  after: string
}

/**
 * HistoryPanel lists the project's versions (service/history), compares a
 * version with the files now and restores files or the whole version.
 * Every restore first saves the current state as a version, so it can be
 * undone from this same list.
 */
function VersionsPanel({ canEdit, onRestored }: { canEdit: boolean; onRestored: (paths: string[]) => void }) {
  const [list, setList] = useState<HistorySnapshot[] | null>(null)
  const [selected, setSelected] = useState<HistorySnapshot | null>(null)
  const [changes, setChanges] = useState<HistoryChange[] | null>(null)
  const [showUnchanged, setShowUnchanged] = useState(false)
  const [label, setLabel] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [diff, setDiff] = useState<DiffState | null>(null)
  const [confirmAll, setConfirmAll] = useState(false)
  const [deleteNewer, setDeleteNewer] = useState(false)

  const refresh = useCallback(async () => {
    try {
      setList(await api.get<HistorySnapshot[]>('/api/history'))
    } catch (err) {
      setError(errorText(err))
    }
  }, [])

  useEffect(() => {
    let alive = true
    api
      .get<HistorySnapshot[]>('/api/history')
      .then((l) => alive && setList(l))
      .catch((err: unknown) => alive && setError(errorText(err)))
    return () => {
      alive = false
    }
  }, [])

  const open = async (snap: HistorySnapshot) => {
    setSelected(snap)
    setChanges(null)
    setConfirmAll(false)
    setError(null)
    try {
      const res = await api.get<{ snapshot: HistorySnapshot; changes: HistoryChange[] }>(`/api/history/${snap.id}`)
      setChanges(res.changes)
    } catch (err) {
      setError(errorText(err))
    }
  }

  const run = async (fn: () => Promise<void>) => {
    setBusy(true)
    setError(null)
    setNotice(null)
    try {
      await fn()
    } catch (err) {
      setError(errorText(err))
    } finally {
      setBusy(false)
    }
  }

  const save = () =>
    run(async () => {
      await api.post('/api/history', { label: label.trim() })
      setLabel('')
      setNotice('Đã lưu phiên bản.')
      await refresh()
    })

  const restore = (paths: string[]) =>
    run(async () => {
      if (!selected) return
      await api.post(`/api/history/${selected.id}/restore`, { paths, deleteNewer: paths.length === 0 && deleteNewer })
      setNotice(
        paths.length === 1
          ? `Đã khôi phục ${paths[0]}. Trạng thái trước đó được lưu thành phiên bản "Trước khi khôi phục".`
          : 'Đã khôi phục cả phiên bản. Trạng thái trước đó được lưu thành phiên bản "Trước khi khôi phục".',
      )
      setConfirmAll(false)
      setDiff(null)
      onRestored(paths.length ? paths : (changes ?? []).map((c) => c.path))
      await refresh()
      await open(selected)
    })

  const showDiff = (c: HistoryChange) =>
    run(async () => {
      if (!selected) return
      const before =
        c.status === 'added'
          ? ''
          : await api.get<string>(`/api/history/${selected.id}/file?path=${encodeURIComponent(c.path)}`)
      const after =
        c.status === 'deleted' ? '' : await api.get<string>(`/api/workspace/file?path=${encodeURIComponent(c.path)}`)
      setDiff({ path: c.path, before, after })
    })

  const visible = (changes ?? []).filter((c) => showUnchanged || c.status !== 'unchanged')

  return (
    <div className="history-panel">
      {canEdit && (
        <form
          className="history-save"
          onSubmit={(e) => {
            e.preventDefault()
            if (label.trim()) void save()
          }}
        >
          <input
            placeholder="Đặt tên phiên bản, ví dụ: Bản gửi nhà in"
            value={label}
            maxLength={120}
            onChange={(e) => setLabel(e.target.value)}
          />
          <button type="submit" className="btn-primary" disabled={busy || !label.trim()}>
            <Icon name="save" size={14} /> Lưu phiên bản
          </button>
        </form>
      )}
      <p className="settings-hint">Máy chủ tự lưu một phiên bản mỗi 5 phút khi dự án có thay đổi.</p>
      {error && (
        <div className="error" role="alert">
          {error}
        </div>
      )}
      {notice && <p className="history-notice">{notice}</p>}

      {selected ? (
        <div className="history-detail">
          <button className="btn-ghost" onClick={() => setSelected(null)}>
            <Icon name="back" size={14} /> Tất cả phiên bản
          </button>
          <h4>
            {title(selected)} · {timeLabel(selected.at)}
          </h4>
          <p className="settings-hint">So với hiện tại. {selected.author && `Người lưu: ${selected.author}.`}</p>
          {changes === null ? (
            <span className="spinner" />
          ) : (
            <>
              <label className="comment-filter">
                <input type="checkbox" checked={showUnchanged} onChange={(e) => setShowUnchanged(e.target.checked)} />{' '}
                Hiện cả tệp không đổi
              </label>
              {visible.length === 0 && <p className="settings-hint">Không có khác biệt với hiện tại.</p>}
              <ul className="history-changes">
                {visible.map((c) => (
                  <li key={c.path} className={`history-change status-${c.status}`}>
                    <button
                      className="history-change-path"
                      disabled={busy || !TEXT_EXT.test(c.path)}
                      title={TEXT_EXT.test(c.path) ? 'Xem khác biệt' : 'Tệp nhị phân: không xem được khác biệt'}
                      onClick={() => void showDiff(c)}
                    >
                      {c.path}
                    </button>
                    <span className="history-status">{STATUS_LABEL[c.status]}</span>
                    {canEdit && c.status !== 'unchanged' && (
                      <button className="btn-ghost" disabled={busy} onClick={() => void restore([c.path])}>
                        Khôi phục
                      </button>
                    )}
                  </li>
                ))}
              </ul>
              {canEdit &&
                (confirmAll ? (
                  <div className="history-confirm">
                    <p>Đưa mọi tệp về phiên bản này? Trạng thái hiện tại sẽ được lưu lại trước.</p>
                    <label className="comment-filter">
                      <input type="checkbox" checked={deleteNewer} onChange={(e) => setDeleteNewer(e.target.checked)} />{' '}
                      Xoá cả các tệp tạo sau phiên bản này
                    </label>
                    <button className="btn-primary" disabled={busy} onClick={() => void restore([])}>
                      Khôi phục cả phiên bản
                    </button>
                    <button className="btn-ghost" onClick={() => setConfirmAll(false)}>
                      Huỷ
                    </button>
                  </div>
                ) : (
                  <button className="btn-ghost" disabled={busy} onClick={() => setConfirmAll(true)}>
                    <Icon name="refresh" size={14} /> Khôi phục cả phiên bản…
                  </button>
                ))}
            </>
          )}
        </div>
      ) : list === null ? (
        <span className="spinner" />
      ) : list.length === 0 ? (
        <p className="settings-hint">Chưa có phiên bản nào.</p>
      ) : (
        <ul className="history-list">
          {list.map((s) => (
            <li key={s.id}>
              <button className={`history-item kind-${s.kind}`} onClick={() => void open(s)}>
                <strong>{title(s)}</strong>
                <span className="history-meta">
                  {timeLabel(s.at)} · {s.author}
                </span>
                <span className="history-meta">
                  {s.added > 0 && `+${s.added} `}
                  {s.modified > 0 && `~${s.modified} `}
                  {s.deleted > 0 && `−${s.deleted} `}
                  {s.files} tệp
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}

      {diff && (
        <Modal
          title={`${diff.path}: phiên bản (trái) và hiện tại (phải)`}
          onClose={() => setDiff(null)}
          className="history-diff-modal"
          footer={
            canEdit ? (
              <button className="btn-primary" disabled={busy} onClick={() => void restore([diff.path])}>
                Khôi phục tệp này
              </button>
            ) : undefined
          }
        >
          <Suspense fallback={<span className="spinner" />}>
            <HistoryDiff before={diff.before} after={diff.after} typst={diff.path.endsWith('.typ')} />
          </Suspense>
        </Modal>
      )}
    </div>
  )
}

/** The History side panel: the server's versions, and Git sync. */
export function HistoryPanel({ canEdit, onRestored }: { canEdit: boolean; onRestored: (paths: string[]) => void }) {
  const [tab, setTab] = useState<'versions' | 'git'>('versions')
  return (
    <div className="history-tabs-host">
      <div className="history-tabs" role="tablist">
        <button
          role="tab"
          aria-selected={tab === 'versions'}
          className={tab === 'versions' ? 'active' : ''}
          onClick={() => setTab('versions')}
        >
          <Icon name="clock" size={14} /> Phiên bản
        </button>
        <button
          role="tab"
          aria-selected={tab === 'git'}
          className={tab === 'git' ? 'active' : ''}
          onClick={() => setTab('git')}
        >
          <Icon name="sync" size={14} /> Git
        </button>
      </div>
      {tab === 'versions' ? (
        <VersionsPanel canEdit={canEdit} onRestored={onRestored} />
      ) : (
        <Suspense fallback={<span className="spinner" />}>
          <GitPanel canEdit={canEdit} onFilesChanged={() => onRestored([])} />
        </Suspense>
      )}
    </div>
  )
}
