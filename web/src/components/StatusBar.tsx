import { Icon } from './Icon'

export interface DiagnosticCounts {
  errors: number
  warnings: number
}

/** StatusBar is the thin strip under the workspace: file, LSP diagnostics,
 * cursor position and save state. */
export function StatusBar({
  activePath,
  cursor,
  diagnostics,
  dirty,
  lastSaved,
}: {
  activePath: string | null
  cursor: { line: number; col: number } | null
  diagnostics: DiagnosticCounts
  dirty: boolean
  lastSaved: Date | null
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
      <span className="status-spacer" />
      {activePath && cursor && (
        <span className="status-item">
          Dòng {cursor.line}, Cột {cursor.col}
        </span>
      )}
      {activePath && <span className="status-item">{isTyp ? 'Typst' : 'Văn bản'} · UTF-8</span>}
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
