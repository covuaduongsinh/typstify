import { describe, expect, it } from 'vitest'
import {
  buildPageLineMap,
  buildPageLineMapFromAnchors,
  calculateLineRatio,
  calculateScrollTarget,
  cursorLineToPageRatio,
  findHeadingLines,
} from './scrollSync'

describe('scrollSync', () => {
  describe('calculateLineRatio', () => {
    it('returns 0 for line 1 or single-line documents', () => {
      expect(calculateLineRatio(1, 100)).toBe(0)
      expect(calculateLineRatio(1, 1)).toBe(0)
      expect(calculateLineRatio(0, 100)).toBe(0)
    })

    it('returns 1 for last line', () => {
      expect(calculateLineRatio(100, 100)).toBe(1)
      expect(calculateLineRatio(120, 100)).toBe(1)
    })

    it('calculates intermediate line ratios accurately', () => {
      expect(calculateLineRatio(51, 101)).toBeCloseTo(0.5, 5)
      expect(calculateLineRatio(26, 101)).toBeCloseTo(0.25, 5)
      expect(calculateLineRatio(76, 101)).toBeCloseTo(0.75, 5)
    })
  })

  describe('buildPageLineMap', () => {
    it('returns [0] for single page', () => {
      const map = buildPageLineMap('line1\nline2\nline3', 1)
      expect(map).toEqual([0])
    })

    it('returns empty for 0 pages', () => {
      expect(buildPageLineMap('content', 0)).toEqual([])
    })

    it('detects explicit #pagebreak() to build accurate map', () => {
      const content = 'intro\n#pagebreak()\ncontent page 2\n#pagebreak()\ncontent page 3'
      const map = buildPageLineMap(content, 3)
      expect(map).toHaveLength(3)
      expect(map[0]).toBe(0)   // Page 1 starts at line 0
      expect(map[1]).toBe(2)   // Page 2 starts after #pagebreak() at line 1
      expect(map[2]).toBe(4)   // Page 3 starts after second #pagebreak() at line 3
    })

    it('distributes implicit page overflow proportionally, not dumped into the last segment', () => {
      // 3 explicit segments of very different length: 10 / 100 / 10 lines, 5 pages total
      // (2 "hidden" pages beyond the 3 explicit #pagebreak() segments). The old buggy
      // implementation always attributed every hidden page to the LAST segment
      // regardless of where the overflow actually happened, which would have produced
      // [0, 11, 112, 115, 118] here (2 extra pages wrongly stuffed into the tiny last
      // segment). The fix must instead give the extra pages to the segment that is
      // actually long enough to contain them (segment 1, the 100-line one).
      const seg0 = Array.from({ length: 10 }, (_, i) => `intro line ${i}`).join('\n')
      const seg1 = Array.from({ length: 100 }, (_, i) => `body line ${i}`).join('\n')
      const seg2 = Array.from({ length: 10 }, (_, i) => `outro line ${i}`).join('\n')
      const content = `${seg0}\n#pagebreak()\n${seg1}\n#pagebreak()\n${seg2}`

      const map = buildPageLineMap(content, 5)

      expect(map).toEqual([0, 11, 44, 78, 112])
      // Segment 0 (short) starts exactly at its explicit break and gets only 1 page.
      expect(map[1]).toBe(11)
      // Segment 2 (short, last) starts exactly at its explicit break, line 112 —
      // it must NOT have absorbed the 2 hidden pages meant for the long segment.
      expect(map[4]).toBe(112)
    })

    it('falls back to uniform distribution when no page breaks found', () => {
      const content = Array.from({ length: 100 }, (_, i) => `line ${i + 1}`).join('\n')
      const map = buildPageLineMap(content, 4)
      expect(map).toHaveLength(4)
      expect(map[0]).toBe(0)    // Page 1 at line 0
      expect(map[1]).toBe(25)   // Page 2 at 25% of 100 lines
      expect(map[2]).toBe(50)   // Page 3 at 50%
      expect(map[3]).toBe(75)   // Page 4 at 75%
    })
  })

  describe('findHeadingLines', () => {
    it('finds heading lines of any level, ignoring non-heading = usage', () => {
      const content = 'intro\n= Chapter One\ntext\n== Sub A\nmore\n=== Sub Sub\nend'
      expect(findHeadingLines(content)).toEqual([1, 3, 5])
    })

    it('does not match a bare "=" line with no heading text', () => {
      const content = 'a\n=\nb\n== \nc'
      expect(findHeadingLines(content)).toEqual([])
    })

    it('returns empty array for content with no headings', () => {
      expect(findHeadingLines('just some\nplain text\nno headings here')).toEqual([])
    })
  })

  describe('buildPageLineMapFromAnchors', () => {
    it('places exactly the known number of page-starts between consecutive anchors', () => {
      // Anchor at line 10 is on page 1, anchor at line 20 jumps to page 3 (2 page
      // starts must fall strictly between them), anchor at line 80 is page 4 (1
      // page start in between), then pageCount=5 implies 1 more page start after
      // line 80 (virtual end anchor at page 6). Splits divide each gap into
      // (pagesToPlace + 1) equal parts, since neither endpoint of a gap is
      // itself a known page boundary.
      const anchors = [
        { line: 10, page: 1 },
        { line: 20, page: 3 },
        { line: 80, page: 4 },
      ]
      const map = buildPageLineMapFromAnchors(anchors, 5, 100)
      expect(map).toEqual([0, 13, 16, 50, 86])
    })

    it('places no page-start between two anchors on the same page', () => {
      const anchors = [
        { line: 5, page: 1 },
        { line: 8, page: 1 }, // same page as previous anchor -> no split needed here
        { line: 50, page: 2 },
      ]
      const map = buildPageLineMapFromAnchors(anchors, 2, 100)
      // The two page-1 anchors (5, 8) contribute no page-start between them.
      // The single split for page 2 falls within (8, 50], at its midpoint
      // (8 + 42/2 = 29) since neither end of that gap is itself a known
      // page boundary -- only that page 2 starts somewhere inside it.
      expect(map).toEqual([0, 29])
    })

    it('ignores out-of-range anchors instead of corrupting the map', () => {
      const anchors = [
        { line: -1, page: 1 },
        { line: 50, page: 999 }, // page out of [1, pageCount]
        { line: 60, page: 2 },
      ]
      const map = buildPageLineMapFromAnchors(anchors, 3, 100)
      expect(map[0]).toBe(0)
      expect(map).toHaveLength(3)
    })
  })

  describe('buildPageLineMap with real anchors', () => {
    it('uses anchors instead of the #pagebreak() heuristic when anchors are provided', () => {
      const content = Array.from({ length: 100 }, (_, i) => `line ${i + 1}`).join('\n')
      const anchors = [
        { line: 10, page: 1 },
        { line: 20, page: 3 },
      ]
      const withAnchors = buildPageLineMap(content, 3, anchors)
      const withoutAnchors = buildPageLineMap(content, 3)
      expect(withAnchors).toEqual(buildPageLineMapFromAnchors(anchors, 3, 100))
      // Sanity: this should differ from the no-anchor uniform fallback, proving
      // anchors actually took priority instead of being silently ignored.
      expect(withAnchors).not.toEqual(withoutAnchors)
    })

    it('falls back to the existing heuristic when anchors is an empty array (safety gate tripped)', () => {
      const content = Array.from({ length: 100 }, (_, i) => `line ${i + 1}`).join('\n')
      expect(buildPageLineMap(content, 4, [])).toEqual(buildPageLineMap(content, 4))
    })
  })

  describe('cursorLineToPageRatio', () => {
    it('maps cursor to correct page ratio with explicit breaks', () => {
      // 3 pages: page1=[0,2), page2=[2,4), page3=[4,5)
      const map = [0, 2, 4] // 3 pages, total 5 lines
      const totalLines = 5

      // Line 1 (idx 0) → page 0 → ratio = 0/3 = 0
      expect(cursorLineToPageRatio(1, map, totalLines)).toBeCloseTo(0, 3)
      // Line 3 (idx 2) → page 1 → ratio = 1/3
      expect(cursorLineToPageRatio(3, map, totalLines)).toBeCloseTo(1 / 3, 2)
      // Line 5 (idx 4) → page 2 → ratio = 2/3
      expect(cursorLineToPageRatio(5, map, totalLines)).toBeCloseTo(2 / 3, 2)
    })

    it('returns 0 for empty or single-page map', () => {
      expect(cursorLineToPageRatio(50, [], 100)).toBe(0)
      expect(cursorLineToPageRatio(50, [0], 100)).toBe(0)
    })
  })

  describe('calculateScrollTarget', () => {
    it('handles fallback when pages array is empty', () => {
      expect(calculateScrollTarget(0.5, [], 400, 1400)).toBe(500)
      expect(calculateScrollTarget(0, [], 400, 1400)).toBe(0)
      expect(calculateScrollTarget(1, [], 400, 1400)).toBe(1000)
    })

    it('calculates multi-page targets smoothly', () => {
      const pages = [
        { top: 0, height: 800 },
        { top: 820, height: 800 },
        { top: 1640, height: 800 },
      ]
      const containerHeight = 600

      // Beginning of document (Page 1)
      const targetStart = calculateScrollTarget(0, pages, containerHeight)
      expect(targetStart).toBe(0) // targetY <= 0 clamped to 0

      // Middle of document (Page 2)
      const targetMid = calculateScrollTarget(0.5, pages, containerHeight)
      // ratio 0.5 * 3 pages = 1.5 => page index 1, inPageRatio 0.5
      // top = 820 + 0.5*800 - 600*0.25 = 820 + 400 - 150 = 1070
      expect(targetMid).toBe(1070)

      // End of document (Page 3)
      const targetEnd = calculateScrollTarget(1.0, pages, containerHeight)
      // ratio 1.0 => page index 2 (last page), inPageRatio 1.0
      // top = 1640 + 800 - 150 = 2290
      expect(targetEnd).toBe(2290)
    })
  })
})
