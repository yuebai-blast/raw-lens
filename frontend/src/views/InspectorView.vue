<script setup lang="ts">
import { watch } from 'vue'
import { useCaptureStore } from '@/stores/captures'
import SignalLog from '@/components/SignalLog.vue'
import RequestDetail from '@/components/RequestDetail.vue'

const props = defineProps<{ id?: string }>()
const store = useCaptureStore()

watch(
  () => props.id,
  (id) => {
    if (id) void store.fetchDetail(Number(id))
    else {
      store.activeId = null
      store.current = null
    }
  },
  { immediate: true },
)
</script>

<template>
  <main class="console">
    <SignalLog />
    <RequestDetail />
  </main>
</template>

<style scoped>
/* ---- 双栏 console 布局 ---- */
.console { display: grid; grid-template-columns: var(--rail) 1fr; min-height: 0; }

@media (max-width: 760px) {
  .console { grid-template-columns: 1fr; }
}
</style>
