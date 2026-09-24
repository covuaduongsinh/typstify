import { useState } from 'react'
import { Modal } from './Modal'

/** PromptDialog replaces window.prompt() with an in-app, on-brand dialog. */
export function PromptDialog({
  title,
  label,
  defaultValue = '',
  hint,
  confirmLabel = 'Đồng ý',
  onConfirm,
  onCancel,
}: {
  title: string
  label: string
  defaultValue?: string
  hint?: string
  confirmLabel?: string
  onConfirm: (value: string) => void
  onCancel: () => void
}) {
  const [value, setValue] = useState(defaultValue)
  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (value.trim()) onConfirm(value.trim())
  }

  return (
    <Modal title={title} onClose={onCancel} className="prompt-dialog">
      <form className="prompt-dialog-body" onSubmit={submit}>
        <label className="prompt-dialog-field">
          <span>{label}</span>
          <input value={value} onChange={(e) => setValue(e.target.value)} onFocus={(e) => e.target.select()} />
        </label>
        {hint && <p className="prompt-dialog-hint">{hint}</p>}
        <div className="prompt-dialog-actions">
          <button type="button" className="btn-ghost" onClick={onCancel}>
            Hủy
          </button>
          <button type="submit" className="btn-primary" disabled={!value.trim()}>
            {confirmLabel}
          </button>
        </div>
      </form>
    </Modal>
  )
}
