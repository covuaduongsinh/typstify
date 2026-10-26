import { describe, expect, it } from 'vitest'
import { isMobileApp } from './platform'

const android = 'Mozilla/5.0 (Linux; Android 16; sdk_gphone64_x86_64) AppleWebKit/537.36 Chrome/140 Mobile Safari/537.36'
const windows = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/140 Safari/537.36 Edg/140'

describe('isMobileApp', () => {
  it('is the Tauri shell on a phone', () => {
    expect(isMobileApp({ navigator: { userAgent: android }, __TAURI_INTERNALS__: {} })).toBe(true)
  })
  it('is not the desktop app or a phone browser', () => {
    expect(isMobileApp({ navigator: { userAgent: windows }, __TAURI_INTERNALS__: {} })).toBe(false)
    expect(isMobileApp({ navigator: { userAgent: android } })).toBe(false)
  })
})
