import { useEffect, useState } from 'react'
import './App.css'
import { ApiError, api } from './api/client'
import type { AuthStatus, ProjectInfo } from './api/types'
import { LoginPage } from './components/LoginPage'
import { ProjectPicker } from './components/ProjectPicker'
import { Workspace } from './components/Workspace'

type Screen = { kind: 'loading' } | { kind: 'login' } | { kind: 'pickProject' } | { kind: 'workspace'; path: string; openFile?: string }

/** takeJoinToken reads a share-link token (?join=...) once and strips it
 * from the address bar, so a reload or a copied URL doesn't leak it. */
function takeJoinToken(): string | null {
  const url = new URL(window.location.href)
  const token = url.searchParams.get('join')
  if (token) {
    url.searchParams.delete('join')
    window.history.replaceState(null, '', url.pathname + url.search + url.hash)
  }
  return token
}

function App() {
  const [screen, setScreen] = useState<Screen>({ kind: 'loading' })
  const [joinToken, setJoinToken] = useState<string | null>(() => takeJoinToken())
  const [notice, setNotice] = useState<string | null>(null)

  useEffect(() => {
    api
      .get<AuthStatus>('/api/auth/status')
      .then(async (status) => {
        if (!status.authenticated) {
          setScreen({ kind: 'login' })
          return
        }
        await afterLogin()
      })
      .catch(() => setScreen({ kind: 'login' }))
    // afterLogin only reads state captured at mount; run once.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const checkCurrentProject = async () => {
    try {
      const current = await api.get<{ path: string }>('/api/workspace/current')
      setScreen(current.path ? { kind: 'workspace', path: current.path } : { kind: 'pickProject' })
    } catch {
      setScreen({ kind: 'pickProject' })
    }
  }

  const afterLogin = async () => {
    if (joinToken) {
      setJoinToken(null)
      try {
        const project = await api.post<ProjectInfo>('/api/projects/join', { token: joinToken })
        const opened = await api.post<{ path: string }>(`/api/projects/${project.id}/open`)
        setScreen({ kind: 'workspace', path: opened.path })
        return
      } catch (err) {
        setNotice(err instanceof ApiError ? err.message : 'Không tham gia được dự án')
        setScreen({ kind: 'pickProject' })
        return
      }
    }
    await checkCurrentProject()
  }

  switch (screen.kind) {
    case 'loading':
      return (
        <div className="app-loading" role="status" aria-label="Đang tải">
          <span className="spinner spinner-lg" />
        </div>
      )
    case 'login':
      return <LoginPage onLoggedIn={afterLogin} />
    case 'pickProject':
      return <ProjectPicker notice={notice} onOpened={(path, openFile) => setScreen({ kind: 'workspace', path, openFile })} />
    case 'workspace':
      return (
        <Workspace
          projectPath={screen.path}
          initialFile={screen.openFile}
          onCloseProject={() => setScreen({ kind: 'pickProject' })}
        />
      )
  }
}

export default App
