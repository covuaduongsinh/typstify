import { useEffect, useRef, useState } from 'react'
import './App.css'
import { ApiError, api } from './api/client'
import type { AuthStatus, ProjectInfo } from './api/types'
import { LoginPage } from './components/LoginPage'
import { ProjectPicker } from './components/ProjectPicker'
import { Workspace } from './components/Workspace'

type Screen =
  | { kind: 'loading' }
  | { kind: 'unreachable' }
  | { kind: 'login' }
  | { kind: 'desktopSignedOut' }
  | { kind: 'pickProject' }
  | { kind: 'workspace'; path: string; openFile?: string }

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

  // Connection attempts: each one supersedes the previous, so an "online"
  // event and a click on "Thử lại" never run the login flow twice.
  const [attempt, setAttempt] = useState(0)
  const failuresRef = useRef(0)
  useEffect(() => {
    let current = true
    api
      .get<AuthStatus>('/api/auth/status')
      .then(async (status) => {
        if (!current) return
        failuresRef.current = 0
        if (!status.authenticated) {
          // The desktop app signs in once at launch; its account has no
          // password anyone knows.
          setScreen(status.desktop ? { kind: 'desktopSignedOut' } : { kind: 'login' })
          return
        }
        await afterLogin()
      })
      .catch((err: unknown) => {
        if (!current) return
        // Only an explicit refusal asks for a password. Offline, the server
        // down (or the proxy in front of it: 502/503), any other error:
        // show that and keep trying.
        if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {
          setScreen({ kind: 'login' })
          return
        }
        failuresRef.current++
        setScreen({ kind: 'unreachable' })
      })
    return () => {
      current = false
    }
    // afterLogin only reads state captured at mount; run once per attempt.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [attempt])

  useEffect(() => {
    if (screen.kind !== 'unreachable') return
    const retry = () => setAttempt((n) => n + 1)
    // Back off: 5 s, 10 s, 20 s… up to a minute; at once when back online.
    const timer = window.setTimeout(retry, Math.min(60_000, 5_000 * 2 ** Math.max(0, failuresRef.current - 1)))
    window.addEventListener('online', retry)
    return () => {
      window.clearTimeout(timer)
      window.removeEventListener('online', retry)
    }
  }, [screen, attempt])

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
    case 'unreachable':
      return (
        <div className="entry-page app-unreachable" role="alert">
          <h1>Không kết nối được máy chủ</h1>
          <p>Mạng đang ngắt hoặc máy chủ đang tạm dừng. Ứng dụng tự thử lại sau ít giây.</p>
          <button
            className="btn-primary"
            onClick={() => {
              setScreen({ kind: 'loading' })
              setAttempt((n) => n + 1)
            }}
          >
            Thử lại
          </button>
        </div>
      )
    case 'desktopSignedOut':
      return (
        <div className="entry-page app-unreachable" role="alert">
          <h1>Phiên làm việc đã kết thúc</h1>
          <p>Hãy đóng rồi mở lại ứng dụng để tiếp tục.</p>
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
