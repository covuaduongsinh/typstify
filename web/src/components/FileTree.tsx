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

  useEffect(() => {
    api.get<TreeEntry[]>('/api/workspace/tree?path=').then(setRoots).catch(console.error)
  }, [])

  return (
    <div className="file-tree" role="tree">
      {roots.map((r) => (
        <Node key={r.path} entry={r} depth={0} activePath={activePath} onOpenFile={onOpenFile} />
      ))}
    </div>
  )
}
