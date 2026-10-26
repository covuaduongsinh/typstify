import { useSyncExternalStore } from 'react'

/** Whether `query` (a CSS media query) currently matches; re-renders on change. */
export function useMediaQuery(query: string): boolean {
  return useSyncExternalStore(
    (onChange) => {
      const mql = window.matchMedia(query)
      mql.addEventListener('change', onChange)
      return () => mql.removeEventListener('change', onChange)
    },
    () => window.matchMedia(query).matches,
    () => false,
  )
}

/** Phone-sized screens: the workspace shows one pane at a time. */
export const COMPACT_QUERY = '(max-width: 720px)'
