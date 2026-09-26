import { useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { DropboxProjectItem, DropboxStatus } from '../api/dropboxTypes'
import { Icon } from './Icon'
import { Modal } from './Modal'

interface DropboxImportModalProps {
  isOpen: boolean
  onClose: () => void
  onImported: (localPath: string) => void
}

export function DropboxImportModal({ isOpen, onClose, onImported }: DropboxImportModalProps) {
  const [status, setStatus] = useState<DropboxStatus | null>(null)
  const [projects, setProjects] = useState<DropboxProjectItem[]>([])
  const [loading, setLoading] = useState(false)
  const [importing, setImporting] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const loadData = async () => {
    try {
      setLoading(true)
      setError(null)
      const st = await api.get<DropboxStatus>('/api/dropbox/status')
      setStatus(st)

      if (st.connected) {
        const list = await api.get<DropboxProjectItem[]>('/api/dropbox/projects')
        setProjects(list || [])
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Không tải được danh sách dự án Dropbox')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (isOpen) {
      void loadData()
    }
  }, [isOpen])

  const handleImport = async (remotePath: string) => {
    try {
      setImporting(remotePath)
      setError(null)
      const res = await api.post<{ path: string; downloaded: number }>('/api/dropbox/import', {
        remotePath,
      })
      onImported(res.path)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Không thể nhập dự án từ Dropbox')
    } finally {
      setImporting(null)
    }
  }

  if (!isOpen) return null

  return (
    <Modal title="Nhập dự án từ Dropbox" onClose={onClose}>
      <div className="dropbox-modal-content">
        {error && (
          <div className="error" role="alert">
            <Icon name="error" size={14} /> {error}
          </div>
        )}

        {!status?.connected ? (
          <div className="dropbox-empty">
            <Icon name="cloud" size={32} />
            <p>Bạn chưa kết nối tài khoản Dropbox.</p>
            <p className="settings-hint">
              Vui lòng vào phần <strong>Cài đặt &gt; Đồng bộ Dropbox</strong> để kết nối tài khoản trước.
            </p>
          </div>
        ) : loading ? (
          <div className="dropbox-loading">
            <span className="spinner" /> Đang tải danh sách dự án từ Dropbox…
          </div>
        ) : projects.length === 0 ? (
          <div className="dropbox-empty">
            <Icon name="folder" size={32} />
            <p>Không tìm thấy thư mục dự án nào trong {status.syncFolder || '/Typstify'}.</p>
          </div>
        ) : (
          <div className="dropbox-project-list">
            <p className="settings-hint">
              Chọn một dự án trên Dropbox để tải về và mở trên máy:
            </p>
            <div className="recent-grid">
              {projects.map((proj) => (
                <div key={proj.path} className="dropbox-project-item">
                  <div className="proj-info">
                    <Icon name="folder" size={20} />
                    <div>
                      <span className="proj-name">{proj.name}</span>
                      <small className="proj-path">{proj.path}</small>
                    </div>
                  </div>
                  <button
                    className="btn-primary btn-sm"
                    onClick={() => handleImport(proj.path)}
                    disabled={importing !== null}
                  >
                    {importing === proj.path ? (
                      <>
                        <span className="spinner" /> Đang tải…
                      </>
                    ) : (
                      <>
                        <Icon name="cloud-download" size={14} /> Nhập
                      </>
                    )}
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="modal-actions" style={{ marginTop: 16 }}>
          <button className="btn-secondary" onClick={onClose}>
            Đóng
          </button>
        </div>
      </div>
    </Modal>
  )
}
