import { lazy, Suspense, useState } from 'react'
import { ApiError, api } from '../api/client'
import { PACKAGE_CATEGORIES, type TypstPackage, type TypstPackageDetail } from '../api/pkgTypes'
import type { UniversePackage, UniversePage } from '../lib/universe'

const KIND_OPTIONS: Array<{ value: string; label: string }> = [
  { value: '', label: 'Tất cả loại' },
  { value: 'pkg', label: 'Package' },
  { value: 'template', label: 'Template' },
]

type Tab = 'search' | 'cache'

/** A Universe index entry in the panel's package shape. The index only
 * lists the latest version, and whether it is cached is not known. */
const fromUniverse = (p: UniversePackage): TypstPackage => ({
  name: p.name,
  namespace: 'preview',
  description: p.description ?? '',
  latest_version: p.version,
  published_at: '',
  license: p.license ?? '',
  is_template: !!p.template,
  authors: p.authors ?? null,
  categories: p.categories ?? null,
  IsCached: false,
})

const PackageDocs = lazy(() => import('./PackageDocs').then((m) => ({ default: m.PackageDocs })))

type ImportResult = 'added' | 'present' | 'readonly' | 'no-file'

const IMPORT_MESSAGES: Record<ImportResult, (spec: string, name: string) => string> = {
  added: (spec) => `Đã thêm #import "${spec}" vào đầu tệp.`,
  present: (_, name) => `Tệp đã import ${name}.`,
  readonly: () => 'Tệp đang mở chỉ đọc.',
  'no-file': () => 'Hãy mở một tệp .typ trước.',
}

/** onInsertImport adds `#import "@preview/name:version": *` to the open
 * document. */
export function PackageManager({
  onInsertImport,
}: { onInsertImport?: (name: string, version: string) => ImportResult } = {}) {
  const [tab, setTab] = useState<Tab>('search')
  const [query, setQuery] = useState('')
  const [kind, setKind] = useState('')
  const [category, setCategory] = useState('')
  const [results, setResults] = useState<TypstPackage[]>([])
  const [total, setTotal] = useState(0)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [cached, setCached] = useState<TypstPackage[]>([])
  const [cacheLoaded, setCacheLoaded] = useState(false)
  const [cacheBusy, setCacheBusy] = useState(false)

  const [expanded, setExpanded] = useState<string | null>(null)
  const [details, setDetails] = useState<Record<string, TypstPackageDetail>>({})
  const [selectedVersion, setSelectedVersion] = useState<Record<string, string>>({})

  const [pullingDeps, setPullingDeps] = useState(false)
  const [pullDepsMsg, setPullDepsMsg] = useState<string | null>(null)
  const [docs, setDocs] = useState<{ name: string; version: string } | null>(null)
  const [importMsg, setImportMsg] = useState<string | null>(null)

  const search = async () => {
    setBusy(true)
    setError(null)
    try {
      // Typst Universe's own index (packages.typst.org): no account needed.
      const params = new URLSearchParams({ q: query, limit: '100' })
      if (kind) params.set('kind', kind === 'pkg' ? 'package' : kind)
      if (category) params.set('category', category)
      const res = await api.get<UniversePage>(`/api/universe/packages?${params.toString()}`)
      setResults((res.packages ?? []).map(fromUniverse))
      setTotal(res.total)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Tìm kiếm thất bại')
    } finally {
      setBusy(false)
    }
  }

  const loadCache = async () => {
    setCacheBusy(true)
    setError(null)
    try {
      const res = await api.get<TypstPackage[]>('/api/packages/cached')
      setCached(res ?? [])
      setCacheLoaded(true)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Không đọc được cache gói')
    } finally {
      setCacheBusy(false)
    }
  }

  const switchTab = (next: Tab) => {
    setTab(next)
    if (next === 'cache' && !cacheLoaded) void loadCache()
  }

  const toggleVersions = async (pkg: TypstPackage) => {
    const key = `${pkg.namespace}/${pkg.name}`
    if (expanded === key) {
      setExpanded(null)
      return
    }
    setExpanded(key)
    if (!details[key]) {
      try {
        const spec = `@${pkg.namespace}/${pkg.name}`
        const detail = await api.get<TypstPackageDetail>(`/api/packages/detail?spec=${encodeURIComponent(spec)}`)
        setDetails((prev) => ({ ...prev, [key]: detail }))
      } catch (err) {
        setError(err instanceof ApiError ? err.message : 'Không tải được thông tin phiên bản')
      }
    }
  }

  const pullDeps = async () => {
    setPullingDeps(true)
    setPullDepsMsg(null)
    setError(null)
    try {
      await api.post('/api/packages/pull-deps')
      setPullDepsMsg('Đã tải xong toàn bộ dependency của dự án.')
      if (cacheLoaded) void loadCache()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Tải dependency thất bại')
    } finally {
      setPullingDeps(false)
    }
  }

  const renderCard = (pkg: TypstPackage, opts: { showVersions?: boolean }) => {
    const key = `${pkg.namespace}/${pkg.name}`
    const detail = details[key]
    const isExpanded = expanded === key
    return (
      <div key={key} className="package-card">
        <div className="package-card-header">
          <span className="package-name">
            @{pkg.namespace}/{pkg.name}
          </span>
          <span className="package-version">{selectedVersion[key] || pkg.latest_version}</span>
        </div>
        <div className="package-description">{pkg.description}</div>
        <div className="package-footer">
          <span className="package-license">{pkg.license}</span>
          <div className="package-actions">
            {pkg.namespace === 'preview' && (
              <>
                <button onClick={() => setDocs({ name: pkg.name, version: selectedVersion[key] || pkg.latest_version })}>
                  Tài liệu
                </button>
                {onInsertImport && (
                  <button
                    title="Thêm #import vào đầu tệp đang mở"
                    onClick={() => {
                      const version = selectedVersion[key] || pkg.latest_version
                      const result = onInsertImport(pkg.name, version)
                      setImportMsg(IMPORT_MESSAGES[result](`@preview/${pkg.name}:${version}`, pkg.name))
                    }}
                  >
                    Chèn #import
                  </button>
                )}
              </>
            )}
            {opts.showVersions !== false && (
              <button className="package-version-btn" onClick={() => toggleVersions(pkg)}>
                {isExpanded ? 'Ẩn phiên bản' : 'Chọn phiên bản'}
              </button>
            )}
          </div>
        </div>
        {isExpanded && (
          <div className="package-versions">
            {!detail ? (
              <span className="package-versions-loading">Đang tải danh sách phiên bản…</span>
            ) : (
              <select
                value={selectedVersion[key] || pkg.latest_version}
                onChange={(e) => setSelectedVersion((prev) => ({ ...prev, [key]: e.target.value }))}
              >
                {(detail.Versions ?? []).map((v) => (
                  <option key={v.version} value={v.version}>
                    {v.version}
                  </option>
                ))}
              </select>
            )}
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="package-manager">
      <div className="tab-buttons">
        <button className={`tab-btn ${tab === 'search' ? 'active' : ''}`} onClick={() => switchTab('search')}>
          Tìm kiếm
        </button>
        <button className={`tab-btn ${tab === 'cache' ? 'active' : ''}`} onClick={() => switchTab('cache')}>
          Cache cục bộ
        </button>
      </div>

      {error && <div className="error">{error}</div>}
      {importMsg && <div className="package-pulldeps-msg">{importMsg}</div>}

      {tab === 'search' ? (
        <>
          <div className="package-search">
            <input
              placeholder="Tìm gói Typst…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && search()}
            />
            <button onClick={search} disabled={busy}>
              {busy ? 'Đang tìm…' : 'Tìm'}
            </button>
          </div>
          <div className="package-filters">
            <select value={kind} onChange={(e) => setKind(e.target.value)}>
              {KIND_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
            <select value={category} onChange={(e) => setCategory(e.target.value)}>
              <option value="">Tất cả danh mục</option>
              {PACKAGE_CATEGORIES.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
            <button className="package-pulldeps-btn" onClick={pullDeps} disabled={pullingDeps}>
              {pullingDeps ? 'Đang tải deps…' : 'Pull toàn bộ dependency'}
            </button>
          </div>
          {pullDepsMsg && <div className="package-pulldeps-msg">{pullDepsMsg}</div>}

          {total > results.length && (
            <div className="package-pulldeps-msg">
              Hiện {results.length} / {total} gói, hãy thu hẹp từ khoá hoặc nhóm.
            </div>
          )}
          <div className="package-list">{results.map((pkg) => renderCard(pkg, { showVersions: false }))}</div>
        </>
      ) : (
        <div className="package-list">
          {cacheBusy ? (
            <span>Đang tải cache…</span>
          ) : cached.length === 0 ? (
            <span className="package-versions-loading">Chưa có gói nào trong cache cục bộ.</span>
          ) : (
            cached.map((pkg) => renderCard(pkg, {}))
          )}
        </div>
      )}
      {docs && (
        <Suspense fallback={null}>
          <PackageDocs name={docs.name} version={docs.version} onClose={() => setDocs(null)} />
        </Suspense>
      )}
    </div>
  )
}
