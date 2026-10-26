// Where the UI runs. The Android app (apps/tauri, src/mobile.rs) bundles
// this UI and serves it with the chosen server's API through a local proxy.

/** Opening this path in the mobile app returns to its server picker; the
 *  shell intercepts the navigation, the path does not exist over HTTP. */
export const CHANGE_SERVER_PATH = '/__app/server'

type Env = { navigator: { userAgent: string }; __TAURI_INTERNALS__?: unknown }

export function isMobileApp(win: Env = window): boolean {
  return '__TAURI_INTERNALS__' in win && /Android|iPhone|iPad/i.test(win.navigator.userAgent)
}
