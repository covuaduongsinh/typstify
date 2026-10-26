import { useEffect, useState, useSyncExternalStore } from 'react'

// Service worker registration (src/sw/sw.ts, built to /sw.js) and the
// state the PWA banners show: an update waiting, or one another tab has
// already switched to, and the network status.

export type UpdateState = 'none' | 'waiting' | 'activated-elsewhere'

let waiting: ServiceWorker | null = null
let state: UpdateState = 'none'
let reloading = false
const listeners = new Set<() => void>()
const set = (next: UpdateState) => {
  state = next
  listeners.forEach((l) => l())
}

function watch(reg: ServiceWorkerRegistration) {
  const track = (sw: ServiceWorker | null) => {
    if (!sw) return
    const check = () => {
      // "installed" with a controller already present: an update, not the
      // first install.
      if (sw.state === 'installed' && navigator.serviceWorker.controller) {
        waiting = sw
        set('waiting')
      } else if ((sw.state === 'activated' || sw.state === 'redundant') && waiting === sw) {
        // Replaced by a newer build, or activated (by another tab): the
        // controllerchange listener below takes it from here.
        waiting = null
        if (state === 'waiting') set('none')
      }
    }
    sw.addEventListener('statechange', check)
    check()
  }
  track(reg.waiting)
  track(reg.installing)
  reg.addEventListener('updatefound', () => track(reg.installing))
}

/** Registers /sw.js in production builds. Checks for a new version when
 * the tab becomes visible again (long-lived editor tabs). */
export function registerServiceWorker() {
  if (!import.meta.env.PROD || !('serviceWorker' in navigator)) return
  const hadController = !!navigator.serviceWorker.controller
  navigator.serviceWorker.addEventListener('controllerchange', () => {
    if (reloading) {
      window.location.reload()
      return
    }
    // A new version took over (another tab chose to reload). This tab still
    // runs the old code; the worker keeps the previous version's files, but
    // the user should reload when convenient. The first install is not news.
    if (hadController) set('activated-elsewhere')
  })
  window.addEventListener('load', () => {
    navigator.serviceWorker
      .register('/sw.js')
      .then((reg) => {
        watch(reg)
        document.addEventListener('visibilitychange', () => {
          if (document.visibilityState === 'visible') void reg.update().catch(() => undefined)
        })
      })
      .catch(() => undefined) // no offline support; the app still works
  })
}

/** Switches to the new version and reloads (the page's own unsaved-changes
 * prompt still applies to the reload). */
export function applyUpdate() {
  if (state === 'waiting' && waiting && waiting.state === 'installed') {
    reloading = true
    waiting.postMessage('SKIP_WAITING')
    return
  }
  window.location.reload()
}

export function useUpdateState(): UpdateState {
  return useSyncExternalStore(
    (l) => {
      listeners.add(l)
      return () => listeners.delete(l)
    },
    () => state,
  )
}

export function useOnline(): boolean {
  const [online, setOnline] = useState(() => navigator.onLine)
  useEffect(() => {
    const on = () => setOnline(true)
    const off = () => setOnline(false)
    window.addEventListener('online', on)
    window.addEventListener('offline', off)
    return () => {
      window.removeEventListener('online', on)
      window.removeEventListener('offline', off)
    }
  }, [])
  return online
}
