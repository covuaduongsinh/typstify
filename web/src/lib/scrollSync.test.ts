import { describe, expect, it } from 'vitest'
import { buildPageLineMap, calculateLineRatio, calculateScrollTarget, cursorLineToPageRatio } from './scrollSync'

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
