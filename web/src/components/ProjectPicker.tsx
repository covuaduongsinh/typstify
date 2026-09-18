import { useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { RecentProject } from '../api/types'

export function ProjectPicker({ onOpened }: { onOpened: (path: string) => void }) {
  const [recent, setRecent] = useState<RecentProject[]>([])
  const [path, setPath] = useState('')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .get<RecentProject[]>('/api/workspace/recent')
      .then((list) => setRecent(list ?? []))
      .catch(() => setRecent([]))
  }, [])

  const open = async (p: string) => {
    setError(null)
    try {
      const res = await api.post<{ path: string }>('/api/workspace/open', { path: p })
      onOpened(res.path)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to open project')
    }
  }

  const create = async () => {
    setError(null)
    try {
      const res = await api.post<{ path: string }>('/api/workspace/create', { path })
      onOpened(res.path)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to create project')
    }
  }

  return (
    <div className="project-picker">
      <h1>Typstify</h1>

      {recent.length > 0 && (
        <div className="recent-projects">
          <h2>Recent projects</h2>
          <ul>
            {recent.map((p) => (
              <li key={p.Path}>
                <button onClick={() => open(p.Path)}>{p.Path}</button>
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="open-project">
        <h2>Open or create a project</h2>
        <input
          placeholder="/path/to/project"
          value={path}
          onChange={(e) => setPath(e.target.value)}
        />
        <button onClick={() => open(path)}>Open</button>
        <button onClick={create}>Create new</button>
      </div>

      {error && <div className="error">{error}</div>}
    </div>
  )
}
