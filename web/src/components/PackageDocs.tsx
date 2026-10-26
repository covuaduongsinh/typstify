import { useEffect, useState } from 'react'
import Markdown from 'react-markdown'
import { ApiError, api } from '../api/client'
import { Modal } from './Modal'

/** README of a Typst Universe package. Raw HTML is dropped (skipHtml) and
 * images are shown as their alt text: they are usually paths inside the
 * package's repository and would not load. */
export function PackageDocs({ name, version, onClose }: { name: string; version: string; onClose: () => void }) {
  const [readme, setReadme] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let alive = true
    api
      .get<{ readme: string }>(`/api/universe/readme/${encodeURIComponent(name)}/${encodeURIComponent(version)}`)
      .then((r) => alive && setReadme(r.readme))
      .catch((err: unknown) => alive && setError(err instanceof ApiError ? err.message : 'Không tải được tài liệu'))
    return () => {
      alive = false
    }
  }, [name, version])

  return (
    <Modal title={`@preview/${name}:${version}`} onClose={onClose} className="package-docs-modal">
      {error && (
        <div className="error" role="alert">
          {error}
        </div>
      )}
      {readme === null && !error && <span className="spinner" />}
      {readme === '' && <p className="settings-hint">Gói này không có README.</p>}
      {readme && (
        <div className="package-docs">
          <Markdown
            skipHtml
            components={{
              a: ({ href, children }) => (
                <a href={href} target="_blank" rel="noopener noreferrer">
                  {children}
                </a>
              ),
              img: ({ alt }) => (alt ? <em>[{alt}]</em> : null),
            }}
          >
            {readme}
          </Markdown>
        </div>
      )}
    </Modal>
  )
}
