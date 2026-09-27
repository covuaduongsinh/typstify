/**
 * Utilities for processing and sanitizing Typst SVG output for seamless multi-page preview.
 */

export interface SvgDimensions {
  width: number
  height: number
  aspectRatio: number
}

/**
 * scopeSvgIds modifies all `id="..."`, `xlink:href="#..."`, and `href="#..."`
 * inside an SVG string to have a unique prefix based on page index.
 * This prevents symbol ID collisions when rendering dozens or hundreds of SVG pages
 * on the same HTML DOM.
 */
export function scopeSvgIds(svgString: string, pageIndex: number): string {
  const prefix = `p${pageIndex}_`
  return svgString
    .replace(/\bid="([^"]+)"/g, `id="${prefix}$1"`)
    .replace(/\b(xlink:href|href)="#([^"]+)"/g, `$1="#${prefix}$2"`)
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
