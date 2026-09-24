import { useEffect, useRef, useState } from 'react'
import { Icon } from './Icon'

// Renders the current .typ file as a plain PDF fetched over a normal GET
// request (server/export_api.go's handlePreviewPdf), instead of tinymist's
// live WebSocket preview (/preview/). The WS approach doesn't work through
// this deployment's reverse proxy chain (a browser WebSocket-over-HTTP/2
// upgrade failing against the shared Traefik instance in front of it -- see
// handlePreviewPdf's doc comment), and a plain PDF fetch needs no WebSocket
// at all. Workspace.tsx bumps `version` whenever the file gets saved --
// same "updates on save, not on keystroke" behavior the WS preview had.
//
// The PDF is fetched as a blob (rather than pointing <embed> at the URL) so
// the pane can show a compiling state, keep the previous render visible
// meanwhile, and turn a compile failure into a readable message instead of
// raw JSON inside the PDF viewer.
type State =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'ready' }
  | { status: 'error'; message: string }

export function PreviewPane({ path, version }: { path: string | null; version: number }) {
  const [state, setState] = useState<State>({ status: 'idle' })
  const [pdfUrl, setPdfUrl] = useState<string | null>(null)
  const [nonce, setNonce] = useState(0)
  const urlRef = useRef<string | null>(null)

  const isTyp = !!path && path.endsWith('.typ')

  useEffect(() => {
    if (!path || !path.endsWith('.typ')) return
    const ctrl = new AbortController()
    void (async () => {
      setState({ status: 'loading' })
      try {
        const res = await fetch(`/api/preview/pdf?path=${encodeURIComponent(path)}`, {
          credentials: 'include',
          signal: ctrl.signal,
        })
        if (!res.ok) {
          let message = res.statusText
          try {
            const body = await res.json()
            if (body?.error) message = body.error
          } catch {
            // non-JSON error body
          }
          setState({ status: 'error', message })
          return
        }
        const url = URL.createObjectURL(await res.blob())
        if (urlRef.current) URL.revokeObjectURL(urlRef.current)
        urlRef.current = url
        setPdfUrl(url)
        setState({ status: 'ready' })
      } catch (err) {
        if (!ctrl.signal.aborted) setState({ status: 'error', message: err instanceof Error ? err.message : String(err) })
      }
    })()
    return () => ctrl.abort()
  }, [path, version, nonce])

  // Drop the old render when switching files; free the blob on unmount.
  useEffect(() => {
    return () => {
      if (urlRef.current) URL.revokeObjectURL(urlRef.current)
      urlRef.current = null
      setPdfUrl(null)
    }
  }, [path])

  if (!isTyp) {
    return (
      <div className="preview-pane">
        <div className="preview-loading">
          <Icon name="eye" size={28} />
          <span>Chọn một file .typ để xem trước</span>
        </div>
      </div>
    )
  }

  return (
    <div className="preview-pane">
      <div className="preview-toolbar">
        <span className={`preview-status preview-status-${state.status}`}>
          {state.status === 'loading' && (
            <>
              <span className="spinner" /> Đang biên dịch…
            </>
          )}
          {state.status === 'ready' && (
            <>
              <Icon name="check" size={13} /> Bản xem trước PDF
            </>
          )}
          {state.status === 'error' && (
            <>
              <Icon name="error" size={13} /> Lỗi biên dịch
            </>
          )}
        </span>
        <span className="preview-toolbar-spacer" />
        <button
          className="btn-ghost preview-tool"
          title="Biên dịch lại"
          aria-label="Biên dịch lại"
          onClick={() => setNonce((n) => n + 1)}
          disabled={state.status === 'loading'}
        >
          <Icon name="refresh" size={14} />
        </button>
        <button
          className="btn-ghost preview-tool"
          title="Mở PDF trong tab mới"
          aria-label="Mở PDF trong tab mới"
          disabled={!pdfUrl}
          onClick={() => pdfUrl && window.open(pdfUrl, '_blank')}
        >
          <Icon name="external" size={14} />
        </button>
      </div>

      <div className="preview-body">
        {pdfUrl && <embed className="preview-pdf" src={pdfUrl} type="application/pdf" />}
        {!pdfUrl && state.status === 'loading' && (
          <div className="preview-loading">
            <span className="spinner spinner-lg" />
            <span>Đang biên dịch tài liệu…</span>
          </div>
        )}
        {state.status === 'error' && (
          <div className="preview-error" role="alert">
            <div className="preview-error-title">
              <Icon name="error" size={16} /> Không biên dịch được tài liệu
            </div>
            <pre className="preview-error-message">{state.message}</pre>
            <p className="preview-error-hint">
              Sửa lỗi trong trình soạn thảo rồi lưu (Ctrl+S), hoặc nhờ Trợ lý AI "Sửa lỗi biên dịch".
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
