/**
 * Utility functions for mapping editor cursor line to preview scroll offset.
 */

export interface PageLayoutInfo {
  top: number
  height: number
}

/**
 * Calculates a normalized line ratio from 0.0 to 1.0 based on 1-indexed line numbers.
 */
export function calculateLineRatio(line: number, totalLines: number): number {
  if (totalLines <= 1 || line <= 1) return 0
  if (line >= totalLines) return 1
  return (line - 1) / (totalLines - 1)
}

/**
 * Calculates the target scrollTop within a preview container for multi-page layouts.
 * Aligns the view so the active section is positioned naturally in the upper-middle view.
 */
export function calculateScrollTarget(
  lineRatio: number,
  pages: PageLayoutInfo[],
  containerHeight: number,
  containerScrollHeight?: number,
): number {
  const boundedRatio = Math.max(0, Math.min(1, lineRatio))

  // If no pages are measured yet, fallback to container scroll height proportion
  if (pages.length === 0) {
    if (!containerScrollHeight || containerScrollHeight <= containerHeight) return 0
    return boundedRatio * (containerScrollHeight - containerHeight)
  }

  // Multi-page layout
  const pageIndex = Math.min(pages.length - 1, Math.floor(boundedRatio * pages.length))
  const page = pages[pageIndex]
  if (!page) return 0

  const inPageRatio = boundedRatio * pages.length - pageIndex
  const targetY = page.top + inPageRatio * page.height - containerHeight * 0.25

  return Math.max(0, Math.round(targetY))
}
