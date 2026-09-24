import { useEffect, useRef, useState } from 'react'
import { api } from '../api/client'
import type { TreeEntry } from '../api/types'
import { useTranslations } from '../lib/i18n'
import { useTheme } from '../lib/theme'
import { AgentChat } from './AgentChat'
import { BrandMark } from './BrandMark'
import { ChessBoardModal } from './ChessBoardModal'
import { ChessToolbar } from './ChessToolbar'
import { Editor, type EditorHandle } from './Editor'
import { ExportButton } from './ExportButton'
import { FileTree } from './FileTree'
import { Icon, type IconName } from './Icon'
import { PackageManager } from './PackageManager'
import { PgnImportModal } from './PgnImportModal'
import { PreviewPane } from './PreviewPane'
import { PromptDialog } from './PromptDialog'
import { Resizer } from './Resizer'
import { SettingsPanel } from './SettingsPanel'
import { StatusBar, type DiagnosticCounts } from './StatusBar'

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

const NEW_DOC_TEMPLATE = `// Tài Liệu Cờ Vua Mới\n#set text(font: ("Arial", "Segoe UI Symbol"), size: 9.5pt, lang: "vi")\n\n= Tiêu Đề Tài Liệu\n\n`

export function Workspace({ projectPath, onCloseProject }: { projectPath: string; onCloseProject: () => void }) {
  const [activePath, setActivePath] = useState<string | null>(null)
  const [content, setContent] = useState<string | null>(null)
  const [dirty, setDirty] = useState(false)
  const [lastSaved, setLastSaved] = useState<Date | null>(null)
  const [cursor, setCursor] = useState<{ line: number; col: number } | null>(null)
  const [diagnostics, setDiagnostics] = useState<DiagnosticCounts>({ errors: 0, warnings: 0 })
  const [sidePanel, setSidePanel] = useState<SidePanel>('agent')
  const [previewVersion, setPreviewVersion] = useState(0)
  const [isBoardOpen, setIsBoardOpen] = useState(false)
  const [isPgnOpen, setIsPgnOpen] = useState(false)
  const [isNewDocOpen, setIsNewDocOpen] = useState(false)
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

  useEffect(() => {
    setDirty(false)
    setCursor(null)
    if (!activePath) {
      setContent(null)
      return
    }
    api
      .get<string>(`/api/workspace/file?path=${encodeURIComponent(activePath)}`)
      .then(setContent)
      .catch(() => setContent(''))

    if (activePath.endsWith('.typ')) {
      void api.post('/api/preview/restart', { entryFile: activePath })
    }
  }, [activePath])

  const saveActiveFile = async (newContent: string) => {
    if (!activePath) return
    await api.put(`/api/workspace/file?path=${encodeURIComponent(activePath)}`, newContent)
    setLastSaved(new Date())

    if (activePath.endsWith('.typ')) {
      await api.post('/api/preview/restart', { entryFile: activePath })
      setPreviewVersion((v) => v + 1)
    }
  }

  const handleInsertText = (text: string) => {
    editorRef.current?.insertText(text)
  }

  const createNewDoc = async (fileName: string) => {
    setIsNewDocOpen(false)
    let target = fileName
    if (!target.includes('.')) target += '.typ'
    try {
      await api.post('/api/workspace/file/create', { path: target, isDir: false })
      await api.put(`/api/workspace/file?path=${encodeURIComponent(target)}`, NEW_DOC_TEMPLATE)
    } catch {
      // already exists: just open it
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
          <button className="btn-ghost hdr-btn" onClick={onCloseProject} title="Quay lại danh sách dự án">
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
            <FileTree key={treeVersion} activePath={activePath ?? ''} onOpenFile={setActivePath} />
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
                    onOpenBoard={() => setIsBoardOpen(true)}
                    onOpenPgn={() => setIsPgnOpen(true)}
                  />
                )}
                <Editor
                  key={activePath}
                  ref={editorRef}
                  path={activePath}
                  initialContent={content}
                  onDirtyChange={setDirty}
                  onSave={saveActiveFile}
                  onCursorChange={setCursor}
                  onDiagnosticsChange={setDiagnostics}
                />
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
                          <button className="welcome-file" onClick={() => setActivePath(f.path)}>
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
                    <button className="welcome-btn" onClick={() => setIsBoardOpen(true)}>
                      <Icon name="board" /> Xếp bàn cờ
                    </button>
                    <button className="welcome-btn" onClick={() => setIsPgnOpen(true)}>
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
              <PreviewPane key={`${activePath}-${previewVersion}`} path={activePath} />
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
            {sidePanel === 'agent' && <AgentChat projectPath={projectPath} />}
            {sidePanel === 'packages' && <PackageManager />}
            {sidePanel === 'settings' && <SettingsPanel />}
          </aside>
        )}
      </div>

      <StatusBar
        activePath={activePath}
        cursor={cursor}
        diagnostics={diagnostics}
        dirty={dirty}
        lastSaved={lastSaved}
      />

      <ChessBoardModal isOpen={isBoardOpen} onClose={() => setIsBoardOpen(false)} onInsertCode={handleInsertText} />
      <PgnImportModal isOpen={isPgnOpen} onClose={() => setIsPgnOpen(false)} onInsertCode={handleInsertText} />
      {isNewDocOpen && (
        <PromptDialog
          title="Tạo tài liệu mới"
          label="Tên file"
          defaultValue="chess_document.typ"
          hint="Tự thêm đuôi .typ nếu bạn không ghi. Có thể dùng thư mục, ví dụ: chapters/chuong-1.typ"
          confirmLabel="Tạo"
          onConfirm={createNewDoc}
          onCancel={() => setIsNewDocOpen(false)}
        />
      )}
    </div>
  )
}
