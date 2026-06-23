<script setup lang="ts">
import { computed } from 'vue'
import type { Detail } from '@/types/api'
import { b64ToBytes, toHexLines } from '@/utils/bytes'

const props = defineProps<{ detail: Detail }>()
const lines = computed(() => toHexLines(b64ToBytes(props.detail.rawBase64)))
</script>

<template>
  <div class="hint">完整原始字节 · 十六进制视图</div>
  <pre class="hex"><span v-if="lines.length === 0" class="off">(空)</span><template v-else>{{ lines.join('\n') }}</template></pre>
</template>

<style scoped>
/* hex dump 显示区 */
pre.hex {
  min-height: 0; overflow-y: auto; /* 深色框内部滚动，hint 固定在框外上方 */
  font-family: var(--mono); font-size: 12px; line-height: 1.6; margin: 0;
  background: var(--panel-2); border: 1px solid var(--line); border-radius: 8px;
  padding: 16px 18px; overflow-x: auto;
}
.off { color: var(--muted); }
</style>
