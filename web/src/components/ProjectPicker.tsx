import { useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { RecentProject } from '../api/types'
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

export function ProjectPicker({ onOpened }: { onOpened: (path: string) => void }) {
  const [recent, setRecent] = useState<RecentProject[]>([])
  const [path, setPath] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [isDropboxImportOpen, setIsDropboxImportOpen] = useState(false)
  const [theme, toggleTheme] = useTheme()

  useEffect(() => {
    api
      .get<RecentProject[]>('/api/workspace/recent')
      .then((list) => setRecent(list ?? []))
      .catch(() => setRecent([]))
  }, [])

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
          {error && (
            <div className="error" role="alert">
              {error}
            </div>
          )}
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
