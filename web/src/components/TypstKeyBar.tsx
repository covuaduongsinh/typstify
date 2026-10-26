// Characters Typst markup leans on that phone keyboards bury behind one or
// two layer switches.
const KEYS = ['#', '$', '[', ']', '{', '}', '(', ')', '*', '_', '=', '-', '+', '/', '\\', '<', '>', '@', '"', ':', ',', '~']

/**
 * A row of quick-insert keys shown above the on-screen keyboard (phone
 * layout). Tapping a key must not take focus away from the editor: that
 * would close the keyboard, so the pointer-down default is cancelled.
 */
export function TypstKeyBar({ onInsert }: { onInsert: (text: string) => void }) {
  return (
    <div className="typst-keybar" role="toolbar" aria-label="Ký hiệu Typst">
      {KEYS.map((k) => (
        <button
          key={k}
          type="button"
          className="typst-key"
          onPointerDown={(e) => e.preventDefault()}
          onClick={() => onInsert(k)}
          aria-label={`Chèn ${k}`}
        >
          {k}
        </button>
      ))}
    </div>
  )
}
