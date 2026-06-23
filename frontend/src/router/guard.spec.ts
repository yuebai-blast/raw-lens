import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { setupGuard } from './index'

const Blank = { template: '<div />' }

function makeRouter() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'home', component: Blank },
      { path: '/login', name: 'login', component: Blank },
    ],
  })
  setupGuard(router)
  return router
}

describe('路由守卫', () => {
  beforeEach(() => setActivePinia(createPinia()))
  afterEach(() => vi.unstubAllGlobals())

  it('开启鉴权且未登录时访问 / 跳转 /login', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, status: 200, json: async () => ({ enabled: true, authenticated: false }),
    }))
    const router = makeRouter()
    await router.push('/')
    await router.isReady()
    expect(router.currentRoute.value.name).toBe('login')
  })

  it('关闭鉴权时访问 / 不跳转', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, status: 200, json: async () => ({ enabled: false, authenticated: true }),
    }))
    const router = makeRouter()
    await router.push('/')
    await router.isReady()
    expect(router.currentRoute.value.name).toBe('home')
  })

  it('已登录访问 /login 跳回首页', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true, status: 200, json: async () => ({ enabled: true, authenticated: true }),
    }))
    const router = makeRouter()
    await router.push('/login')
    await router.isReady()
    expect(router.currentRoute.value.name).toBe('home')
  })
})
