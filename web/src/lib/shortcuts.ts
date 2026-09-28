// Customizable keyboard shortcuts for the app's most-used chess-toolbar
// actions (ChessToolbar.tsx / Workspace.tsx). There is exactly one other
// app-level shortcut in this codebase worth this treatment -- everything
// else is either CodeMirror's own bundled keymap or a local widget
// convention (Escape closes a modal, Enter submits a search box) that isn't
// meaningfully "app-customizable". Overrides are a personal per-browser
// preference, stored the same way as lib/theme.ts (localStorage, no server
// round-trip).
import { useSyncExternalStore } from 'react'

export interface ShortcutAction {
  id: string
  label: string
  /** "Mod-Shift-b" style: Mod = Ctrl on Windows/Linux, Cmd on Mac, matching
   * CodeMirror's own key-string convention so one stored value works on
   * every platform. */
  defaultKey: string
}

// Single-character keys must be uppercase here: keyComboFromEvent /
// matchesCombo below always uppercase a single-char e.key before comparing
// (keyboard events report it lowercase absent Shift), so a lowercase
// default here would silently never match any keypress.
export const SHORTCUT_ACTIONS: ShortcutAction[] = [
  { id: 'save', label: 'Lưu tài liệu', defaultKey: 'Mod-S' },
  { id: 'openBoard', label: 'Xếp bàn cờ', defaultKey: 'Mod-B' },
  { id: 'openPgn', label: 'Nhập PGN', defaultKey: 'Mod-Shift-P' },
  { id: 'insertBoard', label: 'Chèn nhanh khung bàn cờ', defaultKey: 'Mod-Shift-B' },
]

const STORAGE_KEY = 'typstify_shortcuts'
const MODIFIER_KEYS = new Set(['Control', 'Meta', 'Shift', 'Alt'])
export const isMac = /Mac|iPhone|iPad/.test(navigator.platform)

function readStored(): Record<string, string> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    const parsed = raw ? JSON.parse(raw) : {}
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function writeStored(map: Record<string, string>) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(map))
  } catch {
    // storage blocked (private mode, blocked site data): overrides just
    // don't persist across reloads, same tradeoff as lib/theme.ts.
  }
}

// Module-level external store: Workspace.tsx's shortcut dispatcher and
// SettingsPanel's editor both call useShortcuts(), and a remap made in
// Settings must be visible to Workspace's dispatcher immediately (not just
// after a reload) -- plain per-component useState wouldn't do that (each
// call owns an independent copy), so this is a minimal shared store instead,
// read via useSyncExternalStore like any other external mutable state.
let overridesCache: Record<string, string> = readStored()
const listeners = new Set<() => void>()

function getSnapshot() {
  return overridesCache
}

function subscribe(onStoreChange: () => void) {
  listeners.add(onStoreChange)
  return () => listeners.delete(onStoreChange)
}

function commit(next: Record<string, string>) {
  overridesCache = next
  writeStored(next)
  for (const l of listeners) l()
}

/** Reactive access to the user's shortcut overrides, plus setters. Shared
 * across every caller (see the store above), so a remap from Settings is
 * immediately visible to Workspace's shortcut dispatcher. */
export function useShortcuts() {
  const overrides = useSyncExternalStore(subscribe, getSnapshot)

  const setShortcut = (id: string, key: string) => {
    commit({ ...overridesCache, [id]: key })
  }
  const resetShortcut = (id: string) => {
    if (!(id in overridesCache)) return
    const next = { ...overridesCache }
    delete next[id]
    commit(next)
  }

  return { overrides, setShortcut, resetShortcut }
}

export function effectiveKey(action: ShortcutAction, overrides: Record<string, string>): string {
  return overrides[action.id] || action.defaultKey
}

/** Normalizes a keydown into the same "Mod-Shift-b" format as defaultKey.
 * Returns null for a bare modifier press (not a complete combo yet) or a
 * combo missing Ctrl/Cmd -- shortcuts are required to include Mod so a
 * remap can never collide with plain typing in the editor. */
export function keyComboFromEvent(e: KeyboardEvent | React.KeyboardEvent): string | null {
  const mod = isMac ? e.metaKey : e.ctrlKey
  if (!mod || MODIFIER_KEYS.has(e.key)) return null
  const parts: string[] = ['Mod']
  if (e.shiftKey) parts.push('Shift')
  if (e.altKey) parts.push('Alt')
  parts.push(e.key.length === 1 ? e.key.toUpperCase() : e.key)
  return parts.join('-')
}

/** Whether a native keydown event matches a stored "Mod-Shift-b" combo. */
export function matchesCombo(e: KeyboardEvent, combo: string): boolean {
  const parts = combo.split('-')
  const key = parts[parts.length - 1]
  const wantShift = parts.includes('Shift')
  const wantAlt = parts.includes('Alt')
  const mod = isMac ? e.metaKey : e.ctrlKey
  if (!mod || e.shiftKey !== wantShift || e.altKey !== wantAlt) return false
  const eKey = e.key.length === 1 ? e.key.toUpperCase() : e.key
  return eKey === key
}

/** Human-readable form for display ("Ctrl+Shift+B" / "⌘⇧B"). */
export function describeCombo(combo: string): string {
  const parts = combo.split('-').map((p) => {
    if (p === 'Mod') return isMac ? '⌘' : 'Ctrl'
    if (p === 'Shift') return isMac ? '⇧' : 'Shift'
    if (p === 'Alt') return isMac ? '⌥' : 'Alt'
    return p
  })
  return parts.join(isMac ? '' : '+')
}
