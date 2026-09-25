import { Icon } from './Icon'

export interface DiagnosticCounts {
  errors: number
  warnings: number
}

/** StatusBar is the thin strip under the workspace: file, LSP diagnostics,
 * cursor position, save state and quick-fix actions. */
export function StatusBar({
  activePath,
  cursor,
  diagnostics,
  dirty,
  lastSaved,
  saveError,
  onAutoFix,
}: {
  activePath: string | null
  cursor: { line: number; col: number } | null
  diagnostics: DiagnosticCounts
  dirty: boolean
  lastSaved: Date | null
  saveError?: string | null
  onAutoFix?: () => void
}) {
  const isTyp = activePath?.endsWith('.typ')
  return (
    <footer className="status-bar">
      <span className="status-item status-brand">Vui trí tuệ</span>
      {activePath && isTyp && (
        <span
          className={`status-item ${diagnostics.errors ? 'status-error' : diagnostics.warnings ? 'status-warning' : 'status-ok'}`}
          title="Chẩn đoán từ trình biên dịch Typst (LSP)"
        >
          {diagnostics.errors === 0 && diagnostics.warnings === 0 ? (
            <>
              <Icon name="check" size={13} /> Không có lỗi
            </>
          ) : (
            <>
              <Icon name="error" size={13} /> {diagnostics.errors} lỗi
              <Icon name="alert" size={13} /> {diagnostics.warnings} cảnh báo
            </>
          )}
        </span>
      )}
      {activePath && isTyp && diagnostics.errors > 0 && onAutoFix && (
        <button
          className="status-item"
          style={{
            background: 'var(--color-accent, #2563eb)',
            color: '#ffffff',
            border: 'none',
            borderRadius: '4px',
            padding: '2px 8px',
            cursor: 'pointer',
            fontSize: '11px',
            display: 'inline-flex',
            alignItems: 'center',
            gap: '4px',
          }}
          onClick={onAutoFix}
          title="Tự động sửa lỗi thiếu import thư viện cờ vua"
        >
          <Icon name="sparkles" size={12} /> Tự động sửa import
        </button>
      )}
      <span className="status-spacer" />
      {activePath && cursor && (
        <span className="status-item">
          Dòng {cursor.line}, Cột {cursor.col}
        </span>
      )}
      {activePath && <span className="status-item">{isTyp ? 'Typst' : 'Văn bản'} · UTF-8</span>}
      {activePath && saveError && (
        <span className="status-item status-save-error" role="alert" title={saveError}>
          <Icon name="error" size={13} /> Lưu thất bại
        </span>
      )}
      {activePath && (
        <span className={`status-item ${dirty ? 'status-dirty' : ''}`}>
          {dirty
            ? '● Chưa lưu'
            : lastSaved
              ? `Đã lưu ${lastSaved.toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit' })}`
              : 'Đã lưu'}
        </span>
      )}
    </footer>
  )
}
