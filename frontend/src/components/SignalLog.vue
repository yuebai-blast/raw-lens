<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useCaptureStore } from '@/stores/captures'
import { useConfirmStore } from '@/stores/confirm'
import LogItem from './LogItem.vue'

const store = useCaptureStore()
const confirm = useConfirmStore()
const router = useRouter()

// 列表新→旧：后端已按新在前返回，直接用。
function select(id: string) {
  void router.push({ name: 'detail', params: { id } })
}

// 删除前二次确认，避免误删；确认后调用 store.remove。
async function remove(id: string) {
  if (await confirm.confirm({ title: '删除记录', message: `确认删除记录 #${id}？` })) {
    await store.remove(id)
  }
}
</script>

<template>
  <aside
    class="log"
    aria-label="captured requests"
  >
    <div class="log-head">
      SIGNAL LOG
    </div>
    <div class="log-body">
      <LogItem
        v-for="it in store.list"
        :key="it.id"
        :item="it"
        :active="it.id === store.activeId"
        :is-new="store.newIds.has(it.id)"
        @select="select"
        @delete="remove"
      />
    </div>
  </aside>
</template>

<style scoped>
/* ---- 左侧信号日志栏 ---- */
.log {
  border-right: 1px solid var(--line);
  display: grid; grid-template-rows: auto 1fr; min-height: 0;
  background: linear-gradient(180deg, var(--panel), var(--bg));
}
.log-head {
  padding: 11px 18px; font-family: var(--mono); font-size: 10px;
  letter-spacing: 3px; color: var(--muted); border-bottom: 1px solid var(--line-soft);
}
.log-body { overflow-y: auto; }

@media (max-width: 760px) {
  .log { max-height: 38vh; border-right: none; border-bottom: 1px solid var(--line); }
}
</style>
