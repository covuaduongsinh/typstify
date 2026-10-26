import { useState } from 'react'
import { STARTER_TEMPLATES } from '../lib/chessTemplates'
import { Icon } from './Icon'
import { Modal } from './Modal'

export function NewDocModal({
  isOpen,
  onClose,
  onCreate,
  onOpenMarkdownImport,
}: {
  isOpen: boolean
  onClose: () => void
  onCreate: (fileName: string, content: string) => void
  onOpenMarkdownImport?: () => void
}) {
  const [fileName, setFileName] = useState('chess_document.typ')
  const [selectedTemplateId, setSelectedTemplateId] = useState('book')

  if (!isOpen) return null

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    let target = fileName.trim()
    if (!target) return
    if (!target.includes('.')) target += '.typ'
    const tmpl = STARTER_TEMPLATES.find((t) => t.id === selectedTemplateId) ?? STARTER_TEMPLATES[0]
    onCreate(target, tmpl.content)
  }

  return (
    <Modal
      title={
        <>
          <Icon name="file-plus" /> Tạo tài liệu cờ vua mới
        </>
      }
      onClose={onClose}
      className="new-doc-modal"
    >
      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
        <div className="config-group">
          <label style={{ fontWeight: 600, marginBottom: '6px', display: 'block' }}>Tên file:</label>
          <input
            type="text"
            className="chess-input"
            value={fileName}
            onChange={(e) => setFileName(e.target.value)}
            placeholder="VD: chuong1.typ, sach_bai_tap.typ"
            autoFocus
          />
        </div>

        <div className="config-group">
          <label style={{ fontWeight: 600, marginBottom: '6px', display: 'block' }}>Chọn Mẫu Tài Liệu (Template):</label>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', maxHeight: '280px', overflowY: 'auto' }}>
            {STARTER_TEMPLATES.map((tmpl) => (
              <div
                key={tmpl.id}
                onClick={() => setSelectedTemplateId(tmpl.id)}
                style={{
                  padding: '10px 12px',
                  borderRadius: '6px',
                  border: `1.5px solid ${selectedTemplateId === tmpl.id ? 'var(--color-accent, #2563eb)' : 'var(--color-border, #e2e8f0)'}`,
                  background: selectedTemplateId === tmpl.id ? 'var(--color-accent-soft, #eff6ff)' : 'transparent',
                  cursor: 'pointer',
                  transition: 'all 0.15s ease',
                }}
              >
                <div style={{ fontWeight: 600, fontSize: '13px', color: 'var(--color-text, #0f172a)' }}>
                  {tmpl.name}
                </div>
                <div style={{ fontSize: '11px', opacity: 0.75, marginTop: '2px' }}>{tmpl.desc}</div>
              </div>
            ))}
          </div>
        </div>

        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '8px' }}>
          <div>
            {onOpenMarkdownImport && (
              <button
                type="button"
                className="btn-ghost"
                onClick={() => {
                  onClose()
                  onOpenMarkdownImport()
                }}
                style={{ fontSize: '12px' }}
              >
                <Icon name="file-text" size={14} /> Hoặc chuyển từ file Markdown (.md)
              </button>
            )}
          </div>
          <div style={{ display: 'flex', gap: '8px' }}>
            <button type="button" className="btn-ghost" onClick={onClose}>
              Hủy
            </button>
            <button type="submit" className="btn-primary" disabled={!fileName.trim()}>
              <Icon name="check" size={14} /> Tạo tài liệu
            </button>
          </div>
        </div>
      </form>
    </Modal>
  )
}
