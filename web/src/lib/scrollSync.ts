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
 * Scans Typst source content and returns an array of source-line indices (0-based)
 * where each page begins. This allows more accurate cursor → page mapping by
 * detecting explicit `#pagebreak()` calls in the document.
 *
 * Returns an array of length `pageCount`, where element[i] is the 0-based source
 * line index that starts page i+1. Falls back gracefully to uniform distribution
 * if no page breaks are found.
 */
export function buildPageLineMap(content: string, pageCount: number): number[] {
  if (pageCount <= 0) return []
  if (!content || pageCount === 1) return [0]

  const lines = content.split('\n')
  const totalLines = lines.length

  // Find all lines containing explicit page breaks
  const breakLines: number[] = [0] // Page 1 always starts at line 0
  for (let i = 0; i < lines.length; i++) {
    const trimmed = lines[i].trim()
    if (
      trimmed === '#pagebreak()' ||
      trimmed === '#pagebreak(weak: true)' ||
      trimmed === '#pagebreak(weak: false)' ||
      trimmed.startsWith('#pagebreak(')
    ) {
      breakLines.push(i + 1) // Next page starts after the break line
    }
  }

  // If we found explicit breaks that match pageCount, use them directly
  if (breakLines.length === pageCount) {
    return breakLines
  }

  // If we found some breaks but not all (e.g. implicit breaks from long content),
  // distribute the remaining pages evenly within each segment between explicit breaks
  if (breakLines.length > 1 && breakLines.length < pageCount) {
    const result: number[] = []
    const explicitSegments = breakLines.length // number of explicit segments
    const extraPages = pageCount - explicitSegments // pages without explicit breaks

    for (let seg = 0; seg < breakLines.length; seg++) {
      const segStart = breakLines[seg]
      const segEnd = seg + 1 < breakLines.length ? breakLines[seg + 1] : totalLines

      const pagesInSeg = 1 + (seg === breakLines.length - 1 ? extraPages : 0)

      result.push(segStart)
      if (pagesInSeg > 1) {
        const segLines = segEnd - segStart
        for (let p = 1; p < pagesInSeg; p++) {
          result.push(segStart + Math.floor((p / pagesInSeg) * segLines))
        }
      }
    }
    return result.slice(0, pageCount)
  }

  // Fallback: uniform distribution across all lines
  const result: number[] = []
  for (let p = 0; p < pageCount; p++) {
    result.push(Math.floor((p / pageCount) * totalLines))
  }
  return result
}

/**
 * Maps a 1-based cursor line to a page index (0-based) using a pre-built page→line map.
 * Returns a fractional page index for smooth scrolling within a page.
 */
export function cursorLineToPageRatio(
  cursorLine: number,
  pageLineMap: number[],
  totalLines: number,
): number {
  if (pageLineMap.length === 0) return 0
  if (pageLineMap.length === 1) return 0

  const line0 = cursorLine - 1 // convert to 0-based
  const pageCount = pageLineMap.length

  // Binary search for which page contains this line
  let lo = 0
  let hi = pageCount - 1
  while (lo < hi) {
    const mid = Math.floor((lo + hi + 1) / 2)
    if (pageLineMap[mid] <= line0) {
      lo = mid
    } else {
      hi = mid - 1
    }
  }

  const pageIdx = lo
  const pageStartLine = pageLineMap[pageIdx]
  const pageEndLine = pageIdx + 1 < pageCount ? pageLineMap[pageIdx + 1] : totalLines
  const linesInPage = Math.max(1, pageEndLine - pageStartLine)
  const inPageRatio = Math.min(1, (line0 - pageStartLine) / linesInPage)

  // Return fractional index: pageIdx + inPageRatio, normalized to [0, 1]
  return (pageIdx + inPageRatio) / pageCount
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
