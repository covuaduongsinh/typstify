import { useEffect, useState } from 'react'
import { api } from '../api/client'
import type { TreeEntry } from '../api/types'

interface NodeProps {
  entry: TreeEntry
  depth: number
  activePath: string
  onOpenFile: (path: string) => void
}

function Node({ entry, depth, activePath, onOpenFile }: NodeProps) {
  const [expanded, setExpanded] = useState(false)
  const [children, setChildren] = useState<TreeEntry[] | null>(null)

  const toggle = async () => {
    if (!entry.isDir) {
      onOpenFile(entry.path)
      return
    }
    if (!expanded && children === null) {
      const list = await api.get<TreeEntry[]>(`/api/workspace/tree?path=${encodeURIComponent(entry.path)}`)
      setChildren(list)
    }
    setExpanded((v) => !v)
  }

  return (
    <div>
      <div
        className={`file-tree-row${entry.path === activePath ? ' active' : ''}`}
        style={{ paddingLeft: depth * 14 }}
        onClick={toggle}
        role="treeitem"
        aria-expanded={entry.isDir ? expanded : undefined}
        aria-selected={entry.path === activePath}
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            toggle()
          }
        }}
      >
        {entry.isDir ? (expanded ? '▾' : '▸') : '·'} {entry.name}
      </div>
      {expanded && children && (
        <div>
          {children.map((c) => (
            <Node key={c.path} entry={c} depth={depth + 1} activePath={activePath} onOpenFile={onOpenFile} />
          ))}
        </div>
      )}
    </div>
  )
}

export function FileTree({ activePath, onOpenFile }: { activePath: string; onOpenFile: (path: string) => void }) {
  const [roots, setRoots] = useState<TreeEntry[]>([])
  const [isCreating, setIsCreating] = useState<'file' | 'folder' | null>(null)
  const [newItemName, setNewItemName] = useState('')
  const [errorMsg, setErrorMsg] = useState<string | null>(null)

  const refreshTree = () => {
    api.get<TreeEntry[]>('/api/workspace/tree?path=').then(setRoots).catch(console.error)
  }

  useEffect(() => {
    refreshTree()
  }, [])

  const handleCreateSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newItemName.trim()) {
      setIsCreating(null)
      return
    }

    let targetName = newItemName.trim()
    if (isCreating === 'file' && !targetName.includes('.')) {
      targetName += '.typ'
    }

    try {
      setErrorMsg(null)
      await api.post('/api/workspace/file/create', {
        path: targetName,
        isDir: isCreating === 'folder',
      })
      setIsCreating(null)
      setNewItemName('')
      refreshTree()
      if (isCreating === 'file') {
        onOpenFile(targetName)
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Lỗi khi tạo file/thư mục')
    }
  }

  return (
    <div className="file-tree-container">
      {/* File Tree Action Header */}
      <div className="file-tree-header">
        <span className="file-tree-title">📁 TẬP TIN</span>
        <div className="file-tree-actions">
          <button
            className="tree-action-btn"
            title="Tạo File .typ Mới"
            onClick={() => {
              setIsCreating('file')
              setNewItemName('chess_doc.typ')
              setErrorMsg(null)
            }}
          >
            📄+
          </button>
          <button
            className="tree-action-btn"
            title="Tạo Thư Mục Mới"
            onClick={() => {
              setIsCreating('folder')
              setNewItemName('chapters')
              setErrorMsg(null)
            }}
          >
            📁+
          </button>
          <button
            className="tree-action-btn"
            title="Làm mới danh sách"
            onClick={refreshTree}
          >
            🔄
          </button>
        </div>
      </div>

      {/* Inline Create Form */}
      {isCreating && (
        <form className="tree-create-form" onSubmit={handleCreateSubmit}>
          <input
            autoFocus
            type="text"
            className="tree-create-input"
            value={newItemName}
            placeholder={isCreating === 'file' ? 'ten_file.typ' : 'ten_thu_muc'}
            onChange={(e) => setNewItemName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Escape') setIsCreating(null)
            }}
          />
          <div className="tree-create-actions">
            <button type="submit" className="tree-confirm-btn">Tạo</button>
            <button type="button" className="tree-cancel-btn" onClick={() => setIsCreating(null)}>✕</button>
          </div>
        </form>
      )}

      {errorMsg && <div className="tree-error-msg">{errorMsg}</div>}

      <div className="file-tree" role="tree">
        {roots.map((r) => (
          <Node key={r.path} entry={r} depth={0} activePath={activePath} onOpenFile={onOpenFile} />
        ))}
      </div>
    </div>
  )
}
