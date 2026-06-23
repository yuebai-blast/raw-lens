<script setup lang="ts">
import { watch, onMounted, onUnmounted } from 'vue'
import { useCaptureStore } from '@/stores/captures'
import SignalLog from '@/components/SignalLog.vue'
import RequestDetail from '@/components/RequestDetail.vue'

const props = defineProps<{ id?: string }>()
const store = useCaptureStore()

// 轮询归属本视图：登录成功跳转到这里时会重新开始轮询，避免轮询被绑死在只挂载一次的
// App.vue 上（那样首次登录后不会再触发刷新，要手动刷新页面才看得到历史数据）。
onMounted(() => store.startPolling())
onUnmounted(() => store.stopPolling())

watch(
  () => props.id,
  (id) => {
    if (id) void store.fetchDetail(id)
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
