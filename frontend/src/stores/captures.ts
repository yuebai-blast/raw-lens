import { defineStore } from 'pinia'
import type { Summary, Detail, Meta } from '@/types/api'
import router from '@/router'
import { useAuthStore } from '@/stores/auth'

const POLL_INTERVAL = 1500

interface State {
  list: Summary[]
  current: Detail | null
  activeId: string | null
  status: 'CAPTURING' | 'OFFLINE'
  newIds: Set<string>
  knownIds: Set<string>
  firstLoad: boolean
  timer: number | null
  captureUrl: string
  showLockedOnly: boolean
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
    captureUrl: '',
    showLockedOnly: false,
  }),
  getters: {
    // visibleList 是按「只看锁定」开关过滤后的列表，供视图渲染。
    visibleList(state): Summary[] {
      return state.showLockedOnly ? state.list.filter((i) => i.locked) : state.list
    },
  },
  actions: {
    // handleUnauthorized 是 401 响应的共享处理逻辑：停轮询、标记未登录、跳回登录页。
    handleUnauthorized() {
      this.stopPolling()
      useAuthStore().authenticated = false
      void router.push({ name: 'login' })
    },
    async refresh() {
      let items: Summary[]
      try {
        const res = await fetch('/api/requests')
        if (res.status === 401) {
          // 会话过期/被登出，复用共享处理逻辑
          this.handleUnauthorized()
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
    async fetchDetail(id: string) {
      this.activeId = id
      try {
        const res = await fetch('/api/requests/' + id)
        if (res.status === 401) {
          // 会话过期：复用共享未登录处理逻辑，不做静默置空
          this.handleUnauthorized()
          return
        }
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
    // setName 给某条记录设置备注名，成功后同步更新本地 list 与 current。
    async setName(id: string, name: string) {
      try {
        const res = await fetch('/api/requests/' + id, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name }),
        })
        if (res.status === 401) {
          this.handleUnauthorized()
          return
        }
        if (!res.ok) return
      } catch {
        return
      }
      const item = this.list.find((i) => i.id === id)
      if (item) item.name = name
      if (this.current && this.current.id === id) this.current.name = name
    },
    // setLocked 切换某条记录的锁定状态，成功后同步更新本地 list 与 current。
    async setLocked(id: string, locked: boolean) {
      try {
        const res = await fetch('/api/requests/' + id, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ locked }),
        })
        if (res.status === 401) {
          this.handleUnauthorized()
          return
        }
        if (!res.ok) return
      } catch {
        return
      }
      const item = this.list.find((i) => i.id === id)
      if (item) item.locked = locked
      if (this.current && this.current.id === id) this.current.locked = locked
    },
    // remove 删除某条记录，成功后从 list 移除；若删的是当前项则清空详情并回到列表。
    async remove(id: string) {
      try {
        const res = await fetch('/api/requests/' + id, { method: 'DELETE' })
        if (res.status === 401) {
          this.handleUnauthorized()
          return
        }
        if (!res.ok) return
      } catch {
        return
      }
      this.list = this.list.filter((i) => i.id !== id)
      this.knownIds.delete(id)
      this.newIds.delete(id)
      if (this.activeId === id) {
        this.current = null
        this.activeId = null
        void router.push({ name: 'home' })
      }
    },
    async clear() {
      const res = await fetch('/api/clear', { method: 'POST' })
      if (res.status === 401) {
        this.handleUnauthorized()
        return
      }
      // 后端只清未锁定记录，前端同步：保留锁定项、其余清掉。
      this.list = this.list.filter((i) => i.locked)
      this.knownIds = new Set(this.list.map((i) => i.id))
      this.newIds = new Set()
      if (this.current && !this.current.locked) {
        this.current = null
        this.activeId = null
      }
      this.firstLoad = true
    },
    // fetchMeta 取面板元信息（抓包端口对外展示地址），失败静默——只是顶栏文案，不阻塞主流程。
    async fetchMeta() {
      try {
        const res = await fetch('/api/meta')
        if (!res.ok) return
        this.captureUrl = ((await res.json()) as Meta).captureUrl
      } catch {
        // 网络异常时吞掉，captureUrl 保持空、顶栏不展示该块。
      }
    },
    startPolling() {
      if (this.timer !== null) return
      void this.fetchMeta() // 一次性初始化，不进轮询
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
