import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import LogItem from './LogItem.vue'
import type { Summary } from '@/types/api'

const item: Summary = {
  id: 7, time: '2026-06-19T01:02:03Z', remoteAddr: 'x', tls: true,
  method: 'POST', target: '/submit', proto: 'HTTP/1.1', headerCount: 3, bodySize: 0, rawSize: 2048,
}

describe('LogItem', () => {
  it('渲染 method、target、#id，TLS 时显示锁', () => {
    const w = mount(LogItem, { props: { item, active: false, isNew: false } })
    expect(w.text()).toContain('POST')
    expect(w.text()).toContain('/submit')
    expect(w.text()).toContain('#7')
    expect(w.find('.lock').exists()).toBe(true)
  })
  it('点击 emit select 带 id', async () => {
    const w = mount(LogItem, { props: { item, active: false, isNew: false } })
    await w.trigger('click')
    expect(w.emitted('select')?.[0]).toEqual([7])
  })
})
