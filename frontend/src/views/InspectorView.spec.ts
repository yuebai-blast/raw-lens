import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import InspectorView from './InspectorView.vue'
import { useCaptureStore } from '@/stores/captures'

// 轮询生命周期应归属真正展示抓包列表的 InspectorView：
// 这样登录成功跳转到本视图时会重新开始轮询（修复「首次登录后看不到历史、再刷新才出现」）。
describe('InspectorView 轮询生命周期', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => [] }))
  })
  afterEach(() => vi.unstubAllGlobals())

  it('挂载时开始轮询、卸载时停止轮询', () => {
    const store = useCaptureStore()
    const start = vi.spyOn(store, 'startPolling').mockImplementation(() => {})
    const stop = vi.spyOn(store, 'stopPolling').mockImplementation(() => {})
    const w = mount(InspectorView, {
      global: { stubs: { SignalLog: true, RequestDetail: true } },
    })
    expect(start).toHaveBeenCalled()
    w.unmount()
    expect(stop).toHaveBeenCalled()
  })
})
