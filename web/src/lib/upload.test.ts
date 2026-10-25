import { describe, expect, it } from 'vitest'
import { imageMarkup, isImageFile, pastedImageName } from './upload'

describe('upload helpers', () => {
  it('recognises images by type or extension', () => {
    expect(isImageFile(new File([''], 'image.png', { type: 'image/png' }))).toBe(true)
    expect(isImageFile(new File([''], 'ban-co.SVG'))).toBe(true)
    expect(isImageFile(new File([''], 'data.csv', { type: 'text/csv' }))).toBe(false)
  })

  it('names pasted images by date and type', () => {
    const f = new File([''], 'image.png', { type: 'image/jpeg' })
    expect(pastedImageName(f, new Date(2026, 9, 25, 14, 30, 5))).toBe('anh-20261025-143005.jpg')
  })

  it('writes root-relative image markup', () => {
    expect(imageMarkup('assets/anh.png')).toBe('#image("/assets/anh.png", width: 80%)')
  })
})
