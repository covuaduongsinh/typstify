import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, api, PROJECT_CHANGED_EVENT, projectWsUrl, setActiveProject } from './client'

function mockFetch(status: number, body: unknown) {
  const fn = vi.fn(
    async (_input: RequestInfo | URL, _init?: RequestInit) =>
      new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } }),
  )
  vi.stubGlobal('fetch', fn)
  return fn
}

// Tests run under Node: a minimal window (events and location).
beforeEach(() => {
  vi.stubGlobal('window', Object.assign(new EventTarget(), { location: { protocol: 'http:', host: 'localhost:8080' } }))
})

afterEach(() => {
  setActiveProject(null)
  vi.unstubAllGlobals()
})

describe('project of the tab', () => {
  it('names the open project on every request', async () => {
    const fetch = mockFetch(200, { ok: true })
    setActiveProject('D:/Sách cờ/Tập 1')
    await api.get('/api/workspace/tree')
    const init = fetch.mock.calls[0][1]
    expect(new Headers(init?.headers).get('X-Typstify-Project')).toBe(encodeURIComponent('D:/Sách cờ/Tập 1'))
  })

  it('sends no project header outside a project', async () => {
    const fetch = mockFetch(200, [])
    await api.get('/api/projects')
    const init = fetch.mock.calls[0][1]
    expect(new Headers(init?.headers).has('X-Typstify-Project')).toBe(false)
  })

  it('keeps an explicit content type', async () => {
    const fetch = mockFetch(200, {})
    await api.postBinary('/api/settings/fonts', new Blob(['x']))
    const init = fetch.mock.calls[0][1]
    expect(new Headers(init?.headers).get('Content-Type')).toBe('application/octet-stream')
  })

  it('announces a project changed in another tab', async () => {
    mockFetch(409, { error: 'Dự án đã được đổi ở một tab khác', code: 'project_changed' })
    const seen = vi.fn()
    window.addEventListener(PROJECT_CHANGED_EVENT, seen)
    await expect(api.get('/api/workspace/tree')).rejects.toBeInstanceOf(ApiError)
    window.removeEventListener(PROJECT_CHANGED_EVENT, seen)
    expect(seen).toHaveBeenCalledOnce()
  })

  it('does not announce other conflicts', async () => {
    mockFetch(409, { error: 'no project is open' })
    const seen = vi.fn()
    window.addEventListener(PROJECT_CHANGED_EVENT, seen)
    await expect(api.get('/api/workspace/tree')).rejects.toBeInstanceOf(ApiError)
    window.removeEventListener(PROJECT_CHANGED_EVENT, seen)
    expect(seen).not.toHaveBeenCalled()
  })

  it('puts the project in WebSocket URLs', () => {
    setActiveProject('D:/X')
    expect(projectWsUrl('/ws/lsp')).toMatch(/\/ws\/lsp\?project=D%3A%2FX$/)
    expect(projectWsUrl('/ws/agent?sessionId=1')).toMatch(/\/ws\/agent\?sessionId=1&project=D%3A%2FX$/)
    setActiveProject(null)
    expect(projectWsUrl('/ws/lsp')).toMatch(/\/ws\/lsp$/)
  })
})
