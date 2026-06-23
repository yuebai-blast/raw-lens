import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from './auth'

function mockFetch(json: unknown, ok = true) {
  return vi.fn().mockResolvedValue({ ok, json: async () => json, status: ok ? 200 : 401 })
}

describe('useAuthStore', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('fetchSession 填充 enabled/authenticated 并置 ready', async () => {
    vi.stubGlobal('fetch', mockFetch({ enabled: true, authenticated: false }))
    const s = useAuthStore()
    await s.fetchSession()
    expect(s.enabled).toBe(true)
    expect(s.authenticated).toBe(false)
    expect(s.ready).toBe(true)
  })

  it('fetchSession 网络异常时保守置为免鉴权且 ready', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('down')))
    const s = useAuthStore()
    await s.fetchSession()
    expect(s.enabled).toBe(false)
    expect(s.authenticated).toBe(true)
    expect(s.ready).toBe(true)
  })

  it('fetchSession 后端返回非2xx时保守降级为免鉴权且 ready', async () => {
    vi.stubGlobal('fetch', mockFetch({ enabled: true, authenticated: true }, false))
    const s = useAuthStore()
    await s.fetchSession()
    expect(s.enabled).toBe(false)
    expect(s.authenticated).toBe(true)
    expect(s.ready).toBe(true)
  })

  it('login 成功置 authenticated 并返回 true', async () => {
    vi.stubGlobal('fetch', mockFetch({ authenticated: true }))
    const s = useAuthStore()
    const ok = await s.login('admin', 'secret')
    expect(ok).toBe(true)
    expect(s.authenticated).toBe(true)
  })

  it('login 失败（401）返回 false 且 authenticated 为 false', async () => {
    vi.stubGlobal('fetch', mockFetch({ authenticated: false }, false))
    const s = useAuthStore()
    const ok = await s.login('admin', 'wrong')
    expect(ok).toBe(false)
    expect(s.authenticated).toBe(false)
  })

  it('logout 后 authenticated 为 false', async () => {
    vi.stubGlobal('fetch', mockFetch(null))
    const s = useAuthStore()
    s.authenticated = true
    await s.logout()
    expect(s.authenticated).toBe(false)
  })
})
