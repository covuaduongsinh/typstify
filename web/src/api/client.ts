// Thin fetch wrapper for the Go backend's REST API (see server/*.go). All
// requests carry the session cookie (credentials: "include") so the same
// mechanism works whether the frontend is served by the Go backend itself
// (production) or by the Vite dev server proxying to it (development).

export class ApiError extends Error {
  status: number
  /** Parsed JSON error body, when the server sent one (e.g. {mfaRequired: true}). */
  body?: Record<string, unknown>
  constructor(status: number, message: string, body?: Record<string, unknown>) {
    super(message)
    this.status = status
    this.body = body
  }
}

// The project this tab has open. Every request names it, so that when
// another tab of the same browser (one session) opens a different project
// the server answers 409 "project_changed" instead of this tab silently
// working on the other one (docs/plans/plan_phase3b_per_user_projects.md).
let activeProject: string | null = null

export function setActiveProject(path: string | null) {
  activeProject = path
}

/** Fired on window when the server says another tab changed the project. */
export const PROJECT_CHANGED_EVENT = 'typstify:project-changed'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  if (init?.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
  if (activeProject) headers.set('X-Typstify-Project', encodeURIComponent(activeProject))
  const res = await fetch(path, {
    credentials: 'include',
    ...init,
    headers,
  })

  if (!res.ok) {
    let message = res.statusText
    let body: Record<string, unknown> | undefined
    try {
      body = await res.json()
      if (typeof body?.error === 'string') message = body.error
    } catch {
      // ignore non-JSON error bodies
    }
    if (res.status === 409 && body?.code === 'project_changed') {
      window.dispatchEvent(new Event(PROJECT_CHANGED_EVENT))
    }
    throw new ApiError(res.status, message, body)
  }

  const contentType = res.headers.get('Content-Type') ?? ''
  if (contentType.includes('application/json')) {
    return (await res.json()) as T
  }
  return (await res.text()) as unknown as T
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'POST', body: body !== undefined ? JSON.stringify(body) : undefined }),
  put: <T>(path: string, body: string) => request<T>(path, { method: 'PUT', body }),
  putJson: <T>(path: string, body: unknown) =>
    request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
  /** Uploads a raw binary body (not multipart/form-data) -- the server reads
   * it as-is (e.g. POST /api/settings/fonts). */
  postBinary: <T>(path: string, body: Blob) =>
    request<T>(path, { method: 'POST', body, headers: { 'Content-Type': 'application/octet-stream' } }),
  del: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
}

export function wsUrl(path: string): string {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${window.location.host}${path}`
}

/** wsUrl for a project WebSocket: names the tab's project (?project=),
 * since a WebSocket cannot carry the X-Typstify-Project header. */
export function projectWsUrl(path: string): string {
  if (!activeProject) return wsUrl(path)
  const sep = path.includes('?') ? '&' : '?'
  return wsUrl(`${path}${sep}project=${encodeURIComponent(activeProject)}`)
}
