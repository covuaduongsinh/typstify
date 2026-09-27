/**
 * Utilities for processing and sanitizing Typst SVG output for seamless multi-page preview.
 */

export interface SvgDimensions {
  width: number
  height: number
  aspectRatio: number
}

/**
 * scopeSvgIds modifies all id references inside an SVG string to have a unique
 * prefix based on page index, preventing DOM symbol ID collisions across pages.
 *
 * Handles ALL patterns that reference fragment IDs in SVG:
 *   - id="..."              (definitions)
 *   - href="#..."           (use elements, gradients)
 *   - xlink:href="#..."     (legacy use elements)
 *   - url(#...)             (fill, stroke, clip-path, mask, filter attributes)
 *   - url("#...")           (quoted form of url references)
 */
export function scopeSvgIds(svgString: string, pageIndex: number): string {
  const prefix = `p${pageIndex}_`

  return svgString
    // Scope id="..." definitions (both double and single quotes)
    .replace(/\bid="([^"]+)"/g, `id="${prefix}$1"`)
    .replace(/\bid='([^']+)'/g, `id='${prefix}$1'`)
    // Scope xlink:href="#..." BEFORE plain href to avoid double-prefixing
    .replace(/\bxlink:href="#([^"]+)"/g, `xlink:href="#${prefix}$1"`)
    .replace(/\bxlink:href='#([^']+)'/g, `xlink:href='#${prefix}$1'`)
    // Scope href="#..." (plain href, not xlink:href - use negative lookbehind)
    .replace(/(?<!xlink:)href="#([^"]+)"/g, `href="#${prefix}$1"`)
    .replace(/(?<!xlink:)href='#([^']+)'/g, `href='#${prefix}$1'`)
    // Scope url(#...) references used in fill, stroke, clip-path, mask, filter
    .replace(/url\(#([^)]+)\)/g, `url(#${prefix}$1)`)
    // Scope url("#...") and url('...') quoted forms
    .replace(/url\("#([^"]+)"\)/g, `url("#${prefix}$1")`)
    .replace(/url\('#([^']+)'\)/g, `url('#${prefix}$1')`)
}

/**
 * extractSvgDimensions parses viewBox or width/height attributes from the SVG tag
 * to determine its natural aspect ratio.
 */
export function extractSvgDimensions(svgString: string): SvgDimensions {
  const defaultDims: SvgDimensions = { width: 595.28, height: 841.89, aspectRatio: 595.28 / 841.89 }

  const viewBoxMatch = svgString.match(/viewBox=["']\s*([0-9.-]+)\s+([0-9.-]+)\s+([0-9.-]+)\s+([0-9.-]+)\s*["']/i)
  if (viewBoxMatch) {
    const w = parseFloat(viewBoxMatch[3])
    const h = parseFloat(viewBoxMatch[4])
    if (Number.isFinite(w) && Number.isFinite(h) && w > 0 && h > 0) {
      return { width: w, height: h, aspectRatio: w / h }
    }
  }

  const widthMatch = svgString.match(/width=["']\s*([0-9.-]+)(?:pt|px)?\s*["']/i)
  const heightMatch = svgString.match(/height=["']\s*([0-9.-]+)(?:pt|px)?\s*["']/i)

  if (widthMatch && heightMatch) {
    const w = parseFloat(widthMatch[1])
    const h = parseFloat(heightMatch[1])
    if (Number.isFinite(w) && Number.isFinite(h) && w > 0 && h > 0) {
      return { width: w, height: h, aspectRatio: w / h }
    }
  }

  return defaultDims
}
