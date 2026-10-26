import { applyUpdate, useOnline, useUpdateState } from '../lib/pwa'

/** App-wide notices: offline, and a new version to reload into. */
export function PwaBanners() {
  const online = useOnline()
  const update = useUpdateState()
  if (online && update === 'none') return null
  return (
    <div className="pwa-banners" role="status" aria-live="polite">
      {!online && (
        <div className="pwa-banner pwa-offline">
          Đang ngoại tuyến. Bản đang mở vẫn soạn được; thay đổi được đồng bộ khi có mạng trở lại.
        </div>
      )}
      {update !== 'none' && (
        <div className="pwa-banner">
          {update === 'waiting' ? 'Có phiên bản mới của ứng dụng.' : 'Ứng dụng đã được cập nhật ở một tab khác.'}
          <button className="btn-primary" onClick={applyUpdate}>
            Tải lại
          </button>
        </div>
      )}
    </div>
  )
}
