import { useEffect, useRef, useState } from 'react'
import { api } from '../api/client'
import { useTranslations } from '../lib/i18n'
import { AgentChat } from './AgentChat'
import { Editor, type EditorHandle } from './Editor'
import { ExportButton } from './ExportButton'
import { FileTree } from './FileTree'
import { PackageManager } from './PackageManager'
import { PreviewPane } from './PreviewPane'
import { SettingsPanel } from './SettingsPanel'

// Keys matching i18n/translations catalog entries verbatim, so they reuse
// the desktop app's existing zh-CN/de translations (see server/i18n_api.go
// and docs/plans/plan_web_version.md Giai doan 5). "Packages" has no
// desktop equivalent and stays English-only for now.
const I18N_KEYS = ['AI Assistant', 'Settings', 'Export']

type SidePanel = 'agent' | 'packages' | 'settings' | null

export function Workspace({ projectPath, onCloseProject }: { projectPath: string; onCloseProject: () => void }) {
  const [activePath, setActivePath] = useState<string | null>(null)
  const [content, setContent] = useState<string | null>(null)
  const [dirty, setDirty] = useState(false)
  const [sidePanel, setSidePanel] = useState<SidePanel>('agent')
  const [previewVersion, setPreviewVersion] = useState(0)
  const editorRef = useRef<EditorHandle>(null)
  const t = useTranslations(I18N_KEYS)

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

    // Pin this file as the preview's compile entry, matching the desktop
    // app's behaviour (ui/editors/typst_view.go calls RestartPreviewWithEntry
    // whenever a Typst file's editor view opens) -- tinymist's default
    // preview does not automatically follow which file the LSP client is
    // editing, it needs to be told explicitly.
    if (activePath.endsWith('.typ')) {
      void api.post('/api/preview/restart', { entryFile: activePath })
    }
  }, [activePath])

  const saveActiveFile = async (newContent: string) => {
    if (!activePath) return
    await api.put(`/api/workspace/file?path=${encodeURIComponent(activePath)}`, newContent)

    // The preview reacts to disk content on (re)start, not to live LSP
    // didChange edits (see risk #2 follow-up in docs/plans/plan_web_version.md)
    // -- for v1, "save" is what refreshes it, similar to a classic
    // compile-on-save workflow.
    if (activePath.endsWith('.typ')) {
      await api.post('/api/preview/restart', { entryFile: activePath })
      setPreviewVersion((v) => v + 1)
    }
  }

  return (
    <div className="workspace">
      <header className="workspace-header">
        <button onClick={onCloseProject}>&larr; Projects</button>
        <span className="project-path">{projectPath}</span>
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
      </header>

      <div className="workspace-body">
        <aside className="workspace-filetree">
          <FileTree activePath={activePath ?? ''} onOpenFile={setActivePath} />
        </aside>

        <main className="workspace-editor">
          {activePath && content !== null ? (
            <Editor
              key={activePath}
              ref={editorRef}
              path={activePath}
              initialContent={content}
              onDirtyChange={setDirty}
              onSave={saveActiveFile}
            />
          ) : (
            <div className="no-file-open">Select a file to start editing</div>
          )}
        </main>

        <section className="workspace-preview">
          <PreviewPane key={`${activePath}-${previewVersion}`} />
        </section>

        {sidePanel && (
          <aside className="workspace-side-panel">
            {sidePanel === 'agent' && <AgentChat projectPath={projectPath} />}
            {sidePanel === 'packages' && <PackageManager />}
            {sidePanel === 'settings' && <SettingsPanel />}
          </aside>
        )}
      </div>
    </div>
  )
}
