import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BodyView from './BodyView.vue'
import type { Detail } from '@/types/api'

function detailWithBody(body: string): Detail {
  return {
    id: 1, time: 't', remoteAddr: 'x', tls: false, method: 'POST', target: '/', proto: 'HTTP/1.1', name: '',
    headerCount: 0, bodySize: body.length, rawSize: 0, requestLine: 'POST / HTTP/1.1',
    headers: [], rawBase64: '', bodyBase64: btoa(body),
  }
}

describe('BodyView', () => {
  it('JSON body 默认可切 JSON 视图并高亮', () => {
    const w = mount(BodyView, { props: { detail: detailWithBody('{"a":1}') } })
    const jsonBtn = w.findAll('.body-view-btn').find((b) => b.text() === 'JSON')!
    expect(jsonBtn.attributes('disabled')).toBeUndefined()
  })
  it('非 JSON body 时 JSON 按钮禁用', () => {
    const w = mount(BodyView, { props: { detail: detailWithBody('hello') } })
    const jsonBtn = w.findAll('.body-view-btn').find((b) => b.text() === 'JSON')!
    expect(jsonBtn.attributes('disabled')).toBeDefined()
  })
})
