import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useCaptureStore } from './captures'
import type { Summary } from '@/types/api'

function mockFetchOnce(json: unknown, ok = true) {
  return vi.fn().mockResolvedValue({ ok, json: async () => json, status: ok ? 200 : 500 })
}

const sample: Summary[] = [
  { id: 1, time: '2026-06-19T00:00:00Z', remoteAddr: '1.2.3.4:5', tls: false, method: 'GET', target: '/a', proto: 'HTTP/1.1', headerCount: 2, bodySize: 0, rawSize: 30 },
]

describe('useCaptureStore', () => {
  beforeEach(() => setActivePinia(createPinia()))

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
    const grown: Summary[] = [{ ...sample[0], id: 2 }, ...sample]
    vi.stubGlobal('fetch', mockFetchOnce(grown))
    await s.refresh()
    expect(s.newIds.has(2)).toBe(true)
    expect(s.newIds.has(1)).toBe(false)
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
