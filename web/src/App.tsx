import { useEffect, useState } from 'react'
import './App.css'
import { api } from './api/client'
import type { AuthStatus } from './api/types'
import { LoginPage } from './components/LoginPage'
import { ProjectPicker } from './components/ProjectPicker'
import { Workspace } from './components/Workspace'

type Screen = { kind: 'loading' } | { kind: 'login' } | { kind: 'pickProject' } | { kind: 'workspace'; path: string }

function App() {
  const [screen, setScreen] = useState<Screen>({ kind: 'loading' })

  useEffect(() => {
    api
      .get<AuthStatus>('/api/auth/status')
      .then(async (status) => {
        if (!status.authenticated) {
          setScreen({ kind: 'login' })
          return
        }
        await checkCurrentProject()
      })
      .catch(() => setScreen({ kind: 'login' }))
  }, [])

  const checkCurrentProject = async () => {
    try {
      const current = await api.get<{ path: string }>('/api/workspace/current')
      setScreen(current.path ? { kind: 'workspace', path: current.path } : { kind: 'pickProject' })
    } catch {
      setScreen({ kind: 'pickProject' })
    }
  }

  switch (screen.kind) {
    case 'loading':
      return <div className="app-loading">Loading…</div>
    case 'login':
      return <LoginPage onLoggedIn={checkCurrentProject} />
    case 'pickProject':
      return <ProjectPicker onOpened={(path) => setScreen({ kind: 'workspace', path })} />
    case 'workspace':
      return <Workspace projectPath={screen.path} onCloseProject={() => setScreen({ kind: 'pickProject' })} />
  }
}

export default App
