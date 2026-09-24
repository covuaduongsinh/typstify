// Light/dark theme switch. An explicit choice is stored in localStorage and
// mirrored onto <html data-theme="..."> (index.css keys its palettes off
// that attribute); with no stored choice the OS preference applies.
// index.html applies the stored value before first paint -- keep STORAGE_KEY
// in sync with the inline script there.
import { useEffect, useState } from 'react'

export type Theme = 'light' | 'dark'

const STORAGE_KEY = 'typstify_theme'
const media = window.matchMedia('(prefers-color-scheme: dark)')

function readStored(): Theme | null {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    return v === 'light' || v === 'dark' ? v : null
  } catch {
    return null
  }
}

function effectiveTheme(): Theme {
  return readStored() ?? (media.matches ? 'dark' : 'light')
}

export function useTheme(): [Theme, () => void] {
  const [theme, setTheme] = useState<Theme>(effectiveTheme)

  useEffect(() => {
    const onChange = () => setTheme(effectiveTheme())
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }, [])

  const toggle = () => {
    const next: Theme = theme === 'dark' ? 'light' : 'dark'
    document.documentElement.dataset.theme = next
    try {
      localStorage.setItem(STORAGE_KEY, next)
    } catch {
      // storage blocked: the switch still applies for this page load
    }
    setTheme(next)
  }

  return [theme, toggle]
}
