import { defineStore } from 'pinia'
import type { SessionInfo } from '@/types/api'

interface State {
  enabled: boolean // 后端是否开启了登录鉴权
  authenticated: boolean // 当前是否已登录
  ready: boolean // 是否已向后端拉过一次会话状态
}

export const useAuthStore = defineStore('auth', {
  state: (): State => ({ enabled: false, authenticated: false, ready: false }),
  actions: {
    async fetchSession() {
      try {
        const res = await fetch('/api/session')
        if (!res.ok) throw new Error(`session ${res.status}`)
        const info = (await res.json()) as SessionInfo
        this.enabled = info.enabled
        this.authenticated = info.authenticated
      } catch {
        // 拉不到会话信息（后端不可达、或返回非2xx）时保守当作未开启鉴权，避免把人锁在登录页外用不了
        this.enabled = false
        this.authenticated = true
      }
      this.ready = true
    },
    async login(username: string, password: string): Promise<boolean> {
      try {
        const res = await fetch('/api/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password }),
        })
        if (!res.ok) {
          this.authenticated = false
          return false
        }
        this.authenticated = true
        return true
      } catch {
        this.authenticated = false
        return false
      }
    },
    async logout() {
      try {
        await fetch('/api/logout', { method: 'POST' })
      } finally {
        this.authenticated = false
      }
    },
  },
})
