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
 * A real anchor: a source line known (via Typst's own query() introspection,
 * not a guess) to fall on a given physical page. Headings are the natural
 * anchor source — see findHeadingLines.
 */
export interface PageAnchor {
  line: number // 0-based source line
  page: number // 1-based physical page
}

/**
 * Scans Typst source for heading lines (`=`, `==`, `===`, ...), returning
 * their 0-based line indices in document order. Used to correlate against
 * page numbers queried from the compiled document (see PreviewPane's anchors
 * fetch) — the Nth heading found here should be the Nth heading Typst's
 * query(heading) returns, PROVIDED the document doesn't programmatically
 * generate extra headings (e.g. chessbook's render-puzzle-collection does,
 * via #heading(...) calls with no literal "=" line). Callers must verify the
 * counts match before trusting any correlation built from this list.
 */
export function findHeadingLines(content: string): number[] {
  const lines = content.split('\n')
  const headingLines: number[] = []
  const headingPattern = /^=+\s+\S/
  for (let i = 0; i < lines.length; i++) {
    if (headingPattern.test(lines[i])) {
      headingLines.push(i)
    }
  }
  return headingLines
}

/**
 * Builds a page→line map from real (line, page) anchors — e.g. heading
 * positions queried from the compiled document via Typst's query() — instead
 * of guessing. Between two consecutive anchors, the exact number of page
 * starts to place is known (the difference in their page numbers), so each
 * segment only needs even distribution *within itself*, not the cross-
 * segment largest-remainder balancing buildPageLineMap needs for
 * #pagebreak()-only detection (where per-segment page counts aren't known).
 *
 * anchors need not be sorted or deduplicated; virtual anchors at line 0
 * (page 1) and totalLines (page pageCount+1) bound the real ones.
 */
export function buildPageLineMapFromAnchors(
  anchors: PageAnchor[],
  pageCount: number,
  totalLines: number,
): number[] {
  const sorted = [...anchors]
    .filter((a) => a.line >= 0 && a.line <= totalLines && a.page >= 1 && a.page <= pageCount)
    .sort((a, b) => a.line - b.line)

  const bounded: PageAnchor[] = [{ line: 0, page: 1 }, ...sorted, { line: totalLines, page: pageCount + 1 }]

  const pageLineMap = new Array<number>(pageCount).fill(0)
  let lastPage = 1 // pages are non-decreasing in document order; guard against noise
  for (let i = 0; i < bounded.length - 1; i++) {
    const from = bounded[i]
    const to = bounded[i + 1]
    const fromPage = Math.max(from.page, lastPage)
    const toPage = Math.max(to.page, fromPage)
    const pagesToPlace = toPage - fromPage
    if (pagesToPlace <= 0) continue

    // Neither `from` nor `to` is a known page boundary (they're just lines
    // confirmed to fall on fromPage/toPage respectively) -- so pagesToPlace
    // new boundaries divide the interval into (pagesToPlace + 1) equal
    // parts, not `pagesToPlace` parts. Dividing by pagesToPlace would anchor
    // the *last* boundary exactly at `to.line`, which is systematically too
    // late whenever pagesToPlace is small relative to a long gap between
    // anchors (verified against a real multi-page document: with a single
    // page-start needed across a 120-line gap, landing it at the very end
    // put an entire page's worth of content on the wrong side of the split).
    const segLines = Math.max(0, to.line - from.line)
    for (let j = 1; j <= pagesToPlace; j++) {
      const pageIdx = fromPage + j - 1 // 0-based index into pageLineMap
      if (pageIdx >= pageCount) break
      pageLineMap[pageIdx] = from.line + Math.floor((j / (pagesToPlace + 1)) * segLines)
    }
    lastPage = toPage
  }
  return pageLineMap
}

/**
 * Scans Typst source content and returns an array of source-line indices (0-based)
 * where each page begins. This allows more accurate cursor → page mapping by
 * detecting explicit `#pagebreak()` calls in the document.
 *
 * Returns an array of length `pageCount`, where element[i] is the 0-based source
 * line index that starts page i+1. Falls back gracefully to uniform distribution
 * if no page breaks are found.
 *
 * When `anchors` is provided (real page positions, e.g. from Typst's query()),
 * it takes priority over the #pagebreak() heuristic entirely — see
 * buildPageLineMapFromAnchors.
 */
export function buildPageLineMap(content: string, pageCount: number, anchors?: PageAnchor[]): number[] {
  if (pageCount <= 0) return []
  if (!content || pageCount === 1) return [0]

  const lines = content.split('\n')
  const totalLines = lines.length

  if (anchors && anchors.length > 0) {
    return buildPageLineMapFromAnchors(anchors, pageCount, totalLines)
  }

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
  // distribute the remaining pages across segments proportionally to each segment's
  // line count (largest-remainder method), instead of dumping them all into the last
  // segment — a long/heavy segment (e.g. a chess diagram or table) in the *middle* of
  // the document is just as likely to contain implicit page breaks as the last one.
  if (breakLines.length > 1 && breakLines.length < pageCount) {
    const result: number[] = []
    const segLengths = breakLines.map((start, i) =>
      (i + 1 < breakLines.length ? breakLines[i + 1] : totalLines) - start,
    )
    const totalSegLines = segLengths.reduce((a, b) => a + b, 0) || 1
    const rawShares = segLengths.map((len) => (pageCount * len) / totalSegLines)
    const pagesPerSeg = rawShares.map((s) => Math.max(1, Math.floor(s)))

    // Adjust so the total exactly equals pageCount, giving extra pages to the
    // segments with the largest fractional remainder first.
    let diff = pageCount - pagesPerSeg.reduce((a, b) => a + b, 0)
    const remainderOrder = rawShares
      .map((s, i) => ({ i, frac: s - Math.floor(s) }))
      .sort((a, b) => b.frac - a.frac)
    for (let k = 0; diff !== 0; k = (k + 1) % remainderOrder.length) {
      const idx = remainderOrder[k].i
      if (diff > 0) {
        pagesPerSeg[idx] += 1
        diff -= 1
      } else if (pagesPerSeg[idx] > 1) {
        pagesPerSeg[idx] -= 1
        diff += 1
      }
    }

    for (let seg = 0; seg < breakLines.length; seg++) {
      const segStart = breakLines[seg]
      const segEnd = seg + 1 < breakLines.length ? breakLines[seg + 1] : totalLines
      const pagesInSeg = pagesPerSeg[seg]

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
 * Inverse of cursorLineToPageRatio's positioning convention: given the
 * preview's current scrollTop, returns a normalized [0,1] ratio for where in
 * the document that scroll position sits. Anchors at `scrollTop +
 * containerHeight * 0.25`, matching the same offset calculateScrollTarget
 * subtracts when scrolling *to* a ratio, so a scroll round-trip (editor line
 * -> preview scroll -> back to editor line) settles instead of drifting.
 */
export function scrollTopToPageRatio(scrollTop: number, pages: PageLayoutInfo[], containerHeight: number): number {
  if (pages.length === 0) return 0
  const anchorY = scrollTop + containerHeight * 0.25

  let pageIdx = 0
  for (let i = 0; i < pages.length; i++) {
    if (anchorY >= pages[i].top) pageIdx = i
    else break
  }

  const page = pages[pageIdx]
  const inPageRatio = page.height > 0 ? Math.max(0, Math.min(1, (anchorY - page.top) / page.height)) : 0
  return (pageIdx + inPageRatio) / pages.length
}

/**
 * Inverse of cursorLineToPageRatio: maps a normalized [0,1] page ratio back
 * to a 1-based source line, using the same page->line map.
 */
export function pageRatioToLine(ratio: number, pageLineMap: number[], totalLines: number): number {
  if (pageLineMap.length === 0) return 1
  const pageCount = pageLineMap.length
  const boundedRatio = Math.max(0, Math.min(1, ratio))
  const scaled = boundedRatio * pageCount
  const pageIdx = Math.min(pageCount - 1, Math.floor(scaled))
  const inPageRatio = scaled - pageIdx

  const pageStartLine = pageLineMap[pageIdx]
  const pageEndLine = pageIdx + 1 < pageCount ? pageLineMap[pageIdx + 1] : totalLines
  const linesInPage = Math.max(1, pageEndLine - pageStartLine)
  const line0 = pageStartLine + inPageRatio * linesInPage

  return Math.max(1, Math.min(totalLines, Math.round(line0) + 1))
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
