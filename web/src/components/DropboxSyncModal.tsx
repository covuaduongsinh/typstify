import { useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { DropboxStatus, DropboxSyncResult } from '../api/dropboxTypes'
import { Icon } from './Icon'
import { Modal } from './Modal'

interface DropboxSyncModalProps {
  isOpen: boolean
  onClose: () => void
  onSyncCompleted?: () => void
}

export function DropboxSyncModal({ isOpen, onClose, onSyncCompleted }: DropboxSyncModalProps) {
  const [status, setStatus] = useState<DropboxStatus | null>(null)
  const [loading, setLoading] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lastResult, setLastResult] = useState<DropboxSyncResult | null>(null)

  // Manual configuration inputs
  const [showConfig, setShowConfig] = useState(false)
  const [accessToken, setAccessToken] = useState('')
  const [refreshToken, setRefreshToken] = useState('')
  const [appKey, setAppKey] = useState('')
  const [appSecret, setAppSecret] = useState('')
  const [syncFolder, setSyncFolder] = useState('/Typstify')
  const [autoSync, setAutoSync] = useState(false)
  const [autoSyncInterval, setAutoSyncInterval] = useState(10)
  const [syncOnSave, setSyncOnSave] = useState(false)

  const loadStatus = async () => {
    try {
      setLoading(true)
      const res = await api.get<DropboxStatus>('/api/dropbox/status')
      setStatus(res)
      if (res.lastResult) setLastResult(res.lastResult)
      setSyncFolder(res.syncFolder || '/Typstify')
      setAutoSync(res.autoSync)
      setAutoSyncInterval(res.autoSyncInterval || 10)
      setSyncOnSave(res.syncOnSave)
      if (res.appKey) setAppKey(res.appKey)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Không tải được trạng thái Dropbox')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (isOpen) {
      void loadStatus()
    }
  }, [isOpen])

  const handleSync = async (mode: 'two-way' | 'push' | 'pull') => {
    try {
      setSyncing(true)
      setError(null)
      const res = await api.post<DropboxSyncResult>('/api/dropbox/sync', { mode })
      setLastResult(res)
      await loadStatus()
      if (onSyncCompleted) onSyncCompleted()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Đồng bộ thất bại')
    } finally {
      setSyncing(false)
    }
  }

  const handleDisconnect = async () => {
    if (!confirm('Bạn có chắc muốn ngắt kết nối với Dropbox?')) return
    try {
      setLoading(true)
      await api.post('/api/dropbox/auth/disconnect', {})
      await loadStatus()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Ngắt kết nối thất bại')
    } finally {
      setLoading(false)
    }
  }

  const handleSaveConfig = async () => {
    try {
      setLoading(true)
      setError(null)
      await api.post('/api/dropbox/auth/token', {
        accessToken: accessToken.trim(),
        refreshToken: refreshToken.trim(),
        appKey: appKey.trim(),
        appSecret: appSecret.trim(),
        syncFolder: syncFolder.trim() || '/Typstify',
        autoSync,
        autoSyncInterval: Number(autoSyncInterval) || 10,
        syncOnSave,
      })
      setShowConfig(false)
      setAccessToken('')
      setRefreshToken('')
      await loadStatus()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Lưu cấu hình thất bại')
    } finally {
      setLoading(false)
    }
  }

  const handleOAuthConnect = async () => {
    try {
      setLoading(true)
      setError(null)
      const res = await api.post<{ url: string }>('/api/dropbox/auth/url', {
        appKey: appKey.trim(),
        redirectUri: window.location.origin + '/api/dropbox/auth/callback',
      })
      if (res.url) {
        window.open(res.url, '_blank', 'width=600,height=700')
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Không tạo được liên kết đăng nhập Dropbox')
    } finally {
      setLoading(false)
    }
  }

  if (!isOpen) return null

  return (
    <Modal title="Đồng bộ Dropbox" onClose={onClose}>
      <div className="dropbox-modal-content">
        {error && (
          <div className="error" role="alert">
            <Icon name="error" size={14} /> {error}
          </div>
        )}

        {/* Connection status */}
        <div className="dropbox-card">
          <div className="dropbox-header">
            <div className="dropbox-brand">
              <Icon name="cloud" size={24} className="dropbox-icon" />
              <div>
                <h3 className="dropbox-title">
                  {status?.connected ? 'Đã kết nối Dropbox' : 'Chưa kết nối Dropbox'}
                </h3>
                <p className="dropbox-sub">
                  {status?.connected && status.account
                    ? `${status.account.display_name} (${status.account.email})`
                    : 'Lưu trữ và đồng bộ hóa tài liệu Typstify với tài khoản Dropbox của bạn.'}
                </p>
              </div>
            </div>
            {status?.connected ? (
              <button className="btn-secondary btn-sm" onClick={handleDisconnect} disabled={loading || syncing}>
                Ngắt kết nối
              </button>
            ) : (
              <button className="btn-primary btn-sm" onClick={() => setShowConfig(!showConfig)}>
                {showConfig ? 'Đóng cấu hình' : 'Kết nối'}
              </button>
            )}
          </div>
        </div>

        {/* Sync Actions (when connected) */}
        {status?.connected && (
          <div className="dropbox-actions-section">
            <h4>Hành động đồng bộ</h4>
            <div className="dropbox-btn-grid">
              <button
                className="btn-primary sync-action-btn"
                onClick={() => handleSync('two-way')}
                disabled={syncing || loading}
              >
                <Icon name="refresh" size={16} />
                <span>
                  <strong>Đồng bộ 2 chiều</strong>
                  <small>Cập nhật tệp mới nhất cả hai phía</small>
                </span>
              </button>

              <button
                className="btn-secondary sync-action-btn"
                onClick={() => handleSync('push')}
                disabled={syncing || loading}
              >
                <Icon name="cloud-upload" size={16} />
                <span>
                  <strong>Sao lưu lên Dropbox (Push)</strong>
                  <small>Gửi toàn bộ tệp máy lên Dropbox</small>
                </span>
              </button>

              <button
                className="btn-secondary sync-action-btn"
                onClick={() => handleSync('pull')}
                disabled={syncing || loading}
              >
                <Icon name="cloud-download" size={16} />
                <span>
                  <strong>Khôi phục từ Dropbox (Pull)</strong>
                  <small>Tải toàn bộ tệp từ Dropbox về máy</small>
                </span>
              </button>
            </div>
          </div>
        )}

        {/* Syncing Progress Spinner */}
        {syncing && (
          <div className="dropbox-syncing-banner">
            <span className="spinner" />
            <span>Đang tiến hành đồng bộ tệp với Dropbox…</span>
          </div>
        )}

        {/* Last Sync Result Summary */}
        {lastResult && (
          <div className={`dropbox-result-card ${lastResult.success ? 'success' : 'warning'}`}>
            <div className="result-header">
              <Icon name={lastResult.success ? 'check' : 'alert'} size={16} />
              <span>
                {lastResult.success ? 'Đồng bộ hoàn tất thành công' : 'Đồng bộ có lỗi'} (
                {lastResult.duration_ms}ms)
              </span>
            </div>
            <ul className="result-stats">
              <li>
                Tải lên: <strong>{lastResult.uploaded.length} tệp</strong>
              </li>
              <li>
                Tải về: <strong>{lastResult.downloaded.length} tệp</strong>
              </li>
              {lastResult.conflicts.length > 0 && (
                <li>
                  Xung đột: <strong>{lastResult.conflicts.length} tệp</strong>
                </li>
              )}
            </ul>
            {lastResult.errors.length > 0 && (
              <div className="result-errors">
                {lastResult.errors.map((e, idx) => (
                  <div key={idx} className="error-item">
                    • {e}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Settings / Manual Token config */}
        {(showConfig || !status?.connected) && (
          <div className="dropbox-config-box">
            <h4 className="config-title">Cấu hình kết nối & Tùy chọn</h4>
            <div className="config-fields">
              <label className="entry-field">
                <span>Access Token (hoặc Refresh Token)</span>
                <input
                  type="password"
                  placeholder="sl.u.A..."
                  value={accessToken}
                  onChange={(e) => setAccessToken(e.target.value)}
                />
                <small>Nhập Access Token được cấp từ Dropbox App Console.</small>
              </label>

              <div className="grid-2col">
                <label className="entry-field">
                  <span>App Key (tùy chọn)</span>
                  <input
                    placeholder="vd: abc123xyz"
                    value={appKey}
                    onChange={(e) => setAppKey(e.target.value)}
                  />
                </label>
                <label className="entry-field">
                  <span>App Secret (tùy chọn)</span>
                  <input
                    type="password"
                    placeholder="••••••••"
                    value={appSecret}
                    onChange={(e) => setAppSecret(e.target.value)}
                  />
                </label>
              </div>

              <label className="entry-field">
                <span>Thư mục gốc trên Dropbox</span>
                <input
                  value={syncFolder}
                  onChange={(e) => setSyncFolder(e.target.value)}
                  placeholder="/Typstify"
                />
              </label>

              <div className="checkbox-row">
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={autoSync}
                    onChange={(e) => setAutoSync(e.target.checked)}
                  />
                  <span>Tự động đồng bộ định kỳ</span>
                </label>
                {autoSync && (
                  <label className="inline-input">
                    <span>Mỗi</span>
                    <input
                      type="number"
                      min={1}
                      max={120}
                      value={autoSyncInterval}
                      onChange={(e) => setAutoSyncInterval(Number(e.target.value))}
                      style={{ width: 60 }}
                    />
                    <span>phút</span>
                  </label>
                )}
              </div>

              <div className="checkbox-row">
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={syncOnSave}
                    onChange={(e) => setSyncOnSave(e.target.checked)}
                  />
                  <span>Tự động đẩy lên Dropbox khi lưu tệp (Sync on save)</span>
                </label>
              </div>

              <div className="config-actions">
                {appKey && (
                  <button type="button" className="btn-secondary" onClick={handleOAuthConnect}>
                    Đăng nhập qua OAuth
                  </button>
                )}
                <button
                  type="button"
                  className="btn-primary"
                  onClick={handleSaveConfig}
                  disabled={loading}
                >
                  {loading ? 'Đang lưu…' : 'Lưu cấu hình'}
                </button>
              </div>
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
