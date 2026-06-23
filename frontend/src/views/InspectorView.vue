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
/* 关键：必须显式给行设 minmax(0, 1fr)，把 console 这一行钉死成「父容器高度」。
   否则隐式行是 auto，会被长 body 撑高、溢出到 overflow:hidden 的 body 被裁掉，
   导致 .readout / .log 的 overflow-y:auto 永远不触发（主视图过长被截断、无法滚动）。 */
.console {
  display: grid;
  grid-template-columns: var(--rail) 1fr;
  grid-template-rows: minmax(0, 1fr);
  height: 100%;
  min-height: 0;
}

@media (max-width: 760px) {
  /* 窄屏改为上下两行：上 log（自身高度）下 readout（占满剩余、可滚动） */
  .console { grid-template-columns: 1fr; grid-template-rows: auto minmax(0, 1fr); }
}
</style>
