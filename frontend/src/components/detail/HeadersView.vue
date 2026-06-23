<script setup lang="ts">
import { computed } from 'vue'
import type { Detail } from '@/types/api'
import { useCopy } from './useCopy'

const props = defineProps<{ detail: Detail }>()
const { copied, copy } = useCopy()

// 标记重复名（大小写不敏感），但展示保留原始大小写、不合并。
const rows = computed(() => {
  const seen = new Set<string>()
  return props.detail.headers.map((h) => {
    const key = h.name.toLowerCase()
    const dup = seen.has(key)
    seen.add(key)
    return { name: h.name, value: h.value, dup }
  })
})

function copyHeaders() {
  const text = [props.detail.requestLine, ...props.detail.headers.map((h) => `${h.name}: ${h.value}`)].join('\r\n')
  copy(text)
}
</script>

<template>
  <div class="hint">
    按收到顺序排列，header 名保留原始大小写，重复名不合并
    <button
      class="copy"
      @click="copyHeaders"
    >
      {{ copied ? 'COPIED' : 'COPY' }}
    </button>
  </div>
  <table class="htable">
    <tbody>
      <tr class="reqline">
        <td class="idx" /><td class="hname">
          ⟶
        </td><td class="hval">
          {{ detail.requestLine }}
        </td>
      </tr>
      <tr
        v-for="(r, i) in rows"
        :key="i"
      >
        <td class="idx">
          {{ i + 1 }}
        </td>
        <td class="hname">
          {{ r.name }}
        </td>
        <td class="hval">
          {{ r.value }}<span
            v-if="r.dup"
            class="dupe"
          >DUP·重复名</span>
        </td>
      </tr>
    </tbody>
  </table>
</template>

<style scoped>
/* headers 表格 */
/* display:block + overflow-y 让表格自身成为滚动容器（hint/COPY 固定在框外上方）；
   table 默认 display 不能可靠滚动，故改 block。 */
.htable { display: block; min-height: 0; overflow-y: auto; width: 100%; border-collapse: collapse; font-family: var(--mono); font-size: 12.5px; }
.htable tr { border-bottom: 1px solid var(--line-soft); }
.htable tr.reqline td { color: var(--phosphor); font-weight: 600; padding-top: 4px; }
.htable td { padding: 8px 12px; vertical-align: top; }
.htable td.idx { color: var(--muted); width: 1%; text-align: right; user-select: none; }
.htable td.hname { color: var(--amber); white-space: nowrap; width: 1%; font-weight: 500; }
.htable td.hval { color: var(--ink); word-break: break-all; }
.dupe { color: var(--violet); font-size: 10px; letter-spacing: 1px; margin-left: 8px; }
</style>
