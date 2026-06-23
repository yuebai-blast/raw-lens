import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useConfirmStore } from './confirm'

describe('useConfirmStore', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('confirm 打开弹框并记录标题/内容，返回 Promise', () => {
    const s = useConfirmStore()
    expect(s.open).toBe(false)
    const p = s.confirm({ title: '删除记录', message: '确认删除？' })
    expect(s.open).toBe(true)
    expect(s.title).toBe('删除记录')
    expect(s.message).toBe('确认删除？')
    expect(p).toBeInstanceOf(Promise)
  })

  it('accept 使 Promise resolve 为 true 并关闭', async () => {
    const s = useConfirmStore()
    const p = s.confirm({ title: 't', message: 'm' })
    s.accept()
    await expect(p).resolves.toBe(true)
    expect(s.open).toBe(false)
  })

  it('cancel 使 Promise resolve 为 false 并关闭', async () => {
    const s = useConfirmStore()
    const p = s.confirm({ title: 't', message: 'm' })
    s.cancel()
    await expect(p).resolves.toBe(false)
    expect(s.open).toBe(false)
  })
})
