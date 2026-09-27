import { describe, expect, it } from 'vitest'
import { calculateLineRatio, calculateScrollTarget } from './scrollSync'

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
