import { defineStore } from 'pinia'

// 通用确认弹框的选项。
export interface ConfirmOptions {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
}

interface State extends ConfirmOptions {
  open: boolean
  resolver: ((ok: boolean) => void) | null
}

// useConfirmStore 提供 Promise 化的全局确认：调用 confirm() 弹框并 await 用户选择。
// 由 App.vue 挂载的单个 <ConfirmDialog> 绑定本 store 的状态来呈现。
export const useConfirmStore = defineStore('confirm', {
  state: (): State => ({
    open: false,
    title: '',
    message: '',
    confirmText: '确认',
    cancelText: '取消',
    resolver: null,
  }),
  actions: {
    confirm(opts: ConfirmOptions): Promise<boolean> {
      this.title = opts.title
      this.message = opts.message
      this.confirmText = opts.confirmText ?? '确认'
      this.cancelText = opts.cancelText ?? '取消'
      this.open = true
      return new Promise<boolean>((resolve) => {
        this.resolver = resolve
      })
    },
    // settle 收口：关闭弹框并以 ok 兑现 Promise（accept/cancel 共用）。
    settle(ok: boolean) {
      this.open = false
      const r = this.resolver
      this.resolver = null
      r?.(ok)
    },
    accept() {
      this.settle(true)
    },
    cancel() {
      this.settle(false)
    },
  },
})
