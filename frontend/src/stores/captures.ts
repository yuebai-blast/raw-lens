import { defineStore } from 'pinia'
import type { Summary, Detail } from '@/types/api'
import router from '@/router'
import { useAuthStore } from '@/stores/auth'

const POLL_INTERVAL = 1500

interface State {
  list: Summary[]
  current: Detail | null
  activeId: number | null
  status: 'CAPTURING' | 'OFFLINE'
  newIds: Set<number>
  knownIds: Set<number>
  firstLoad: boolean
  timer: number | null
}

export const useCaptureStore = defineStore('captures', {
  state: (): State => ({
    list: [],
    current: null,
    activeId: null,
    status: 'CAPTURING',
    newIds: new Set(),
    knownIds: new Set(),
    firstLoad: true,
    timer: null,
  }),
  actions: {
    async refresh() {
      let items: Summary[]
      try {
        const res = await fetch('/api/requests')
        if (res.status === 401) {
          // 会话过期/被登出：停轮询、标记未登录并回登录页
          this.stopPolling()
          useAuthStore().authenticated = false
          void router.push({ name: 'login' })
          return
        }
        if (!res.ok) throw new Error(String(res.status))
        items = (await res.json()) as Summary[]
        this.status = 'CAPTURING'
      } catch {
        this.status = 'OFFLINE'
        return
      }
      const ids = new Set(items.map((i) => i.id))
      this.newIds = this.firstLoad
        ? new Set()
        : new Set(items.filter((i) => !this.knownIds.has(i.id)).map((i) => i.id))
      this.knownIds = ids
      this.firstLoad = false
      this.list = items
    },
    async fetchDetail(id: number) {
      this.activeId = id
      try {
        const res = await fetch('/api/requests/' + id)
        if (!res.ok) {
          this.current = null
          return
        }
        this.current = (await res.json()) as Detail
      } catch {
        // 与 refresh 一致：网络异常时吞掉，置空详情而非向上抛出。
        this.current = null
      }
    },
    async clear() {
      await fetch('/api/clear', { method: 'POST' })
      this.list = []
      this.current = null
      this.activeId = null
      this.newIds = new Set()
      this.knownIds = new Set()
      this.firstLoad = true
    },
    startPolling() {
      if (this.timer !== null) return
      void this.refresh()
      this.timer = window.setInterval(() => void this.refresh(), POLL_INTERVAL)
    },
    stopPolling() {
      if (this.timer !== null) {
        window.clearInterval(this.timer)
        this.timer = null
      }
    },
  },
})
