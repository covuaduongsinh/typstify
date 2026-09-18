import { useEffect, useState } from 'react'
import { api } from '../api/client'

export function PreviewPane() {
  const [ready, setReady] = useState(false)

  useEffect(() => {
    let cancelled = false
    let timer: number | undefined

    const poll = async () => {
      try {
        const status = await api.get<{ ready: boolean }>('/api/preview/status')
        if (cancelled) return
        setReady(status.ready)
        if (!status.ready) timer = window.setTimeout(poll, 1000)
      } catch {
        if (!cancelled) timer = window.setTimeout(poll, 2000)
      }
    }
    poll()

    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [])

  return (
    <div className="preview-pane">
      {ready ? (
        <iframe title="Typst preview" src="/preview/" />
      ) : (
        <div className="preview-loading">Waiting for preview server…</div>
      )}
    </div>
  )
}
