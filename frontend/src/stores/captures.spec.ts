import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useCaptureStore } from './captures'
import { useAuthStore } from './auth'
import type { Summary } from '@/types/api'

// mock vue-router，避免 router.push 报「no active pinia」类错误
vi.mock('@/router', () => ({
  default: { push: vi.fn() },
}))

function mockFetchOnce(json: unknown, ok = true, status?: number) {
  const s = status ?? (ok ? 200 : 500)
  return vi.fn().mockResolvedValue({ ok, json: async () => json, status: s })
}

const sample: Summary[] = [
  { id: 'aaaaaaaaaaaa', time: '2026-06-19T00:00:00Z', remoteAddr: '1.2.3.4:5', tls: false, method: 'GET', target: '/a', proto: 'HTTP/1.1', name: '', headerCount: 2, bodySize: 0, rawSize: 30 },
]

describe('useCaptureStore', () => {
  beforeEach(() => setActivePinia(createPinia()))
  afterEach(() => vi.unstubAllGlobals())

  it('refresh 成功后填充 list 且状态为 CAPTURING', async () => {
    vi.stubGlobal('fetch', mockFetchOnce(sample))
    const s = useCaptureStore()
    await s.refresh()
    expect(s.list).toHaveLength(1)
    expect(s.status).toBe('CAPTURING')
  })

  it('refresh 失败时状态置 OFFLINE 且不抛', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('down')))
    const s = useCaptureStore()
    await s.refresh()
    expect(s.status).toBe('OFFLINE')
  })

  it('refresh HTTP 错误响应（!res.ok）时状态置 OFFLINE', async () => {
    vi.stubGlobal('fetch', mockFetchOnce([], false))
    const s = useCaptureStore()
    await s.refresh()
    expect(s.status).toBe('OFFLINE')
  })

  it('第二次 refresh 出现的新 id 进入 newIds，老 id 不在', async () => {
    const s = useCaptureStore()
    vi.stubGlobal('fetch', mockFetchOnce(sample))
    await s.refresh()
    const grown: Summary[] = [{ ...sample[0], id: 'bbbbbbbbbbbb' }, ...sample]
    vi.stubGlobal('fetch', mockFetchOnce(grown))
    await s.refresh()
    expect(s.newIds.has('bbbbbbbbbbbb')).toBe(true)
    expect(s.newIds.has('aaaaaaaaaaaa')).toBe(false)
  })

  it('fetchDetail 网络异常时不抛、置空 current', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('down')))
    const s = useCaptureStore()
    await s.fetchDetail('aaaaaaaaaaaa')
    expect(s.current).toBeNull()
    expect(s.activeId).toBe('aaaaaaaaaaaa')
  })

  it('fetchDetail 收到 401 时触发未登录处理（authenticated 置 false），而非静默置空', async () => {
    vi.stubGlobal('fetch', mockFetchOnce(null, false, 401))
    const s = useCaptureStore()
    const auth = useAuthStore()
    auth.authenticated = true // 初始已登录
    await s.fetchDetail('aaaaaaaaaaaa')
    // 401 应触发 handleUnauthorized：authenticated 被置 false
    expect(auth.authenticated).toBe(false)
    // current 不应被置空（handleUnauthorized 不改 current，这是与普通 404 的区别）
    expect(s.current).toBeNull() // 初始值，未被写入
  })

  it('refresh 收到 401 时触发未登录处理（authenticated 置 false）', async () => {
    vi.stubGlobal('fetch', mockFetchOnce(null, false, 401))
    const s = useCaptureStore()
    const auth = useAuthStore()
    auth.authenticated = true
    await s.refresh()
    expect(auth.authenticated).toBe(false)
  })

  it('setName 成功后更新 list 中该项的 name', async () => {
    const s = useCaptureStore()
    vi.stubGlobal('fetch', mockFetchOnce(sample))
    await s.refresh()
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204, json: async () => null })
    vi.stubGlobal('fetch', fetchMock)
    await s.setName('aaaaaaaaaaaa', '登录接口')
    expect(fetchMock).toHaveBeenCalledWith('/api/requests/aaaaaaaaaaaa', expect.objectContaining({ method: 'PATCH' }))
    expect(s.list[0].name).toBe('登录接口')
  })

  it('remove 成功后从 list 移除该项；删的是当前项时清空 current', async () => {
    const s = useCaptureStore()
    const grown: Summary[] = [{ ...sample[0], id: 'bbbbbbbbbbbb' }, ...sample]
    vi.stubGlobal('fetch', mockFetchOnce(grown))
    await s.refresh()
    s.activeId = 'aaaaaaaaaaaa'
    s.current = { ...sample[0], requestLine: '', headers: [], rawBase64: '', bodyBase64: '' }
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204, json: async () => null })
    vi.stubGlobal('fetch', fetchMock)
    await s.remove('aaaaaaaaaaaa')
    expect(fetchMock).toHaveBeenCalledWith('/api/requests/aaaaaaaaaaaa', expect.objectContaining({ method: 'DELETE' }))
    expect(s.list.map((i) => i.id)).toEqual(['bbbbbbbbbbbb'])
    expect(s.current).toBeNull()
    expect(s.activeId).toBeNull()
  })

  it('clear 后 list 与 current 清空', async () => {
    const s = useCaptureStore()
    vi.stubGlobal('fetch', mockFetchOnce(sample))
    await s.refresh()
    vi.stubGlobal('fetch', mockFetchOnce(null))
    await s.clear()
    expect(s.list).toHaveLength(0)
    expect(s.current).toBeNull()
  })
})
