import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import Masthead from './Masthead.vue'
import { useCaptureStore } from '@/stores/captures'

// captures store 会 import '@/router'（内部 createRouter），mock 掉避免拉起真实路由
vi.mock('@/router', () => ({ default: { push: vi.fn() } }))
// Masthead 内部 useRouter，保留 vue-router 其余导出，仅替换 useRouter
vi.mock('vue-router', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-router')>()),
  useRouter: () => ({ push: vi.fn() }),
}))

describe('Masthead', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('captureUrl 为空时不渲染监听地址块', () => {
    const w = mount(Masthead)
    expect(w.find('.gauge-listen').exists()).toBe(false)
  })

  it('有 captureUrl 时渲染监听地址，点击复制到剪贴板', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    const store = useCaptureStore()
    store.captureUrl = 'https://xxx.xx.com:9100'
    const w = mount(Masthead)
    const btn = w.find('.gauge-listen')
    expect(btn.exists()).toBe(true)
    expect(btn.text()).toContain('https://xxx.xx.com:9100')
    await btn.trigger('click')
    expect(writeText).toHaveBeenCalledWith('https://xxx.xx.com:9100')
    vi.unstubAllGlobals()
  })
})
