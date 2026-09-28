import { useState } from 'react'
import { ApiError, api } from '../api/client'
import { PACKAGE_CATEGORIES, type SearchResponse, type TypstPackage, type TypstPackageDetail } from '../api/pkgTypes'

const KIND_OPTIONS: Array<{ value: string; label: string }> = [
  { value: '', label: 'Tất cả loại' },
  { value: 'pkg', label: 'Package' },
  { value: 'template', label: 'Template' },
]

type Tab = 'search' | 'cache'

export function PackageManager() {
  const [tab, setTab] = useState<Tab>('search')
  const [query, setQuery] = useState('')
  const [kind, setKind] = useState('')
  const [category, setCategory] = useState('')
  const [results, setResults] = useState<TypstPackage[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [downloading, setDownloading] = useState<string | null>(null)

  const [cached, setCached] = useState<TypstPackage[]>([])
  const [cacheLoaded, setCacheLoaded] = useState(false)
  const [cacheBusy, setCacheBusy] = useState(false)

  const [expanded, setExpanded] = useState<string | null>(null)
  const [details, setDetails] = useState<Record<string, TypstPackageDetail>>({})
  const [selectedVersion, setSelectedVersion] = useState<Record<string, string>>({})

  const [pullingDeps, setPullingDeps] = useState(false)
  const [pullDepsMsg, setPullDepsMsg] = useState<string | null>(null)

  const search = async () => {
    setBusy(true)
    setError(null)
    try {
      const params = new URLSearchParams({ query })
      if (kind) params.set('kind', kind)
      if (category) params.set('category', category)
      const res = await api.get<SearchResponse>(`/api/packages/search?${params.toString()}`)
      setResults(res.packages ?? [])
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

  const download = async (pkg: TypstPackage) => {
    const key = `${pkg.namespace}/${pkg.name}`
    setDownloading(key)
    setError(null)
    try {
      await api.post('/api/packages/download', {
        namespace: pkg.namespace,
        name: pkg.name,
        version: selectedVersion[key] || pkg.latest_version,
      })
      setResults((prev) => prev.map((p) => (p === pkg ? { ...p, IsCached: true } : p)))
      if (cacheLoaded) void loadCache()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Tải xuống thất bại')
    } finally {
      setDownloading(null)
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

  const renderCard = (pkg: TypstPackage, opts: { showDownload: boolean }) => {
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
            <button className="package-version-btn" onClick={() => toggleVersions(pkg)}>
              {isExpanded ? 'Ẩn phiên bản' : 'Chọn phiên bản'}
            </button>
            {opts.showDownload &&
              (pkg.IsCached && !selectedVersion[key] ? (
                <span className="package-cached">Đã tải</span>
              ) : (
                <button onClick={() => download(pkg)} disabled={downloading === key}>
                  {downloading === key ? 'Đang tải…' : 'Tải về'}
                </button>
              ))}
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

          <div className="package-list">{results.map((pkg) => renderCard(pkg, { showDownload: true }))}</div>
        </>
      ) : (
        <div className="package-list">
          {cacheBusy ? (
            <span>Đang tải cache…</span>
          ) : cached.length === 0 ? (
            <span className="package-versions-loading">Chưa có gói nào trong cache cục bộ.</span>
          ) : (
            cached.map((pkg) => renderCard(pkg, { showDownload: false }))
          )}
        </div>
      )}
    </div>
  )
}
