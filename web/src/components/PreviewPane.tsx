import { useEffect, useRef, useState } from 'react'
import { calculateLineRatio, calculateScrollTarget, type PageLayoutInfo } from '../lib/scrollSync'
import { Icon } from './Icon'

type Status = 'idle' | 'loading' | 'ready' | 'error'
type ViewMode = 'svg' | 'pdf'

interface PreviewPaneProps {
  path: string | null
  version: number
  liveContent?: string | null
  cursor?: { line: number; col: number; totalLines: number } | null
}

export function PreviewPane({ path, version, liveContent, cursor }: PreviewPaneProps) {
  const [status, setStatus] = useState<Status>('idle')
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [pages, setPages] = useState<string[]>([])
  const [pageCount, setPageCount] = useState<number>(0)
  const [viewMode, setViewMode] = useState<ViewMode>('svg')
  const [syncScroll, setSyncScroll] = useState<boolean>(true)
  const [zoom, setZoom] = useState<number>(100)
  const [fitWidth, setFitWidth] = useState<boolean>(false)
  const [pdfUrl, setPdfUrl] = useState<string | null>(null)
  const [nonce, setNonce] = useState(0)

  const scrollContainerRef = useRef<HTMLDivElement | null>(null)
  const pageRefs = useRef<(HTMLDivElement | null)[]>([])
  const manualScrollTimerRef = useRef<number | undefined>(undefined)
  const isUserScrollingRef = useRef<boolean>(false)
  const lastRenderedContentRef = useRef<string | null>(null)

  const isTyp = !!path && path.endsWith('.typ')

  // Fetch / render live SVG or PDF
  useEffect(() => {
    if (!path || !path.endsWith('.typ')) return
    const ctrl = new AbortController()

    void (async () => {
      setStatus('loading')
      try {
        if (viewMode === 'svg') {
          const res = await fetch('/api/preview/render', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            signal: ctrl.signal,
            body: JSON.stringify({
              path,
              content: liveContent ?? undefined,
              format: 'svg',
            }),
          })

          if (!res.ok) {
            let msg = res.statusText
            try {
              const body = await res.json()
              if (body?.error) msg = body.error
            } catch {
              // ignore
            }
            if (!ctrl.signal.aborted) {
              setErrorMessage(msg)
              setStatus('error')
            }
            return
          }

          const data = await res.json()
          if (!ctrl.signal.aborted) {
            if (data.ok && Array.isArray(data.pages)) {
              setPages(data.pages)
              setPageCount(data.pageCount || data.pages.length)
              setErrorMessage(null)
              setStatus('ready')
              lastRenderedContentRef.current = liveContent ?? ''
            } else {
              setErrorMessage(data.error || 'Biên dịch không thành công')
              setStatus('error')
            }
          }
        } else {
          // Native PDF mode
          const res = await fetch(`/api/preview/pdf?path=${encodeURIComponent(path)}`, {
            credentials: 'include',
            signal: ctrl.signal,
          })
          if (!res.ok) {
            let msg = res.statusText
            try {
              const body = await res.json()
              if (body?.error) msg = body.error
            } catch {
              // non-JSON
            }
            if (!ctrl.signal.aborted) {
              setErrorMessage(msg)
              setStatus('error')
            }
            return
          }
          const blobUrl = URL.createObjectURL(await res.blob())
          if (!ctrl.signal.aborted) {
            if (pdfUrl) URL.revokeObjectURL(pdfUrl)
            setPdfUrl(blobUrl)
            setErrorMessage(null)
            setStatus('ready')
          }
        }
      } catch (err) {
        if (!ctrl.signal.aborted) {
          setErrorMessage(err instanceof Error ? err.message : String(err))
          setStatus('error')
        }
      }
    })()

    return () => ctrl.abort()
  }, [path, version, liveContent, nonce, viewMode])

  // Sync scroll to cursor position
  useEffect(() => {
    if (!syncScroll || !cursor || !scrollContainerRef.current || viewMode !== 'svg') return
    if (isUserScrollingRef.current) return // User is actively scrolling manually

    const container = scrollContainerRef.current
    const lineRatio = calculateLineRatio(cursor.line, cursor.totalLines)

    // Gather rendered page bounds
    const pageLayouts: PageLayoutInfo[] = []
    for (let i = 0; i < pages.length; i++) {
      const el = pageRefs.current[i]
      if (el) {
        pageLayouts.push({ top: el.offsetTop, height: el.offsetHeight })
      }
    }

    const targetY = calculateScrollTarget(
      lineRatio,
      pageLayouts,
      container.clientHeight,
      container.scrollHeight,
    )

    container.scrollTo({
      top: targetY,
      behavior: 'smooth',
    })
  }, [cursor, syncScroll, pages, viewMode])

  // Track manual scroll by user to avoid jumping while reading
  const handleScroll = () => {
    isUserScrollingRef.current = true
    window.clearTimeout(manualScrollTimerRef.current)
    manualScrollTimerRef.current = window.setTimeout(() => {
      isUserScrollingRef.current = false
    }, 1500)
  }

  // Cleanup blob URL on unmount or file switch
  useEffect(() => {
    return () => {
      if (pdfUrl) URL.revokeObjectURL(pdfUrl)
      window.clearTimeout(manualScrollTimerRef.current)
    }
  }, [path, pdfUrl])

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

  const handleZoomIn = () => {
    setFitWidth(false)
    setZoom((z) => Math.min(200, z + 15))
  }
  const handleZoomOut = () => {
    setFitWidth(false)
    setZoom((z) => Math.max(50, z - 15))
  }
  const handleZoomReset = () => {
    setFitWidth(false)
    setZoom(100)
  }
  const handleToggleFit = () => {
    setFitWidth((f) => !f)
  }

  return (
    <div className="preview-pane">
      <div className="preview-toolbar">
        {/* Status indicator */}
        <span className={`preview-status preview-status-${status}`}>
          {status === 'loading' && (
            <>
              <span className="spinner" /> Đang cập nhật…
            </>
          )}
          {status === 'ready' && (
            <>
              <Icon name="check" size={13} /> {viewMode === 'svg' ? `Xem trực tiếp (${pageCount} trang)` : 'Bản xem trước PDF'}
            </>
          )}
          {status === 'error' && (
            <>
              <Icon name="error" size={13} /> Lỗi biên dịch
            </>
          )}
        </span>

        <span className="preview-toolbar-spacer" />

        {/* Scroll Sync Toggle */}
        {viewMode === 'svg' && (
          <button
            className={`btn-ghost preview-tool preview-tool-btn${syncScroll ? ' active' : ''}`}
            title={syncScroll ? 'Đang bật đồng bộ cuộn theo con trỏ (Bấm để tắt)' : 'Đang tắt đồng bộ cuộn (Bấm để bật)'}
            aria-label="Đồng bộ cuộn"
            aria-pressed={syncScroll}
            onClick={() => setSyncScroll((s) => !s)}
          >
            <Icon name={syncScroll ? 'link' : 'unlink'} size={14} />
            <span className="preview-tool-label">Đồng bộ cuộn</span>
          </button>
        )}

        {/* Zoom controls for SVG mode */}
        {viewMode === 'svg' && (
          <div className="preview-zoom-group">
            <button
              className="btn-ghost preview-tool"
              title="Thu nhỏ (-)"
              aria-label="Thu nhỏ"
              onClick={handleZoomOut}
            >
              <Icon name="zoom-out" size={14} />
            </button>
            <button
              className="btn-ghost preview-tool preview-zoom-value"
              title="Cỡ chuẩn 100%"
              aria-label="Cỡ chuẩn 100%"
              onClick={handleZoomReset}
            >
              {fitWidth ? 'Vừa khung' : `${zoom}%`}
            </button>
            <button
              className="btn-ghost preview-tool"
              title="Phóng to (+)"
              aria-label="Phóng to"
              onClick={handleZoomIn}
            >
              <Icon name="zoom-in" size={14} />
            </button>
            <button
              className={`btn-ghost preview-tool${fitWidth ? ' active' : ''}`}
              title="Vừa chiều rộng khung nhìn"
              aria-label="Vừa chiều rộng"
              onClick={handleToggleFit}
            >
              <Icon name="maximize-2" size={14} />
            </button>
          </div>
        )}

        {/* View mode toggle (SVG Paper vs PDF Embed) */}
        <div className="preview-mode-group">
          <button
            className={`btn-ghost preview-tool preview-mode-btn${viewMode === 'svg' ? ' active' : ''}`}
            title="Chế độ trang SVG (Cập nhật tức thì, mượt mà)"
            onClick={() => setViewMode('svg')}
          >
            Trang in
          </button>
          <button
            className={`btn-ghost preview-tool preview-mode-btn${viewMode === 'pdf' ? ' active' : ''}`}
            title="Chế độ xem nhúng PDF gốc"
            onClick={() => setViewMode('pdf')}
          >
            PDF
          </button>
        </div>

        {/* Reload */}
        <button
          className="btn-ghost preview-tool"
          title="Biên dịch lại ngay"
          aria-label="Biên dịch lại"
          onClick={() => setNonce((n) => n + 1)}
          disabled={status === 'loading'}
        >
          <Icon name="refresh" size={14} />
        </button>

        {/* Open PDF in new tab */}
        <button
          className="btn-ghost preview-tool"
          title="Mở PDF trong tab mới để in ấn"
          aria-label="Mở PDF trong tab mới"
          onClick={() => window.open(`/api/preview/pdf?path=${encodeURIComponent(path)}`, '_blank')}
        >
          <Icon name="external" size={14} />
        </button>
      </div>

      <div className="preview-body">
        {/* SVG Paper View Mode */}
        {viewMode === 'svg' && (
          <div
            className="preview-scroll-container"
            ref={scrollContainerRef}
            onScroll={handleScroll}
          >
            {pages.length > 0 ? (
              <div
                className={`preview-pages-wrapper${fitWidth ? ' fit-width' : ''}`}
                style={!fitWidth && zoom !== 100 ? { transform: `scale(${zoom / 100})`, transformOrigin: 'top center' } : undefined}
              >
                {pages.map((svgContent, idx) => (
                  <div
                    key={idx}
                    className="preview-page-card"
                    data-page={idx + 1}
                    ref={(el) => {
                      pageRefs.current[idx] = el
                    }}
                  >
                    <div className="preview-page-header">
                      <span>Trang {idx + 1} / {pageCount}</span>
                    </div>
                    <div
                      className="preview-svg-content"
                      dangerouslySetInnerHTML={{ __html: svgContent }}
                    />
                  </div>
                ))}
              </div>
            ) : status === 'loading' ? (
              <div className="preview-loading">
                <span className="spinner spinner-lg" />
                <span>Đang kết xuất tài liệu xem trước…</span>
              </div>
            ) : null}
          </div>
        )}

        {/* PDF Embed Mode */}
        {viewMode === 'pdf' && (
          <div className="preview-pdf-wrapper">
            {pdfUrl && <embed className="preview-pdf" src={pdfUrl} type="application/pdf" />}
            {!pdfUrl && status === 'loading' && (
              <div className="preview-loading">
                <span className="spinner spinner-lg" />
                <span>Đang biên dịch PDF…</span>
              </div>
            )}
          </div>
        )}

        {/* Error notification banner / popup */}
        {errorMessage && (
          <div className="preview-error-banner" role="alert">
            <div className="preview-error-header">
              <Icon name="error" size={15} />
              <span>Lỗi cú pháp Typst</span>
              <button
                className="btn-ghost preview-error-close"
                onClick={() => setErrorMessage(null)}
                title="Đóng thông báo"
              >
                <Icon name="x" size={13} />
              </button>
            </div>
            <pre className="preview-error-message">{errorMessage}</pre>
          </div>
        )}
      </div>
    </div>
  )
}
