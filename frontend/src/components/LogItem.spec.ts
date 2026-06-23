import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import LogItem from './LogItem.vue'
import type { Summary } from '@/types/api'

const item: Summary = {
  id: 'abc123def456', time: '2026-06-19T01:02:03Z', remoteAddr: 'x', tls: true,
  method: 'POST', target: '/submit', proto: 'HTTP/1.1', name: '', headerCount: 3, bodySize: 0, rawSize: 2048,
}

describe('LogItem', () => {
  it('渲染 method、target、#id，TLS 时显示锁', () => {
    const w = mount(LogItem, { props: { item, active: false, isNew: false } })
    expect(w.text()).toContain('POST')
    expect(w.text()).toContain('/submit')
    expect(w.text()).toContain('#abc123def456')
    expect(w.find('.lock').exists()).toBe(true)
  })
  it('点击 emit select 带 id', async () => {
    const w = mount(LogItem, { props: { item, active: false, isNew: false } })
    await w.trigger('click')
    expect(w.emitted('select')?.[0]).toEqual(['abc123def456'])
  })
  it('有 name 时展示名称，无 name 时不展示', () => {
    const named = mount(LogItem, { props: { item: { ...item, name: '登录接口' }, active: false, isNew: false } })
    expect(named.find('.item-name').text()).toContain('登录接口')
    const unnamed = mount(LogItem, { props: { item, active: false, isNew: false } })
    expect(unnamed.find('.item-name').exists()).toBe(false)
  })
  it('点删除按钮 emit delete 带 id，且不冒泡触发 select', async () => {
    const w = mount(LogItem, { props: { item, active: false, isNew: false } })
    await w.find('.item-del').trigger('click')
    expect(w.emitted('delete')?.[0]).toEqual(['abc123def456'])
    expect(w.emitted('select')).toBeUndefined()
  })
})
