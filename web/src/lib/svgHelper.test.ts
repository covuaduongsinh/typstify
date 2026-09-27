import { describe, expect, it } from 'vitest'
import { scopeSvgIds, extractSvgDimensions } from './svgHelper'

describe('svgHelper', () => {
  it('scopes id, href, and xlink:href with page prefix', () => {
    const rawSvg = `<svg viewBox="0 0 100 200"><defs><symbol id="g123"><path d="M0 0"/></symbol></defs><g><use xlink:href="#g123" href="#g123"/></g></svg>`
    const scoped = scopeSvgIds(rawSvg, 3)

    expect(scoped).toContain('id="p3_g123"')
    expect(scoped).toContain('xlink:href="#p3_g123"')
    expect(scoped).toContain('href="#p3_g123"')
    expect(scoped).not.toContain('id="g123"')
    expect(scoped).not.toContain('xlink:href="#g123"')
  })

  it('scopes url(#...) references used in fill, clip-path, mask, filter', () => {
    const rawSvg = `<svg><defs><clipPath id="cp1"><rect/></clipPath><linearGradient id="grad1"/></defs><rect clip-path="url(#cp1)" fill="url(#grad1)"/></svg>`
    const scoped = scopeSvgIds(rawSvg, 5)

    expect(scoped).toContain('id="p5_cp1"')
    expect(scoped).toContain('id="p5_grad1"')
    expect(scoped).toContain('clip-path="url(#p5_cp1)"')
    expect(scoped).toContain('fill="url(#p5_grad1)"')
    expect(scoped).not.toContain('url(#cp1)')
    expect(scoped).not.toContain('url(#grad1)')
  })

  it('scopes single-quoted id attributes', () => {
    const rawSvg = `<svg><defs><symbol id='sym1'></symbol></defs><use href='#sym1'/></svg>`
    const scoped = scopeSvgIds(rawSvg, 2)

    expect(scoped).toContain("id='p2_sym1'")
    expect(scoped).toContain("href='#p2_sym1'")
  })

  it('extracts aspect ratio and viewBox dimensions correctly', () => {
    const rawSvg = `<svg viewBox="0 0 450 600" width="450pt" height="600pt"></svg>`
    const dims = extractSvgDimensions(rawSvg)
    expect(dims.width).toBe(450)
    expect(dims.height).toBe(600)
    expect(dims.aspectRatio).toBeCloseTo(450 / 600)
  })

  it('handles SVGs without viewBox gracefully', () => {
    const rawSvg = `<svg width="400" height="800"></svg>`
    const dims = extractSvgDimensions(rawSvg)
    expect(dims.width).toBe(400)
    expect(dims.height).toBe(800)
    expect(dims.aspectRatio).toBeCloseTo(0.5)
  })
})
