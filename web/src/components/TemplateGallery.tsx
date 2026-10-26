import { useEffect, useRef, useState } from 'react'
import { ApiError, api } from '../api/client'
import { PACKAGE_CATEGORIES } from '../api/pkgTypes'
import type { ProjectInfo } from '../api/types'
import { STARTER_TEMPLATES, type TemplateOption } from '../lib/chessTemplates'
import { universeThumbnailUrl, type UniversePackage, type UniversePage } from '../lib/universe'
import { Modal } from './Modal'

const PAGE_SIZE = 24

type Choice = { kind: 'chess'; template: TemplateOption } | { kind: 'universe'; pkg: UniversePackage }

/** "my-thesis-template" -> "My thesis template", a default project name. */
const prettyName = (name: string) => {
  const s = name.replace(/[-_]+/g, ' ').trim()
  return s.charAt(0).toUpperCase() + s.slice(1)
}

/**
 * TemplateGallery creates a project from a template: the built-in chess
 * templates or a Typst Universe template (copied like `typst init`).
 * onCreated gets the new project's path and the file to open first.
 */
export function TemplateGallery({
  onClose,
  onCreated,
}: {
  onClose: () => void
  onCreated: (path: string, main: string) => void
}) {
  const [tab, setTab] = useState<'chess' | 'universe'>('chess')
  const [choice, setChoice] = useState<Choice | null>(null)
  const [name, setName] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Universe search.
  const [query, setQuery] = useState('')
  const [category, setCategory] = useState('')
  const [results, setResults] = useState<UniversePackage[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const requestRef = useRef(0)
  // A project already created when opening it failed: retry only the open.
  const createdRef = useRef<{ id: string; main: string } | null>(null)

  const load = (offset: number) => {
    const id = ++requestRef.current
    setLoading(true)
    setError(null)
    const params = new URLSearchParams({ kind: 'template', q: query, category, offset: String(offset), limit: String(PAGE_SIZE) })
    api
      .get<UniversePage>(`/api/universe/packages?${params}`)
      .then((page) => {
        if (id !== requestRef.current) return
        // The index can be refreshed between pages: skip repeats.
        setResults((prev) => {
          if (offset === 0) return page.packages
          const seen = new Set(prev.map((p) => p.name))
          return [...prev, ...page.packages.filter((p) => !seen.has(p.name))]
        })
        setTotal(page.total)
      })
      .catch((err: unknown) => {
        if (id === requestRef.current) setError(err instanceof ApiError ? err.message : 'Không tải được danh sách mẫu')
      })
      .finally(() => {
        if (id === requestRef.current) setLoading(false)
      })
  }

  useEffect(() => {
    if (tab !== 'universe') return
    // New filters: drop the old results (and any request for them) now, so
    // "show more" never pages through the previous query.
    requestRef.current++
    setResults([])
    setTotal(0)
    const timer = window.setTimeout(() => load(0), 300)
    return () => window.clearTimeout(timer)
    // load reads query/category; re-run only when they (or the tab) change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab, query, category])

  const choose = (c: Choice) => {
    createdRef.current = null
    setChoice(c)
    setName(c.kind === 'chess' ? c.template.name : prettyName(c.pkg.name))
    setError(null)
  }

  const create = async () => {
    if (!choice || !name.trim()) return
    setBusy(true)
    setError(null)
    try {
      if (!createdRef.current) {
        const body =
          choice.kind === 'chess'
            ? { name: name.trim(), mainContent: choice.template.content }
            : { name: name.trim(), template: { name: choice.pkg.name, version: choice.pkg.version } }
        const p = await api.post<ProjectInfo & { main: string }>('/api/projects', body)
        createdRef.current = { id: p.id, main: p.main }
      }
      const { id, main } = createdRef.current
      const opened = await api.post<{ path: string }>(`/api/projects/${id}/open`)
      onCreated(opened.path, main)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Không tạo được dự án')
      setBusy(false)
    }
  }

  return (
    <Modal
      title="Dự án mới từ mẫu"
      onClose={onClose}
      className="template-gallery-modal"
      footer={
        <form
          className="template-create"
          onSubmit={(e) => {
            e.preventDefault()
            void create()
          }}
        >
          <span className="template-chosen">
            {choice ? (
              <>
                Mẫu: <strong>{choice.kind === 'chess' ? choice.template.name : `${choice.pkg.name} ${choice.pkg.version}`}</strong>
              </>
            ) : (
              'Chọn một mẫu'
            )}
          </span>
          <input
            aria-label="Tên dự án"
            placeholder="Tên dự án"
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={!choice}
          />
          <button type="submit" className="btn-primary" disabled={!choice || !name.trim() || busy}>
            {busy ? 'Đang tạo…' : 'Tạo dự án'}
          </button>
        </form>
      }
    >
      <div className="history-tabs template-tabs" role="tablist">
        <button role="tab" aria-selected={tab === 'chess'} className={tab === 'chess' ? 'active' : ''} onClick={() => setTab('chess')}>
          Cờ vua
        </button>
        <button
          role="tab"
          aria-selected={tab === 'universe'}
          className={tab === 'universe' ? 'active' : ''}
          onClick={() => setTab('universe')}
        >
          Typst Universe
        </button>
      </div>

      {error && (
        <div className="error" role="alert">
          {error}
        </div>
      )}

      {tab === 'chess' ? (
        <div className="template-grid">
          {STARTER_TEMPLATES.map((t) => (
            <button
              key={t.id}
              className={`template-card${choice?.kind === 'chess' && choice.template.id === t.id ? ' selected' : ''}`}
              onClick={() => choose({ kind: 'chess', template: t })}
            >
              <span className="template-name">{t.name}</span>
              <span className="template-desc">{t.desc}</span>
            </button>
          ))}
        </div>
      ) : (
        <>
          <div className="template-filters">
            <input
              className="symbol-search"
              placeholder="Tìm mẫu (thesis, letter, cv…)"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
            <select aria-label="Nhóm" value={category} onChange={(e) => setCategory(e.target.value)}>
              <option value="">Mọi nhóm</option>
              {PACKAGE_CATEGORIES.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
          </div>
          <div className="template-grid">
            {results.map((p) => (
              <button
                key={p.name}
                className={`template-card${choice?.kind === 'universe' && choice.pkg.name === p.name ? ' selected' : ''}`}
                onClick={() => choose({ kind: 'universe', pkg: p })}
                title={p.description}
              >
                {p.template?.thumbnail ? (
                  <img className="template-thumb" src={universeThumbnailUrl(p)} alt="" loading="lazy" />
                ) : (
                  <span className="template-thumb template-thumb-empty" />
                )}
                <span className="template-name">
                  {p.name} <span className="template-version">{p.version}</span>
                </span>
                <span className="template-desc">{p.description}</span>
              </button>
            ))}
          </div>
          {loading && <span className="spinner" />}
          {!loading && results.length === 0 && !error && <p className="settings-hint">Không tìm thấy mẫu nào.</p>}
          {!loading && results.length < total && (
            <button className="template-more" onClick={() => load(results.length)}>
              Xem thêm ({total - results.length})
            </button>
          )}
          {choice?.kind === 'universe' && (
            <p className="settings-hint">
              {choice.pkg.authors?.length ? `Tác giả: ${choice.pkg.authors.join(', ').replace(/\s*<[^>]*>/g, '')}. ` : ''}
              {choice.pkg.license ? `Giấy phép ${choice.pkg.license}. ` : ''}
              {/^https?:\/\//i.test(choice.pkg.repository ?? '') && (
                <a href={choice.pkg.repository} target="_blank" rel="noopener noreferrer">
                  Mã nguồn
                </a>
              )}
            </p>
          )}
        </>
      )}
    </Modal>
  )
}
