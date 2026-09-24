import { useEffect, useId, useRef, type ReactNode } from 'react'
import { Icon } from './Icon'

const FOCUSABLE = 'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

/** Modal is the shared dialog shell: role="dialog", Esc and backdrop click
 * close it, Tab stays inside it, and focus returns to whatever had it
 * before the dialog opened. */
export function Modal({
  title,
  onClose,
  children,
  footer,
  className,
}: {
  title: ReactNode
  onClose: () => void
  children: ReactNode
  footer?: ReactNode
  className?: string
}) {
  const boxRef = useRef<HTMLDivElement>(null)
  const titleId = useId()
  const onCloseRef = useRef(onClose)
  useEffect(() => {
    onCloseRef.current = onClose
  })

  useEffect(() => {
    const previouslyFocused = document.activeElement as HTMLElement | null
    const box = boxRef.current
    // Focus the first form field if there is one, else the dialog itself.
    const first = box?.querySelector<HTMLElement>('input, select, textarea') ?? box
    first?.focus()

    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation()
        onCloseRef.current()
        return
      }
      if (e.key !== 'Tab' || !box) return
      const items = Array.from(box.querySelectorAll<HTMLElement>(FOCUSABLE))
      if (items.length === 0) return
      const firstItem = items[0]
      const lastItem = items[items.length - 1]
      if (e.shiftKey && document.activeElement === firstItem) {
        e.preventDefault()
        lastItem.focus()
      } else if (!e.shiftKey && document.activeElement === lastItem) {
        e.preventDefault()
        firstItem.focus()
      }
    }
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('keydown', onKey)
      previouslyFocused?.focus?.()
    }
  }, [])

  return (
    <div className="chess-modal-overlay" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div
        ref={boxRef}
        className={`chess-modal-container${className ? ` ${className}` : ''}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
      >
        <div className="chess-modal-header">
          <h3 id={titleId}>{title}</h3>
          <button className="chess-modal-close btn-ghost" onClick={onClose} aria-label="Đóng">
            <Icon name="x" />
          </button>
        </div>
        {children}
        {footer && <div className="chess-modal-footer">{footer}</div>}
      </div>
    </div>
  )
}
