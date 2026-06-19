import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import HeadersView from './HeadersView.vue'
import type { Detail } from '@/types/api'

const detail: Detail = {
  id: 1, time: 't', remoteAddr: 'x', tls: false, method: 'GET', target: '/', proto: 'HTTP/1.1',
  headerCount: 3, bodySize: 0, rawSize: 0, requestLine: 'GET / HTTP/1.1',
  headers: [
    { name: 'Host', value: 'a' },
    { name: 'X-Dup', value: '1' },
    { name: 'x-dup', value: '2' },
  ],
  rawBase64: '', bodyBase64: '',
}

describe('HeadersView', () => {
  it('保留原始大小写、按序、重复名标 DUP', () => {
    const w = mount(HeadersView, { props: { detail } })
    const text = w.text()
    expect(text).toContain('Host')
    expect(text).toContain('X-Dup')
    expect(text).toContain('x-dup') // 原始大小写不被合并
    expect(w.findAll('.dupe')).toHaveLength(1) // 第二个同名（忽略大小写）标 DUP
  })
  it('首行展示请求行', () => {
    const w = mount(HeadersView, { props: { detail } })
    expect(w.find('.reqline').text()).toContain('GET / HTTP/1.1')
  })
})
