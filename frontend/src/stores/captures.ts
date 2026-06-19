import { defineStore } from 'pinia'
import type { Summary, Detail } from '@/types/api'

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
      const res = await fetch('/api/requests/' + id)
      if (!res.ok) {
        this.current = null
        return
      }
      this.current = (await res.json()) as Detail
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
