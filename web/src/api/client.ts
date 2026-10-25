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

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: 'include',
    headers: init?.body ? { 'Content-Type': 'application/json' } : undefined,
    ...init,
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
