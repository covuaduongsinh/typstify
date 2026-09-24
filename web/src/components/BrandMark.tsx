/** BrandMark is the Dương Sinh chess-square logo (same drawing as
 * public/favicon.svg) with an optional wordmark next to it. */
export function BrandMark({ size = 24, withText = false }: { size?: number; withText?: boolean }) {
  return (
    <span className="brand-mark">
      <svg width={size} height={size} viewBox="0 0 32 32" aria-hidden="true">
        <rect width="32" height="32" rx="7" fill="#2B3990" />
        <g fill="#C9A227">
          <rect x="6" y="6" width="5" height="5" />
          <rect x="16" y="6" width="5" height="5" />
          <rect x="11" y="11" width="5" height="5" />
          <rect x="21" y="11" width="5" height="5" />
          <rect x="6" y="16" width="5" height="5" />
          <rect x="16" y="16" width="5" height="5" />
          <rect x="11" y="21" width="5" height="5" />
          <rect x="21" y="21" width="5" height="5" />
        </g>
      </svg>
      {withText && (
        <span className="brand-mark-text">
          Dương Sinh <b>Chess Studio</b>
        </span>
      )}
    </span>
  )
}
