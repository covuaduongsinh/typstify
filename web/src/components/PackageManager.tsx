import { useState } from 'react'
import { ApiError, api } from '../api/client'
import type { SearchResponse, TypstPackage } from '../api/pkgTypes'

export function PackageManager() {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<TypstPackage[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [downloading, setDownloading] = useState<string | null>(null)

  const search = async () => {
    setBusy(true)
    setError(null)
    try {
      const res = await api.get<SearchResponse>(`/api/packages/search?query=${encodeURIComponent(query)}`)
      setResults(res.packages ?? [])
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Tìm kiếm thất bại')
    } finally {
      setBusy(false)
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
        version: pkg.latest_version,
      })
      setResults((prev) => prev.map((p) => (p === pkg ? { ...p, IsCached: true } : p)))
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Tải xuống thất bại')
    } finally {
      setDownloading(null)
    }
  }

  return (
    <div className="package-manager">
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

      {error && <div className="error">{error}</div>}

      <div className="package-list">
        {results.map((pkg) => {
          const key = `${pkg.namespace}/${pkg.name}`
          return (
            <div key={key} className="package-card">
              <div className="package-card-header">
                <span className="package-name">
                  @{pkg.namespace}/{pkg.name}
                </span>
                <span className="package-version">{pkg.latest_version}</span>
              </div>
              <div className="package-description">{pkg.description}</div>
              <div className="package-footer">
                <span className="package-license">{pkg.license}</span>
                {pkg.IsCached ? (
                  <span className="package-cached">Đã tải</span>
                ) : (
                  <button onClick={() => download(pkg)} disabled={downloading === key}>
                    {downloading === key ? 'Đang tải…' : 'Tải về'}
                  </button>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
