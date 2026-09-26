import { lazy, Suspense, useEffect, useRef, useState } from 'react'
import { api } from '../api/client'
import type { TreeEntry } from '../api/types'
import { useTranslations } from '../lib/i18n'
import { useTheme } from '../lib/theme'
import { CHESSBOOK_IMPORT } from '../lib/typst'
import { BrandMark } from './BrandMark'
import { ChessToolbar } from './ChessToolbar'
import type { EditorHandle } from './Editor'
import { ExportButton } from './ExportButton'
import { FileTree } from './FileTree'
import { Icon, type IconName } from './Icon'
import { Modal } from './Modal'
import { NewDocModal } from './NewDocModal'
import { PreviewPane } from './PreviewPane'
import { Resizer } from './Resizer'
import { StatusBar, type DiagnosticCounts } from './StatusBar'

// Heavy parts load on demand, so the login/project screens don't pull in
// CodeMirror, react-markdown or the chess dialogs.
const Editor = lazy(() => import('./Editor').then((m) => ({ default: m.Editor })))
const AgentChat = lazy(() => import('./AgentChat').then((m) => ({ default: m.AgentChat })))
const PackageManager = lazy(() => import('./PackageManager').then((m) => ({ default: m.PackageManager })))
const SettingsPanel = lazy(() => import('./SettingsPanel').then((m) => ({ default: m.SettingsPanel })))
const ChessBoardModal = lazy(() => import('./ChessBoardModal').then((m) => ({ default: m.ChessBoardModal })))
const PgnImportModal = lazy(() => import('./PgnImportModal').then((m) => ({ default: m.PgnImportModal })))
const DropboxSyncModal = lazy(() => import('./DropboxSyncModal').then((m) => ({ default: m.DropboxSyncModal })))

function PanelLoading() {
  return (
    <div className="editor-state">
      <span className="spinner" /> Đang tải…
    </div>
  )
}

const I18N_KEYS = ['AI Assistant', 'Settings', 'Export']

type SidePanel = 'agent' | 'packages' | 'settings' | null

const FILETREE_DEFAULT = 230
const FILETREE_MIN = 160
const FILETREE_MAX = 500
const SIDEPANEL_DEFAULT = 340
const SIDEPANEL_MIN = 260
const SIDEPANEL_MAX = 650
const RATIO_MIN = 0.15
const RATIO_MAX = 0.85

const clamp = (v: number, lo: number, hi: number) => Math.max(lo, Math.min(hi, v))

// localStorage can throw (private mode, blocked site data); layout
// preferences are a convenience, so failures just fall back to defaults.
function loadPref(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}
function savePref(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    // ignore
  }
}

const NEW_DOC_TEMPLATE = `${CHESSBOOK_IMPORT}

#show: chess-book-init.with(
  title: "TÊN TÀI LIỆU",
  subtitle: "",
  author: "CLB Cờ vua Dương Sinh",
  paper-size: "a5",
)

= Tiêu đề chương

Nội dung bài viết…
`

export function Workspace({ projectPath, onCloseProject }: { projectPath: string; onCloseProject: () => void }) {
  const [activePath, setActivePath] = useState<string | null>(null)
  const [content, setContent] = useState<string | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [loadAttempt, setLoadAttempt] = useState(0)
  const [saveError, setSaveError] = useState<string | null>(null)
  // A navigation (open another file / close the project) waiting on the
  // unsaved-changes dialog.
  const [pendingNav, setPendingNav] = useState<{ kind: 'open'; path: string } | { kind: 'close' } | null>(null)
  const [savingForNav, setSavingForNav] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [lastSaved, setLastSaved] = useState<Date | null>(null)
  const [cursor, setCursor] = useState<{ line: number; col: number } | null>(null)
  const [diagnostics, setDiagnostics] = useState<DiagnosticCounts>({ errors: 0, warnings: 0 })
  const [sidePanel, setSidePanel] = useState<SidePanel>('agent')
  const [previewVersion, setPreviewVersion] = useState(0)
  const [isBoardOpen, setIsBoardOpen] = useState(false)
  const [isPgnOpen, setIsPgnOpen] = useState(false)
  // Once opened, the chess dialogs stay mounted (hidden) so the position
  // being set up survives closing and reopening.
  const [boardUsed, setBoardUsed] = useState(false)
  const [pgnUsed, setPgnUsed] = useState(false)
  const openBoard = () => {
    setBoardUsed(true)
    setIsBoardOpen(true)
  }
  const openPgn = () => {
    setPgnUsed(true)
    setIsPgnOpen(true)
  }
  const [isNewDocOpen, setIsNewDocOpen] = useState(false)
  const [isDropboxOpen, setIsDropboxOpen] = useState(false)
  const [treeVersion, setTreeVersion] = useState(0)
  const [rootFiles, setRootFiles] = useState<TreeEntry[]>([])

  // Layout & resizing state
  const [showFileTree, setShowFileTree] = useState(() => loadPref('typstify_show_filetree') !== 'false')
  const [fileTreeWidth, setFileTreeWidth] = useState(() => {
    const saved = parseInt(loadPref('typstify_filetree_width') ?? '', 10)
    return Number.isFinite(saved) ? clamp(saved, FILETREE_MIN, FILETREE_MAX) : FILETREE_DEFAULT
  })
  const [showPreview, setShowPreview] = useState(() => loadPref('typstify_show_preview') !== 'false')
  const [editorRatio, setEditorRatio] = useState(() => {
    const saved = parseFloat(loadPref('typstify_editor_ratio') ?? '')
    return Number.isFinite(saved) ? clamp(saved, RATIO_MIN, RATIO_MAX) : 0.5
  })
  const [sidePanelWidth, setSidePanelWidth] = useState(() => {
    const saved = parseInt(loadPref('typstify_sidepanel_width') ?? '', 10)
    return Number.isFinite(saved) ? clamp(saved, SIDEPANEL_MIN, SIDEPANEL_MAX) : SIDEPANEL_DEFAULT
  })
  const [resizing, setResizing] = useState(false)

  const bodyRef = useRef<HTMLDivElement>(null)
  const middleAreaRef = useRef<HTMLDivElement>(null)
  const editorRef = useRef<EditorHandle>(null)
  const t = useTranslations(I18N_KEYS)
  const [theme, toggleTheme] = useTheme()

  useEffect(() => savePref('typstify_show_filetree', String(showFileTree)), [showFileTree])
  useEffect(() => savePref('typstify_filetree_width', String(fileTreeWidth)), [fileTreeWidth])
  useEffect(() => savePref('typstify_show_preview', String(showPreview)), [showPreview])
  useEffect(() => savePref('typstify_editor_ratio', String(editorRatio)), [editorRatio])
  useEffect(() => savePref('typstify_sidepanel_width', String(sidePanelWidth)), [sidePanelWidth])

  // Root-level .typ files for the welcome screen's "open" shortcuts.
  useEffect(() => {
    api
      .get<TreeEntry[]>('/api/workspace/tree?path=')
      .then((list) => setRootFiles((list ?? []).filter((e) => !e.isDir && e.name.endsWith('.typ'))))
      .catch(() => setRootFiles([]))
  }, [treeVersion])

  // Load the active file. content is reset first so the Editor (keyed by
  // path, and reading initialContent only on mount) can never mount the
  // new path with the previous file's text -- saving that would overwrite
  // the new file. A response for a path that is no longer active is
  // dropped, and a failed load shows an error instead of an empty editor
  // whose save would wipe the file.
  useEffect(() => {
    let cancelled = false
    setDirty(false)
    setCursor(null)
    setContent(null)
    setLoadError(null)
    setSaveError(null)
    if (!activePath) return

    api
      .get<string>(`/api/workspace/file?path=${encodeURIComponent(activePath)}`)
      .then((text) => {
        if (!cancelled) setContent(text)
      })
      .catch((err) => {
        if (!cancelled) setLoadError(err instanceof Error ? err.message : String(err))
      })

    if (activePath.endsWith('.typ')) {
      void api.post('/api/preview/restart', { entryFile: activePath })
    }
    return () => {
      cancelled = true
    }
  }, [activePath, loadAttempt])

  // Warn before the tab closes or reloads with unsaved edits.
  useEffect(() => {
    if (!dirty) return
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault()
      e.returnValue = ''
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [dirty])

  const openFile = (path: string) => {
    if (path === activePath) return
    if (dirty) setPendingNav({ kind: 'open', path })
    else setActivePath(path)
  }

  const closeProject = () => {
    if (dirty) setPendingNav({ kind: 'close' })
    else onCloseProject()
  }

  const continueNav = (nav: NonNullable<typeof pendingNav>) => {
    setPendingNav(null)
    setDirty(false)
    if (nav.kind === 'open') setActivePath(nav.path)
    else onCloseProject()
  }

  const saveThenContinue = async () => {
    if (!pendingNav) return
    setSavingForNav(true)
    const ok = (await editorRef.current?.save()) ?? false
    setSavingForNav(false)
    if (ok) continueNav(pendingNav)
    // on failure the dialog stays open and the status bar shows the error
  }

  const saveActiveFile = async (newContent: string) => {
    if (!activePath) return
    await api.put(`/api/workspace/file?path=${encodeURIComponent(activePath)}`, newContent)
    setLastSaved(new Date())
    setSaveError(null)

    if (activePath.endsWith('.typ')) {
      await api.post('/api/preview/restart', { entryFile: activePath })
      setPreviewVersion((v) => v + 1)
    }
  }

  // Every toolbar/modal snippet calls chessbook functions, so make sure the
  // document imports the library -- otherwise it fails with "unknown
  // variable" as soon as it compiles.
  const handleInsertText = (text: string) => {
    editorRef.current?.insertChessSnippet(text)
  }

  const handleAutoFix = () => {
    editorRef.current?.autoFixImports()
  }

  const createNewDoc = async (fileName: string, templateContent?: string) => {
    setIsNewDocOpen(false)
    let target = fileName
    if (!target.includes('.')) target += '.typ'
    const initialCode = templateContent ?? NEW_DOC_TEMPLATE
    try {
      await api.post('/api/workspace/file', { path: target, isDir: false })
      await api.put(`/api/workspace/file?path=${encodeURIComponent(target)}`, initialCode)
    } catch {
      // already exists (the create is O_EXCL, so the template write is
      // skipped and nothing gets overwritten): just open it
    }
    setTreeVersion((v) => v + 1)
    setActivePath(target)
  }

  const togglePanel = (panel: Exclude<SidePanel, null>) => setSidePanel(sidePanel === panel ? null : panel)

  const projectName = projectPath.split('/').filter(Boolean).pop() ?? projectPath
  const crumbs = activePath ? activePath.split('/') : []

  const panelToggles: Array<{ id: Exclude<SidePanel, null>; icon: IconName; label: string }> = [
    { id: 'agent', icon: 'sparkles', label: t('AI Assistant') },
    { id: 'packages', icon: 'package', label: 'Gói Typst' },
    { id: 'settings', icon: 'sliders', label: t('Settings') },
  ]

  return (
    <div className="workspace">
      {/* Transparent overlay while resizing so the PDF <embed> can't swallow pointer events */}
      {resizing && <div className="resizing-overlay" />}

      <header className="workspace-header">
        <div className="hdr-group hdr-left">
          <BrandMark size={24} />
          <button className="btn-ghost hdr-btn" onClick={closeProject} title="Quay lại danh sách dự án">
            <Icon name="back" />
            <span className="hdr-label">Dự án</span>
          </button>
          <span className="project-name" title={projectPath}>
            {projectName}
          </span>
        </div>

        <div className="hdr-group hdr-center">
          {activePath && (
            <>
              <nav className="breadcrumb" aria-label="File đang mở">
                {crumbs.map((c, i) => (
                  <span key={i} className={i === crumbs.length - 1 ? 'crumb crumb-current' : 'crumb'}>
                    {i > 0 && <Icon name="chevron-right" size={12} className="crumb-sep" />}
                    {c}
                  </span>
                ))}
                {dirty && <span className="dirty-dot" title="Có thay đổi chưa lưu" />}
              </nav>
              <button
                className={`hdr-btn save-btn${dirty ? ' btn-accent' : ' btn-ghost'}`}
                title="Lưu (Ctrl+S)"
                disabled={!dirty}
                onClick={() => editorRef.current?.save()}
              >
                <Icon name={dirty ? 'save' : 'check'} />
                <span className="hdr-label">{dirty ? 'Lưu' : 'Đã lưu'}</span>
              </button>
            </>
          )}
        </div>

        <div className="hdr-group hdr-right">
          {activePath?.endsWith('.typ') && <ExportButton path={activePath} />}

          <button
            className="hdr-btn"
            onClick={() => setIsDropboxOpen(true)}
            title="Đồng bộ Dropbox"
            aria-label="Đồng bộ Dropbox"
          >
            <Icon name="cloud" />
            <span className="hdr-label">Dropbox</span>
          </button>

          <div className="hdr-toggle-group" role="group" aria-label="Bố cục">
            <button
              className={`hdr-btn icon-toggle${showFileTree ? ' active' : ''}`}
              onClick={() => setShowFileTree(!showFileTree)}
              aria-pressed={showFileTree}
              title={showFileTree ? 'Ẩn cây thư mục' : 'Hiện cây thư mục'}
            >
              <Icon name="folder" />
              <span className="hdr-label">Tệp</span>
            </button>
            <button
              className={`hdr-btn icon-toggle${showPreview ? ' active' : ''}`}
              onClick={() => setShowPreview(!showPreview)}
              aria-pressed={showPreview}
              title={showPreview ? 'Ẩn bản xem trước' : 'Hiện bản xem trước'}
            >
              <Icon name="eye" />
              <span className="hdr-label">Xem trước</span>
            </button>
          </div>

          <div className="hdr-toggle-group" role="group" aria-label="Bảng bên phải">
            {panelToggles.map((p) => (
              <button
                key={p.id}
                className={`hdr-btn icon-toggle${sidePanel === p.id ? ' active' : ''}`}
                onClick={() => togglePanel(p.id)}
                aria-pressed={sidePanel === p.id}
                title={p.label}
              >
                <Icon name={p.icon} />
                <span className="hdr-label">{p.label}</span>
              </button>
            ))}
          </div>

          <button
            className="btn-ghost hdr-btn theme-toggle"
            onClick={toggleTheme}
            title={theme === 'dark' ? 'Chuyển sang giao diện sáng' : 'Chuyển sang giao diện tối'}
            aria-label={theme === 'dark' ? 'Chuyển sang giao diện sáng' : 'Chuyển sang giao diện tối'}
          >
            <Icon name={theme === 'dark' ? 'sun' : 'moon'} />
          </button>
        </div>
      </header>

      <div className="workspace-body" ref={bodyRef}>
        {showFileTree && (
          <aside className="workspace-filetree" style={{ width: fileTreeWidth }}>
            <FileTree
              key={treeVersion}
              activePath={activePath ?? ''}
              onOpenFile={openFile}
              onPathRemoved={(p) => {
                if (activePath && (activePath === p || activePath.startsWith(`${p}/`))) setActivePath(null)
                setTreeVersion((v) => v + 1)
              }}
              onPathRenamed={(from, to) => {
                if (activePath === from) setActivePath(to)
                else if (activePath?.startsWith(`${from}/`)) setActivePath(to + activePath.slice(from.length))
                setTreeVersion((v) => v + 1)
              }}
            />
          </aside>
        )}

        {showFileTree && (
          <Resizer
            label="Cây thư mục"
            className="resizer-filetree"
            onActiveChange={setResizing}
            onDrag={(x) => {
              const left = bodyRef.current?.getBoundingClientRect().left ?? 0
              setFileTreeWidth(clamp(x - left, FILETREE_MIN, FILETREE_MAX))
            }}
            onStep={(d) => setFileTreeWidth((w) => clamp(w + d * 16, FILETREE_MIN, FILETREE_MAX))}
            onReset={() => setFileTreeWidth(FILETREE_DEFAULT)}
          />
        )}

        <div className="workspace-middle-area" ref={middleAreaRef}>
          <main
            className="workspace-editor"
            style={{
              flex: showPreview ? `0 0 ${editorRatio * 100}%` : '1 1 100%',
              maxWidth: showPreview ? `${editorRatio * 100}%` : '100%',
            }}
          >
            {activePath && content !== null ? (
              <div className="editor-container-with-toolbar">
                {activePath.endsWith('.typ') && (
                  <ChessToolbar
                    onInsertText={handleInsertText}
                    onOpenBoard={openBoard}
                    onOpenPgn={openPgn}
                  />
                )}
                <Suspense fallback={<PanelLoading />}>
                  <Editor
                    key={activePath}
                    ref={editorRef}
                    path={activePath}
                    initialContent={content}
                    onDirtyChange={setDirty}
                    onSave={saveActiveFile}
                    onSaveError={setSaveError}
                    onCursorChange={setCursor}
                    onDiagnosticsChange={setDiagnostics}
                  />
                </Suspense>
              </div>
            ) : activePath ? (
              <div className="editor-state">
                {loadError ? (
                  <div className="editor-state-error" role="alert">
                    <Icon name="error" size={20} />
                    <p>
                      Không mở được <b>{activePath}</b>: {loadError}
                    </p>
                    <button className="btn-primary" onClick={() => setLoadAttempt((n) => n + 1)}>
                      <Icon name="refresh" size={14} /> Thử lại
                    </button>
                  </div>
                ) : (
                  <>
                    <span className="spinner spinner-lg" /> Đang mở {activePath}…
                  </>
                )}
              </div>
            ) : (
              <div className="no-file-open-welcome">
                <div className="welcome-chess-card">
                  <BrandMark size={44} />
                  <h3>Dương Sinh Chess Studio</h3>
                  <p>Soạn sách, giáo trình và bài tập cờ vua bằng Typst. Chọn một tài liệu để bắt đầu:</p>
                  {rootFiles.length > 0 && (
                    <ul className="welcome-file-list">
                      {rootFiles.slice(0, 6).map((f) => (
                        <li key={f.path}>
                          <button className="welcome-file" onClick={() => openFile(f.path)}>
                            <Icon name="file-text" />
                            {f.name}
                          </button>
                        </li>
                      ))}
                    </ul>
                  )}
                  <div className="welcome-actions">
                    <button className="welcome-btn btn-primary" onClick={() => setIsNewDocOpen(true)}>
                      <Icon name="file-plus" /> Tạo tài liệu mới
                    </button>
                    <button className="welcome-btn" onClick={openBoard}>
                      <Icon name="board" /> Xếp bàn cờ
                    </button>
                    <button className="welcome-btn" onClick={openPgn}>
                      <Icon name="scroll" /> Nhập PGN
                    </button>
                  </div>
                </div>
              </div>
            )}
          </main>

          {showPreview && (
            <Resizer
              label="Tỷ lệ soạn thảo / xem trước"
              className="resizer-editor"
              onActiveChange={setResizing}
              onDrag={(x) => {
                const rect = middleAreaRef.current?.getBoundingClientRect()
                if (rect) setEditorRatio(clamp((x - rect.left) / rect.width, RATIO_MIN, RATIO_MAX))
              }}
              onStep={(d) => setEditorRatio((r) => clamp(r + d * 0.02, RATIO_MIN, RATIO_MAX))}
              onReset={() => setEditorRatio(0.5)}
            />
          )}

          {showPreview && (
            <section
              className="workspace-preview"
              style={{
                flex: `0 0 ${(1 - editorRatio) * 100}%`,
                maxWidth: `${(1 - editorRatio) * 100}%`,
              }}
            >
              <PreviewPane path={activePath} version={previewVersion} />
            </section>
          )}
        </div>

        {sidePanel && (
          <Resizer
            label="Bảng bên phải"
            className="resizer-sidepanel"
            onActiveChange={setResizing}
            onDrag={(x) => {
              const right = bodyRef.current?.getBoundingClientRect().right ?? window.innerWidth
              setSidePanelWidth(clamp(right - x, SIDEPANEL_MIN, SIDEPANEL_MAX))
            }}
            onStep={(d) => setSidePanelWidth((w) => clamp(w - d * 16, SIDEPANEL_MIN, SIDEPANEL_MAX))}
            onReset={() => setSidePanelWidth(SIDEPANEL_DEFAULT)}
          />
        )}

        {sidePanel && (
          <aside className="workspace-side-panel" style={{ width: sidePanelWidth }}>
            <Suspense fallback={<PanelLoading />}>
              {sidePanel === 'agent' && <AgentChat projectPath={projectPath} />}
              {sidePanel === 'packages' && <PackageManager />}
              {sidePanel === 'settings' && <SettingsPanel />}
            </Suspense>
          </aside>
        )}
      </div>

      <StatusBar
        activePath={activePath}
        cursor={cursor}
        diagnostics={diagnostics}
        dirty={dirty}
        saveError={saveError}
        lastSaved={lastSaved}
        onAutoFix={handleAutoFix}
      />

      <Suspense fallback={null}>
        {boardUsed && (
          <ChessBoardModal isOpen={isBoardOpen} onClose={() => setIsBoardOpen(false)} onInsertCode={handleInsertText} />
        )}
        {pgnUsed && (
          <PgnImportModal isOpen={isPgnOpen} onClose={() => setIsPgnOpen(false)} onInsertCode={handleInsertText} />
        )}
        {isDropboxOpen && (
          <DropboxSyncModal
            isOpen={isDropboxOpen}
            onClose={() => setIsDropboxOpen(false)}
            onSyncCompleted={() => setTreeVersion((v) => v + 1)}
          />
        )}
      </Suspense>
      {pendingNav && (
        <Modal
          title="Có thay đổi chưa lưu"
          className="prompt-dialog"
          onClose={() => setPendingNav(null)}
          footer={
            <>
              <button className="btn-ghost" onClick={() => setPendingNav(null)} disabled={savingForNav}>
                Hủy
              </button>
              <button className="btn-danger" onClick={() => continueNav(pendingNav)} disabled={savingForNav}>
                Bỏ thay đổi
              </button>
              <button className="btn-primary" onClick={saveThenContinue} disabled={savingForNav}>
                {savingForNav ? 'Đang lưu…' : 'Lưu rồi tiếp tục'}
              </button>
            </>
          }
        >
          <p className="prompt-dialog-body">
            <b>{activePath}</b> có thay đổi chưa lưu.{' '}
            {pendingNav.kind === 'open' ? `Lưu trước khi mở ${pendingNav.path}?` : 'Lưu trước khi đóng dự án?'}
            {saveError && <span className="error"> Lưu thất bại: {saveError}</span>}
          </p>
        </Modal>
      )}
      {isNewDocOpen && (
        <NewDocModal
          isOpen={isNewDocOpen}
          onClose={() => setIsNewDocOpen(false)}
          onCreate={createNewDoc}
        />
      )}
    </div>
  )
}
