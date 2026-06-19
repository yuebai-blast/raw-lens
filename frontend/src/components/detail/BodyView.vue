<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Detail } from '@/types/api'
import { b64ToBytes, bytesToText } from '@/utils/bytes'
import { tryParseJSON, highlightTokens } from '@/utils/json'
import { useCopy } from './useCopy'

const props = defineProps<{ detail: Detail }>()
const { copied, copy } = useCopy()
const view = ref<'text' | 'json'>('json')

const bodyText = computed(() => bytesToText(b64ToBytes(props.detail.bodyBase64)))
const parsed = computed(() => tryParseJSON(bodyText.value))
const hasJSON = computed(() => parsed.value !== null)
const activeView = computed<'text' | 'json'>(() => (view.value === 'json' && hasJSON.value ? 'json' : 'text'))
const tokens = computed(() => (activeView.value === 'json' ? highlightTokens(parsed.value) : []))

function copyBody() {
  copy(activeView.value === 'json' ? JSON.stringify(parsed.value, null, 2) : bodyText.value)
}
</script>

<template>
  <div class="hint">
    {{ detail.bodySize }} 字节 body · {{ activeView.toUpperCase() }} 视图
    <div class="body-view-tabs">
      <button class="body-view-btn" :class="{ active: activeView === 'text' }" @click="view = 'text'">TEXT</button>
      <button
        class="body-view-btn"
        :class="{ active: activeView === 'json' }"
        :disabled="!hasJSON"
        :title="hasJSON ? '' : 'body 不是合法 JSON'"
        @click="hasJSON && (view = 'json')"
      >JSON</button>
    </div>
    <button class="copy" @click="copyBody">{{ copied ? 'COPIED' : 'COPY' }}</button>
  </div>
  <pre class="wire" :class="{ 'json-wire': activeView === 'json' }"><template v-if="activeView === 'json'"><span
      v-for="(t, i) in tokens" :key="i" :class="t.cls">{{ t.text }}</span></template><template v-else-if="bodyText">{{ bodyText }}</template><span v-else class="empty-body">(无 body)</span></pre>
</template>

<style scoped>
/* body 视图切换按钮 */
.body-view-tabs { margin-left: auto; display: inline-flex; gap: 6px; }
.body-view-btn {
  font-family: var(--mono); font-size: 10px; letter-spacing: 1px;
  color: var(--ink-dim); background: #0a0f0e;
  border: 1px solid var(--line); border-radius: 4px;
  padding: 3px 10px; cursor: pointer;
}
.body-view-btn:hover:not(:disabled) { color: var(--ink); border-color: #2f4540; }
.body-view-btn.active { color: var(--phosphor); border-color: #264438; background: #0f1815; }
.body-view-btn:disabled { color: var(--muted); cursor: not-allowed; opacity: .7; }

/* 原始字节显示区 */
pre.wire {
  font-family: var(--mono); font-size: 12.5px; line-height: 1.65;
  white-space: pre-wrap; word-break: break-all; margin: 0;
  background:
    linear-gradient(180deg, var(--panel-2), var(--panel));
  border: 1px solid var(--line); border-radius: 8px;
  padding: 18px 20px; color: var(--ink);
  box-shadow: inset 0 0 30px #00000055;
}
.json-wire {
  tab-size: 2;
  word-break: normal;
  overflow-x: auto;
  background:
    linear-gradient(90deg, #34e0a10d 1px, transparent 1px) 0 0 / 24px 24px,
    linear-gradient(180deg, #101a17, #0b100f);
}
</style>
