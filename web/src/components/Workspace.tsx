import { useEffect, useRef, useState, useCallback } from 'react'
import { api } from '../api/client'
import { useTranslations } from '../lib/i18n'
import { AgentChat } from './AgentChat'
import { Editor, type EditorHandle } from './Editor'
import { ExportButton } from './ExportButton'
import { FileTree } from './FileTree'
import { PackageManager } from './PackageManager'
import { PreviewPane } from './PreviewPane'
import { SettingsPanel } from './SettingsPanel'
import { ChessToolbar } from './ChessToolbar'
import { ChessBoardModal } from './ChessBoardModal'
import { PgnImportModal } from './PgnImportModal'
import { BrandMark } from './BrandMark'
import { useTheme } from '../lib/theme'

const I18N_KEYS = ['AI Assistant', 'Settings', 'Export']

type SidePanel = 'agent' | 'packages' | 'settings' | null

export function Workspace({ projectPath, onCloseProject }: { projectPath: string; onCloseProject: () => void }) {
  const [activePath, setActivePath] = useState<string | null>(null)
  const [content, setContent] = useState<string | null>(null)
  const [dirty, setDirty] = useState(false)
  const [sidePanel, setSidePanel] = useState<SidePanel>('agent')
  const [previewVersion, setPreviewVersion] = useState(0)
  const [isBoardOpen, setIsBoardOpen] = useState(false)
  const [isPgnOpen, setIsPgnOpen] = useState(false)

  // Layout & Resizing States
  const [showFileTree, setShowFileTree] = useState<boolean>(() => {
    const saved = localStorage.getItem('typstify_show_filetree')
    return saved !== null ? saved === 'true' : true
  })
  const [fileTreeWidth, setFileTreeWidth] = useState<number>(() => {
    const saved = localStorage.getItem('typstify_filetree_width')
    return saved ? Math.max(160, Math.min(500, parseInt(saved, 10))) : 230
  })

  const [showPreview, setShowPreview] = useState<boolean>(() => {
    const saved = localStorage.getItem('typstify_show_preview')
    return saved !== null ? saved === 'true' : true
  })
  const [editorRatio, setEditorRatio] = useState<number>(() => {
    const saved = localStorage.getItem('typstify_editor_ratio')
    return saved ? Math.max(0.15, Math.min(0.85, parseFloat(saved))) : 0.5
  })

  const [sidePanelWidth, setSidePanelWidth] = useState<number>(() => {
    const saved = localStorage.getItem('typstify_sidepanel_width')
    return saved ? Math.max(260, Math.min(650, parseInt(saved, 10))) : 340
  })

  const [resizing, setResizing] = useState<'filetree' | 'editor' | 'sidepanel' | null>(null)

  const bodyRef = useRef<HTMLDivElement>(null)
  const middleAreaRef = useRef<HTMLDivElement>(null)
  const editorRef = useRef<EditorHandle>(null)
  const t = useTranslations(I18N_KEYS)
  const [theme, toggleTheme] = useTheme()

  // Persist layout settings
  useEffect(() => {
    localStorage.setItem('typstify_show_filetree', String(showFileTree))
  }, [showFileTree])

  useEffect(() => {
    localStorage.setItem('typstify_filetree_width', String(fileTreeWidth))
  }, [fileTreeWidth])

  useEffect(() => {
    localStorage.setItem('typstify_show_preview', String(showPreview))
  }, [showPreview])

  useEffect(() => {
    localStorage.setItem('typstify_editor_ratio', String(editorRatio))
  }, [editorRatio])

  useEffect(() => {
    localStorage.setItem('typstify_sidepanel_width', String(sidePanelWidth))
  }, [sidePanelWidth])

  // Mouse move and up handlers for resizing
  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!resizing) return

      if (resizing === 'filetree' && bodyRef.current) {
        const bodyRect = bodyRef.current.getBoundingClientRect()
        const newWidth = Math.max(160, Math.min(500, e.clientX - bodyRect.left))
        setFileTreeWidth(newWidth)
      } else if (resizing === 'editor' && middleAreaRef.current) {
        const middleRect = middleAreaRef.current.getBoundingClientRect()
        const relativeX = e.clientX - middleRect.left
        const ratio = Math.max(0.15, Math.min(0.85, relativeX / middleRect.width))
        setEditorRatio(ratio)
      } else if (resizing === 'sidepanel' && bodyRef.current) {
        const bodyRect = bodyRef.current.getBoundingClientRect()
        const newWidth = Math.max(260, Math.min(650, bodyRect.right - e.clientX))
        setSidePanelWidth(newWidth)
      }
    },
    [resizing]
  )

  const handleMouseUp = useCallback(() => {
    if (resizing) {
      setResizing(null)
    }
  }, [resizing])

  useEffect(() => {
    if (resizing) {
      window.addEventListener('mousemove', handleMouseMove)
      window.addEventListener('mouseup', handleMouseUp)
      return () => {
        window.removeEventListener('mousemove', handleMouseMove)
        window.removeEventListener('mouseup', handleMouseUp)
      }
    }
  }, [resizing, handleMouseMove, handleMouseUp])

  useEffect(() => {
    setDirty(false)
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

    if (activePath.endsWith('.typ')) {
      await api.post('/api/preview/restart', { entryFile: activePath })
      setPreviewVersion((v) => v + 1)
    }
  }

  const handleInsertText = (text: string) => {
    editorRef.current?.insertText(text)
  }

  const handleCreateNewDoc = async () => {
    const fileName = window.prompt('Nhập tên file Typst mới:', 'chess_document.typ')
    if (!fileName || !fileName.trim()) return
    let target = fileName.trim()
    if (!target.includes('.')) target += '.typ'
    const initialCode = `// Tài Liệu Cờ Vua Mới\n#set text(font: ("Arial", "Segoe UI Symbol"), size: 9.5pt, lang: "vi")\n\n= Tiêu Đề Tài Liệu\n\n`
    try {
      await api.post('/api/workspace/file/create', { path: target, isDir: false })
      await api.put(`/api/workspace/file?path=${encodeURIComponent(target)}`, initialCode)
    } catch {
      // open anyway if exists
    }
    setActivePath(target)
  }

  return (
    <div className="workspace">
      {/* Invisible overlay while resizing to prevent iframes/embeds from swallowing mouse events */}
      {resizing && <div className="resizing-overlay" />}

      <header className="workspace-header">
        <BrandMark size={22} />
        <button onClick={onCloseProject} title="Quay lại danh sách dự án">&larr; Projects</button>
        <span className="project-path">{projectPath}</span>

        {/* View Layout Toggles */}
        <div className="workspace-header-layout-toggles">
          <button
            className={`toggle-panel-btn ${showFileTree ? 'active' : ''}`}
            onClick={() => setShowFileTree(!showFileTree)}
            title={showFileTree ? 'Ẩn Cây Thư Mục' : 'Hiện Cây Thư Mục'}
          >
            📁 Files
          </button>
          <button
            className={`toggle-panel-btn ${showPreview ? 'active' : ''}`}
            onClick={() => setShowPreview(!showPreview)}
            title={showPreview ? 'Ẩn Bản Xem Trước' : 'Hiện Bản Xem Trước'}
          >
            👁️ Preview
          </button>
        </div>

        {activePath && (
          <button
            className="save-btn"
            title="Save (Ctrl+S)"
            disabled={!dirty}
            onClick={() => editorRef.current?.save()}
          >
            {dirty ? 'Save*' : 'Saved'}
          </button>
        )}
        <div className="header-spacer" />
        {activePath?.endsWith('.typ') && <ExportButton path={activePath} />}
        <button
          className={sidePanel === 'agent' ? 'active' : ''}
          onClick={() => setSidePanel(sidePanel === 'agent' ? null : 'agent')}
        >
          {t('AI Assistant')}
        </button>
        <button
          className={sidePanel === 'packages' ? 'active' : ''}
          onClick={() => setSidePanel(sidePanel === 'packages' ? null : 'packages')}
        >
          Packages
        </button>
        <button
          className={sidePanel === 'settings' ? 'active' : ''}
          onClick={() => setSidePanel(sidePanel === 'settings' ? null : 'settings')}
        >
          {t('Settings')}
        </button>
        <button
          className="btn-ghost theme-toggle"
          onClick={toggleTheme}
          title={theme === 'dark' ? 'Chuyển sang giao diện sáng' : 'Chuyển sang giao diện tối'}
          aria-label={theme === 'dark' ? 'Chuyển sang giao diện sáng' : 'Chuyển sang giao diện tối'}
        >
          {theme === 'dark' ? '☀' : '☾'}
        </button>
      </header>

      <div className="workspace-body" ref={bodyRef}>
        {/* Left FileTree Panel */}
        {showFileTree && (
          <aside className="workspace-filetree" style={{ width: fileTreeWidth }}>
            <FileTree activePath={activePath ?? ''} onOpenFile={setActivePath} />
          </aside>
        )}

        {/* Resizer 1: FileTree <-> Middle Area */}
        {showFileTree && (
          <div
            className={`panel-resizer resizer-filetree ${resizing === 'filetree' ? 'active' : ''}`}
            onMouseDown={(e) => {
              e.preventDefault()
              setResizing('filetree')
            }}
            title="Kéo để chỉnh kích thước File Tree (Nhấp đúp về mặc định)"
            onDoubleClick={() => setFileTreeWidth(230)}
          />
        )}

        {/* Middle Area: Editor + Preview */}
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
                />
              </div>
            ) : (
              <div className="no-file-open-welcome">
                <div className="welcome-chess-card">
                  <h3>♟️ Typstify Chess Publishing Studio</h3>
                  <p>Chọn một file <code>.typ</code> ở danh sách bên trái để bắt đầu soạn thảo, hoặc sử dụng các công cụ nhanh dưới đây:</p>
                  <div className="welcome-actions">
                    <button className="welcome-btn primary" onClick={() => setActivePath('main.typ')}>
                      📄 Mở main.typ
                    </button>
                    <button className="welcome-btn primary" onClick={handleCreateNewDoc}>
                      ➕ Tạo File Mới
                    </button>
                    <button className="welcome-btn" onClick={() => setIsBoardOpen(true)}>
                      ♟️ Xếp Bàn Cờ
                    </button>
                    <button className="welcome-btn" onClick={() => setIsPgnOpen(true)}>
                      📜 Nhập PGN
                    </button>
                  </div>
                </div>
              </div>
            )}
          </main>

          {/* Resizer 2: Editor <-> Preview */}
          {showPreview && (
            <div
              className={`panel-resizer resizer-editor ${resizing === 'editor' ? 'active' : ''}`}
              onMouseDown={(e) => {
                e.preventDefault()
                setResizing('editor')
              }}
              title="Kéo để chỉnh tỷ lệ Editor / Preview (Nhấp đúp về 50/50)"
              onDoubleClick={() => setEditorRatio(0.5)}
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

        {/* Resizer 3: Middle Area <-> Side Panel */}
        {sidePanel && (
          <div
            className={`panel-resizer resizer-sidepanel ${resizing === 'sidepanel' ? 'active' : ''}`}
            onMouseDown={(e) => {
              e.preventDefault()
              setResizing('sidepanel')
            }}
            title="Kéo để chỉnh kích thước bảng điều khiển phụ (Nhấp đúp về mặc định)"
            onDoubleClick={() => setSidePanelWidth(340)}
          />
        )}

        {/* Right Side Panel */}
        {sidePanel && (
          <aside className="workspace-side-panel" style={{ width: sidePanelWidth }}>
            {sidePanel === 'agent' && <AgentChat projectPath={projectPath} />}
            {sidePanel === 'packages' && <PackageManager />}
            {sidePanel === 'settings' && <SettingsPanel />}
          </aside>
        )}
      </div>

      {/* Chess Modals */}
      <ChessBoardModal
        isOpen={isBoardOpen}
        onClose={() => setIsBoardOpen(false)}
        onInsertCode={handleInsertText}
      />
      <PgnImportModal
        isOpen={isPgnOpen}
        onClose={() => setIsPgnOpen(false)}
        onInsertCode={handleInsertText}
      />
    </div>
  )
}

