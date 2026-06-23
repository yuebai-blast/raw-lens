import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ConfirmDialog from './ConfirmDialog.vue'

const base = { title: '删除记录', message: '确认删除 #abc？' }

describe('ConfirmDialog', () => {
  it('open=false 时不渲染对话框', () => {
    const w = mount(ConfirmDialog, { props: { open: false, ...base } })
    expect(w.find('.dialog').exists()).toBe(false)
  })

  it('open=true 时渲染标题与内容', () => {
    const w = mount(ConfirmDialog, { props: { open: true, ...base } })
    expect(w.find('.dialog').text()).toContain('删除记录')
    expect(w.find('.dialog').text()).toContain('确认删除 #abc？')
  })

  it('点确认按钮 emit confirm，点取消按钮 emit cancel', async () => {
    const w = mount(ConfirmDialog, { props: { open: true, ...base } })
    await w.find('.dlg-confirm').trigger('click')
    expect(w.emitted('confirm')).toHaveLength(1)
    await w.find('.dlg-cancel').trigger('click')
    expect(w.emitted('cancel')).toHaveLength(1)
  })

  it('点遮罩 emit cancel', async () => {
    const w = mount(ConfirmDialog, { props: { open: true, ...base } })
    await w.find('.dlg-overlay').trigger('click')
    expect(w.emitted('cancel')).toHaveLength(1)
  })
})
