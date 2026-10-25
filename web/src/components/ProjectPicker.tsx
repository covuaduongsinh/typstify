import { useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'
import { ROLE_LABELS, type ProjectInfo, type RecentProject } from '../api/types'
import { useTheme } from '../lib/theme'
import { BrandMark } from './BrandMark'
import { DropboxImportModal } from './DropboxImportModal'
import { Icon } from './Icon'

const relTime = new Intl.RelativeTimeFormat('vi', { numeric: 'auto' })

function timeAgo(iso: string): string {
  const then = Date.parse(iso)
  if (Number.isNaN(then)) return ''
  const minutes = Math.round((then - Date.now()) / 60000)
  if (Math.abs(minutes) < 60) return relTime.format(minutes, 'minute')
  const hours = Math.round(minutes / 60)
  if (Math.abs(hours) < 24) return relTime.format(hours, 'hour')
  return relTime.format(Math.round(hours / 24), 'day')
}

const baseName = (p: string) => p.split('/').filter(Boolean).pop() ?? p

/** A pasted share link (".../?join=TOKEN") or the bare token itself. */
function extractJoinToken(input: string): string {
  const s = input.trim()
  try {
    return new URL(s).searchParams.get('join') ?? s
  } catch {
    return s
  }
}

export function ProjectPicker({ onOpened, notice }: { onOpened: (path: string) => void; notice?: string | null }) {
  const [recent, setRecent] = useState<RecentProject[]>([])
  const [mine, setMine] = useState<ProjectInfo[]>([])
  const [newName, setNewName] = useState('')
  const [joinInput, setJoinInput] = useState('')
  const [path, setPath] = useState('')
  const [error, setError] = useState<string | null>(notice ?? null)
  const [busy, setBusy] = useState(false)
  const [isDropboxImportOpen, setIsDropboxImportOpen] = useState(false)
  const [theme, toggleTheme] = useTheme()

  useEffect(() => {
    api
      .get<RecentProject[]>('/api/workspace/recent')
      .then((list) => setRecent(list ?? []))
      .catch(() => setRecent([]))
    api
      .get<ProjectInfo[]>('/api/projects')
      .then((list) => setMine(list ?? []))
      .catch(() => setMine([]))
  }, [])

  const run = async (fallback: string, fn: () => Promise<string>) => {
    setError(null)
    setBusy(true)
    try {
      onOpened(await fn())
    } catch (err) {
      setError(err instanceof ApiError ? err.message : fallback)
    } finally {
      setBusy(false)
    }
  }

  const openRegistered = (id: string) =>
    run('Không mở được dự án', async () => (await api.post<{ path: string }>(`/api/projects/${id}/open`)).path)

  const createNamed = () =>
    run('Không tạo được dự án', async () => {
      const p = await api.post<ProjectInfo>('/api/projects', { name: newName.trim() })
      return (await api.post<{ path: string }>(`/api/projects/${p.id}/open`)).path
    })

  const join = () =>
    run('Không tham gia được dự án', async () => {
      const p = await api.post<ProjectInfo>('/api/projects/join', { token: extractJoinToken(joinInput) })
      return (await api.post<{ path: string }>(`/api/projects/${p.id}/open`)).path
    })

  const call = async (endpoint: 'open' | 'create', p: string) => {
    setError(null)
    setBusy(true)
    try {
      const res = await api.post<{ path: string }>(`/api/workspace/${endpoint}`, { path: p })
      onOpened(res.path)
    } catch (err) {
      const fallback = endpoint === 'open' ? 'Không mở được dự án' : 'Không tạo được dự án'
      setError(err instanceof ApiError ? err.message : fallback)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="entry-page picker-page">
      <header className="picker-topbar">
        <BrandMark size={28} withText />
        <button
          className="btn-ghost hdr-btn theme-toggle"
          onClick={toggleTheme}
          aria-label={theme === 'dark' ? 'Chuyển sang giao diện sáng' : 'Chuyển sang giao diện tối'}
          title={theme === 'dark' ? 'Chuyển sang giao diện sáng' : 'Chuyển sang giao diện tối'}
        >
          <Icon name={theme === 'dark' ? 'sun' : 'moon'} />
        </button>
      </header>

      <main className="picker-main">
        <h1 className="picker-heading">Chọn dự án</h1>
        <p className="picker-sub">Mỗi dự án là một thư mục chứa sách, giáo trình hoặc tuyển tập bài tập cờ vua.</p>

        {error && (
          <div className="error" role="alert">
            {error}
          </div>
        )}

        <section className="picker-section" aria-labelledby="mine-h">
          <h2 id="mine-h">Dự án của tôi</h2>
          {mine.length > 0 ? (
            <div className="recent-grid">
              {mine.map((p) => (
                <button key={p.id} className="recent-card" onClick={() => openRegistered(p.id)} disabled={busy}>
                  <Icon name="folder" size={22} className="recent-icon" />
                  <span className="recent-name">{p.name}</span>
                  <span className="recent-path">
                    {ROLE_LABELS[p.role]}
                    {p.role !== 'owner' && ` · của ${p.owner}`}
                  </span>
                  <span className="recent-time">{timeAgo(p.updatedAt)}</span>
                </button>
              ))}
            </div>
          ) : (
            <p className="entry-hint">Chưa có dự án nào. Tạo dự án mới hoặc dán liên kết chia sẻ bên dưới.</p>
          )}
          <form
            className="open-project"
            onSubmit={(e) => {
              e.preventDefault()
              if (newName.trim()) void createNamed()
            }}
          >
            <label className="entry-field">
              <span>Tạo dự án mới</span>
              <input placeholder="Ví dụ: Giáo trình Khai cuộc" value={newName} onChange={(e) => setNewName(e.target.value)} />
            </label>
            <div className="open-actions">
              <button type="submit" className="btn-primary" disabled={busy || !newName.trim()}>
                <Icon name="folder-plus" size={15} /> Tạo
              </button>
            </div>
          </form>
          <form
            className="open-project"
            onSubmit={(e) => {
              e.preventDefault()
              if (joinInput.trim()) void join()
            }}
          >
            <label className="entry-field">
              <span>Tham gia bằng liên kết chia sẻ</span>
              <input
                placeholder="Dán liên kết …?join=… hoặc mã"
                value={joinInput}
                onChange={(e) => setJoinInput(e.target.value)}
              />
            </label>
            <div className="open-actions">
              <button type="submit" disabled={busy || !joinInput.trim()}>
                <Icon name="folder-open" size={15} /> Tham gia
              </button>
            </div>
          </form>
        </section>

        {recent.length > 0 && (
          <section className="picker-section" aria-labelledby="recent-h">
            <h2 id="recent-h">Gần đây</h2>
            <div className="recent-grid">
              {recent.map((p) => (
                <button key={p.Path} className="recent-card" onClick={() => call('open', p.Path)} disabled={busy}>
                  <Icon name="folder" size={22} className="recent-icon" />
                  <span className="recent-name">{baseName(p.Path)}</span>
                  <span className="recent-path" title={p.Path}>
                    {p.Path}
                  </span>
                  {p.LastAccessAt && <span className="recent-time">{timeAgo(p.LastAccessAt)}</span>}
                </button>
              ))}
            </div>
          </section>
        )}

        <section className="picker-section entry-card open-card" aria-labelledby="open-h">
          <h2 id="open-h">Mở hoặc tạo dự án</h2>
          <form
            className="open-project"
            onSubmit={(e) => {
              e.preventDefault()
              if (path.trim()) void call('open', path.trim())
            }}
          >
            <label className="entry-field">
              <span>Đường dẫn thư mục</span>
              <input
                placeholder="/data/ten-du-an"
                value={path}
                onChange={(e) => setPath(e.target.value)}
                aria-describedby="path-hint"
              />
            </label>
            <p id="path-hint" className="entry-hint">
              Trên máy chủ, dự án phải nằm trong thư mục <code>/data</code>.
            </p>
            <div className="open-actions">
              <button type="submit" disabled={busy || !path.trim()}>
                <Icon name="folder-open" size={15} /> Mở
              </button>
              <button
                type="button"
                className="btn-primary"
                disabled={busy || !path.trim()}
                onClick={() => call('create', path.trim())}
              >
                <Icon name="folder-plus" size={15} /> Tạo dự án mới
              </button>
              <button
                type="button"
                className="btn-secondary"
                disabled={busy}
                onClick={() => setIsDropboxImportOpen(true)}
              >
                <Icon name="cloud-download" size={15} /> Nhập từ Dropbox
              </button>
            </div>
          </form>
        </section>
      </main>

      {isDropboxImportOpen && (
        <DropboxImportModal
          isOpen={isDropboxImportOpen}
          onClose={() => setIsDropboxImportOpen(false)}
          onImported={(p) => {
            setIsDropboxImportOpen(false)
            onOpened(p)
          }}
        />
      )}
    </div>
  )
}
