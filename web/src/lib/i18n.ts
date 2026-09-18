// Minimal i18n bridge to the desktop app's translation catalog
// (i18n/translations, generated from golang.org/x/text -- see
// server/i18n_api.go). Only en-US/zh-CN/de are supported, matching
// i18n/localizer.go's Locales list. Keys are the same English source
// strings the desktop app's Translate() calls use; a key with no catalog
// entry for the current locale falls back to itself.
import { useEffect, useState } from 'react'

const SUPPORTED_LOCALES = ['en-US', 'zh-CN', 'de'] as const
type Locale = (typeof SUPPORTED_LOCALES)[number]

function detectLocale(): Locale {
  const nav = navigator.language.toLowerCase()
  const match = SUPPORTED_LOCALES.find((l) => nav.startsWith(l.split('-')[0].toLowerCase()))
  return match ?? 'en-US'
}

export const locale: Locale = detectLocale()

const dictionary: Record<string, string> = {}
const listeners = new Set<() => void>()

async function loadTranslations(keys: string[]) {
  if (locale === 'en-US') return // catalog fallback for en-US is the key itself anyway
  const missing = keys.filter((k) => !(k in dictionary))
  if (missing.length === 0) return

  try {
    const res = await fetch('/api/i18n', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ locale, keys: missing }),
    })
    if (!res.ok) return
    Object.assign(dictionary, await res.json())
    listeners.forEach((l) => l())
  } catch {
    // offline/unreachable: keep using source-language fallback
  }
}

/** useTranslations(keys) returns a t(key) function that resolves to the
 * translated string once loaded, falling back to key itself until then
 * (or permanently, for en-US / untranslated keys). */
export function useTranslations(keys: string[]): (key: string) => string {
  const [, forceRender] = useState(0)

  useEffect(() => {
    const listener = () => forceRender((n) => n + 1)
    listeners.add(listener)
    void loadTranslations(keys)
    return () => {
      listeners.delete(listener)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [keys.join('|')])

  return (key: string) => dictionary[key] ?? key
}
