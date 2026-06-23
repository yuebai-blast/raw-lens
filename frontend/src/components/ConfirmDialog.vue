<script setup lang="ts">
// 纯展示型确认弹框：受控于 props，交互结果通过 confirm/cancel 事件抛出。
// 挂在 App.vue 根部，position:fixed 覆盖全屏，无需 Teleport。
const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    message: string
    confirmText?: string
    cancelText?: string
  }>(),
  { confirmText: '确认', cancelText: '取消' },
)
const emit = defineEmits<{ confirm: []; cancel: [] }>()

// 点遮罩本身（而非弹框内部）才算取消。
function onOverlay(e: MouseEvent) {
  if (e.target === e.currentTarget) emit('cancel')
}
</script>

<template>
  <div
    v-if="props.open"
    class="dlg-overlay"
    @click="onOverlay"
    @keydown.esc="emit('cancel')"
  >
    <div
      class="dialog"
      role="alertdialog"
      aria-modal="true"
      tabindex="-1"
      @keydown.esc="emit('cancel')"
      @keydown.enter="emit('confirm')"
    >
      <div class="dlg-title">
        {{ props.title }}
      </div>
      <div class="dlg-message">
        {{ props.message }}
      </div>
      <div class="dlg-actions">
        <button
          class="dlg-btn dlg-cancel"
          @click="emit('cancel')"
        >
          {{ props.cancelText }}
        </button>
        <button
          class="dlg-btn dlg-confirm"
          @click="emit('confirm')"
        >
          {{ props.confirmText }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dlg-overlay {
  position: fixed; inset: 0; z-index: 100;
  display: flex; align-items: center; justify-content: center;
  background: rgba(2, 6, 5, 0.66); backdrop-filter: blur(2px);
}
.dialog {
  width: min(92vw, 380px);
  background: linear-gradient(180deg, var(--panel-2), var(--panel));
  border: 1px solid var(--line); border-radius: 8px;
  box-shadow: 0 18px 50px rgba(0, 0, 0, 0.5), 0 0 0 1px var(--line-soft);
  padding: 20px 22px 18px; outline: none;
}
.dlg-title {
  font-family: var(--mono); font-size: 13px; letter-spacing: 1.5px;
  color: var(--ink); margin-bottom: 10px;
}
.dlg-message {
  font-family: var(--mono); font-size: 12px; line-height: 1.7;
  color: var(--ink-dim); margin-bottom: 20px; word-break: break-all;
}
.dlg-actions { display: flex; justify-content: flex-end; gap: 10px; }
.dlg-btn {
  font-family: var(--mono); font-size: 11px; letter-spacing: 1.5px;
  padding: 7px 16px; border-radius: 5px; cursor: pointer;
  background: #0a0f0e; border: 1px solid var(--line);
  color: var(--ink-dim); transition: all .15s;
}
.dlg-cancel:hover { color: var(--ink); border-color: var(--line-soft); }
.dlg-confirm { color: var(--red); border-color: #4a2a2a; }
.dlg-confirm:hover { background: #1a0f0f; box-shadow: 0 0 12px #ef6f6c22; }
</style>
