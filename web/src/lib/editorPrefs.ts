// Personal editor preferences (Vim keys, spell checking), kept per
// browser like lib/theme.ts and lib/shortcuts.ts: they describe how this
// person types, not the project.
import { useSyncExternalStore } from 'react'

export interface EditorPrefs {
  vim: boolean
  spellcheck: boolean
}

const STORAGE_KEY = 'typstify_editor_prefs'
const DEFAULTS: EditorPrefs = { vim: false, spellcheck: false }

function read(): EditorPrefs {
  try {
    const parsed = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '{}')
    return { ...DEFAULTS, ...(parsed && typeof parsed === 'object' ? parsed : {}) }
  } catch {
    return DEFAULTS
  }
}

let current = read()
const listeners = new Set<() => void>()

export function setEditorPref<K extends keyof EditorPrefs>(key: K, value: EditorPrefs[K]) {
  current = { ...current, [key]: value }
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(current))
  } catch {
    // not persisted; still applies until reload
  }
  for (const fn of listeners) fn()
}

const subscribe = (fn: () => void) => {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

export const useEditorPrefs = () => useSyncExternalStore(subscribe, () => current)
