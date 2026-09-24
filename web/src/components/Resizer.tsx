/** Resizer is a vertical splitter between two panels. It uses Pointer
 * Events with pointer capture (mouse, pen and touch alike), and is a
 * focusable role="separator" whose arrow keys nudge the size; double-click
 * resets it. The parent owns the actual size math via onDrag/onStep. */
export function Resizer({
  label,
  className,
  onDrag,
  onStep,
  onReset,
  onActiveChange,
}: {
  label: string
  className?: string
  onDrag: (clientX: number) => void
  onStep: (direction: -1 | 1) => void
  onReset: () => void
  onActiveChange?: (active: boolean) => void
}) {
  return (
    <div
      className={`panel-resizer${className ? ` ${className}` : ''}`}
      role="separator"
      aria-orientation="vertical"
      aria-label={label}
      tabIndex={0}
      title={`${label} — kéo để đổi kích thước, nhấp đúp để về mặc định`}
      onPointerDown={(e) => {
        e.preventDefault()
        e.currentTarget.setPointerCapture(e.pointerId)
        e.currentTarget.classList.add('active')
        onActiveChange?.(true)
      }}
      onPointerMove={(e) => {
        if (e.currentTarget.hasPointerCapture(e.pointerId)) onDrag(e.clientX)
      }}
      onPointerUp={(e) => {
        e.currentTarget.releasePointerCapture(e.pointerId)
        e.currentTarget.classList.remove('active')
        onActiveChange?.(false)
      }}
      onPointerCancel={(e) => {
        e.currentTarget.classList.remove('active')
        onActiveChange?.(false)
      }}
      onDoubleClick={onReset}
      onKeyDown={(e) => {
        if (e.key === 'ArrowLeft') {
          e.preventDefault()
          onStep(-1)
        } else if (e.key === 'ArrowRight') {
          e.preventDefault()
          onStep(1)
        } else if (e.key === 'Home' || e.key === 'Enter') {
          e.preventDefault()
          onReset()
        }
      }}
    />
  )
}
