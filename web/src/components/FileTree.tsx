import { useEffect, useRef, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { TreeEntry } from '../api/types'
import { Icon, type IconName } from './Icon'
import { Modal } from './Modal'
import { PromptDialog } from './PromptDialog'

const IMAGE_EXT = /\.(png|jpe?g|gif|svg|webp)$/i

function iconFor(entry: TreeEntry, expanded: boolean): IconName {
  if (entry.isDir) return expanded ? 'folder-open' : 'folder'
  if (entry.name.endsWith('.typ')) return 'file-text'
  if (entry.name.endsWith('.pgn')) return 'scroll'
  if (IMAGE_EXT.test(entry.name)) return 'image'
  return 'file'
}

function parentOf(path: string): string {
  const i = path.lastIndexOf('/')
  return i < 0 ? '' : path.slice(0, i)
}

function errorText(err: unknown, fallback: string): string {
  return err instanceof ApiError || err instanceof Error ? err.message : fallback
}

type MenuState = { entry: TreeEntry; x: number; y: number }

interface NodeProps {
  entry: TreeEntry
  depth: number
  activePath: string
  onOpenFile: (path: string) => void
  onMenu: (m: MenuState) => void
}

function Node({ entry, depth, activePath, onOpenFile, onMenu }: NodeProps) {
  const [expanded, setExpanded] = useState(false)
  const [children, setChildren] = useState<TreeEntry[] | null>(null)

  const loadChildren = async () => {
    const list = await api.get<TreeEntry[]>(`/api/workspace/tree?path=${encodeURIComponent(entry.path)}`)
    setChildren(list ?? [])
    return list
  }

  // Reveal the active file: expand every ancestor folder of it.
  const containsActive = entry.isDir && activePath.startsWith(`${entry.path}/`)
  useEffect(() => {
    if (!containsActive) return
    let cancelled = false
    void (async () => {
      if (children === null) await loadChildren().catch(() => null)
      if (!cancelled) setExpanded(true)
    })()
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [containsActive])

  const toggle = async () => {
    if (!entry.isDir) {
      onOpenFile(entry.path)
      return
    }
    if (!expanded && children === null) await loadChildren().catch(() => null)
    setExpanded((v) => !v)
  }

  const openMenu = (x: number, y: number) => onMenu({ entry, x, y })
  const isActive = entry.path === activePath

  return (
    <div>
      <div
        className={`file-tree-row${isActive ? ' active' : ''}`}
        style={{ paddingLeft: 6 + depth * 14 }}
        onClick={toggle}
        onContextMenu={(e) => {
          e.preventDefault()
          openMenu(e.clientX, e.clientY)
        }}
        role="treeitem"
        aria-expanded={entry.isDir ? expanded : undefined}
        aria-selected={isActive}
        tabIndex={0}
        title={entry.path}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            toggle()
          } else if (e.key === 'F2' || (e.shiftKey && e.key === 'F10') || e.key === 'ContextMenu') {
            e.preventDefault()
            const r = e.currentTarget.getBoundingClientRect()
            openMenu(r.left + 24, r.bottom)
          }
        }}
      >
        <span className="file-tree-caret">
          {entry.isDir && <Icon name={expanded ? 'chevron-down' : 'chevron-right'} size={12} />}
        </span>
        <Icon name={iconFor(entry, expanded)} size={15} className={`file-icon file-icon-${iconFor(entry, false)}`} />
        <span className="file-tree-name">{entry.name}</span>
        <button
          className="file-tree-more"
          aria-label={`Tùy chọn cho ${entry.name}`}
          tabIndex={-1}
          onClick={(e) => {
            e.stopPropagation()
            const r = e.currentTarget.getBoundingClientRect()
            openMenu(r.left, r.bottom)
          }}
        >
          ⋯
        </button>
      </div>
      {expanded && children && (
        <div role="group">
          {children.map((c) => (
            <Node
              key={c.path}
              entry={c}
              depth={depth + 1}
              activePath={activePath}
              onOpenFile={onOpenFile}
              onMenu={onMenu}
            />
          ))}
        </div>
      )}
    </div>
  )
}

type Dialog =
  | { kind: 'create'; isDir: boolean; parent: string }
  | { kind: 'rename'; entry: TreeEntry }
  | { kind: 'delete'; entry: TreeEntry }

export function FileTree({
  activePath,
  onOpenFile,
  onPathRemoved,
  onPathRenamed,
}: {
  activePath: string
  onOpenFile: (path: string) => void
  /** Called after a file/folder is deleted, so an open editor on it can close. */
  onPathRemoved?: (path: string) => void
  /** Called after a rename, so an open editor can follow the file. */
  onPathRenamed?: (from: string, to: string) => void
}) {
  const [roots, setRoots] = useState<TreeEntry[]>([])
  const [version, setVersion] = useState(0)
  const [menu, setMenu] = useState<MenuState | null>(null)
  const [dialog, setDialog] = useState<Dialog | null>(null)
  const [errorMsg, setErrorMsg] = useState<string | null>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    api
      .get<TreeEntry[]>('/api/workspace/tree?path=')
      .then((list) => setRoots(list ?? []))
      .catch((err) => setErrorMsg(errorText(err, 'Không tải được danh sách tệp')))
  }, [version])

  const refresh = () => setVersion((v) => v + 1)

  // Close the context menu on outside click / Esc, and focus its first item.
  useEffect(() => {
    if (!menu) return
    menuRef.current?.querySelector<HTMLElement>('button')?.focus()
    const onDown = (e: MouseEvent) => {
      if (!menuRef.current?.contains(e.target as Node)) setMenu(null)
    }
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setMenu(null)
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [menu])

  const run = async (action: () => Promise<void>, fallback: string) => {
    setDialog(null)
    setErrorMsg(null)
    try {
      await action()
      refresh()
    } catch (err) {
      setErrorMsg(errorText(err, fallback))
    }
  }

  const create = (name: string, isDir: boolean, parent: string) =>
    run(async () => {
      let target = parent ? `${parent}/${name}` : name
      if (!isDir && !name.includes('.')) target += '.typ'
      await api.post('/api/workspace/file', { path: target, isDir })
      if (!isDir) onOpenFile(target)
    }, 'Không tạo được tệp/thư mục')

  const rename = (entry: TreeEntry, newName: string) =>
    run(async () => {
      const parent = parentOf(entry.path)
      const to = parent ? `${parent}/${newName}` : newName
      if (to === entry.path) return
      await api.post('/api/workspace/rename', { from: entry.path, to })
      onPathRenamed?.(entry.path, to)
    }, 'Không đổi tên được')

  const remove = (entry: TreeEntry) =>
    run(async () => {
      await api.del(`/api/workspace/file?path=${encodeURIComponent(entry.path)}`)
      onPathRemoved?.(entry.path)
    }, 'Không xóa được')

  const menuItem = (label: string, icon: IconName, onClick: () => void, danger = false) => (
    <button
      className={`context-menu-item${danger ? ' danger' : ''}`}
      role="menuitem"
      onClick={() => {
        setMenu(null)
        onClick()
      }}
    >
      <Icon name={icon} size={14} />
      {label}
    </button>
  )

  return (
    <div className="file-tree-container">
      <div className="file-tree-header">
        <span className="file-tree-title">Tệp dự án</span>
        <div className="file-tree-actions">
          <button
            className="tree-action-btn btn-ghost"
            title="Tạo file .typ mới"
            aria-label="Tạo file .typ mới"
            onClick={() => setDialog({ kind: 'create', isDir: false, parent: '' })}
          >
            <Icon name="file-plus" size={15} />
          </button>
          <button
            className="tree-action-btn btn-ghost"
            title="Tạo thư mục mới"
            aria-label="Tạo thư mục mới"
            onClick={() => setDialog({ kind: 'create', isDir: true, parent: '' })}
          >
            <Icon name="folder-plus" size={15} />
          </button>
          <button className="tree-action-btn btn-ghost" title="Làm mới" aria-label="Làm mới" onClick={refresh}>
            <Icon name="refresh" size={15} />
          </button>
        </div>
      </div>

      {errorMsg && (
        <div className="tree-error-msg" role="alert">
          {errorMsg}
        </div>
      )}

      <div className="file-tree" role="tree" aria-label="Tệp dự án" key={version}>
        {roots.map((r) => (
          <Node key={r.path} entry={r} depth={0} activePath={activePath} onOpenFile={onOpenFile} onMenu={setMenu} />
        ))}
        {roots.length === 0 && !errorMsg && <div className="file-tree-empty">Dự án chưa có tệp nào.</div>}
      </div>

      {menu && (
        <div
          ref={menuRef}
          className="context-menu"
          role="menu"
          style={{ left: Math.min(menu.x, window.innerWidth - 200), top: Math.min(menu.y, window.innerHeight - 180) }}
        >
          {!menu.entry.isDir && menuItem('Mở', 'file-text', () => onOpenFile(menu.entry.path))}
          {menu.entry.isDir &&
            menuItem('Tạo file trong thư mục', 'file-plus', () =>
              setDialog({ kind: 'create', isDir: false, parent: menu.entry.path }),
            )}
          {menu.entry.isDir &&
            menuItem('Tạo thư mục con', 'folder-plus', () =>
              setDialog({ kind: 'create', isDir: true, parent: menu.entry.path }),
            )}
          {menuItem('Đổi tên', 'template', () => setDialog({ kind: 'rename', entry: menu.entry }))}
          {menuItem('Xóa', 'x', () => setDialog({ kind: 'delete', entry: menu.entry }), true)}
        </div>
      )}

      {dialog?.kind === 'create' && (
        <PromptDialog
          title={dialog.isDir ? 'Tạo thư mục mới' : 'Tạo file mới'}
          label={dialog.parent ? `Tên (trong ${dialog.parent}/)` : 'Tên'}
          defaultValue={dialog.isDir ? 'chapters' : 'chuong-moi.typ'}
          hint={dialog.isDir ? undefined : 'Tự thêm đuôi .typ nếu bạn không ghi.'}
          confirmLabel="Tạo"
          onConfirm={(name) => create(name, dialog.isDir, dialog.parent)}
          onCancel={() => setDialog(null)}
        />
      )}
      {dialog?.kind === 'rename' && (
        <PromptDialog
          title="Đổi tên"
          label="Tên mới"
          defaultValue={dialog.entry.name}
          confirmLabel="Đổi tên"
          onConfirm={(name) => rename(dialog.entry, name)}
          onCancel={() => setDialog(null)}
        />
      )}
      {dialog?.kind === 'delete' && (
        <Modal
          title="Xóa vĩnh viễn?"
          className="prompt-dialog"
          onClose={() => setDialog(null)}
          footer={
            <>
              <button className="btn-ghost" onClick={() => setDialog(null)}>
                Hủy
              </button>
              <button className="btn-danger" onClick={() => remove(dialog.entry)}>
                Xóa
              </button>
            </>
          }
        >
          <p className="prompt-dialog-body">
            {dialog.entry.isDir ? 'Thư mục' : 'Tệp'} <b>{dialog.entry.path}</b>
            {dialog.entry.isDir ? ' và toàn bộ nội dung bên trong' : ''} sẽ bị xóa khỏi máy chủ. Không thể hoàn tác.
          </p>
        </Modal>
      )}
    </div>
  )
}
