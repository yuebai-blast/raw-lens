<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Detail } from '@/types/api'
import { b64ToBytes, bytesToText } from '@/utils/bytes'
import { markCRLFSegments } from '@/utils/crlf'
import { useCopy } from './useCopy'

const props = defineProps<{ detail: Detail }>()
const { copied, copy } = useCopy()
const showCRLF = ref(false)

const text = computed(() => bytesToText(b64ToBytes(props.detail.rawBase64)))
const segments = computed(() => markCRLFSegments(text.value))
</script>

<template>
  <div class="hint">
    连接上读到的原始字节，顺序 / 大小写 / 空白完全保真
    <label class="toggle"><input type="checkbox" v-model="showCRLF" />显示换行符</label>
    <button class="copy" @click="copy(text)">{{ copied ? 'COPIED' : 'COPY' }}</button>
  </div>
  <pre class="wire"><template v-if="showCRLF"><template v-for="(s, i) in segments" :key="i"><span
        v-if="s.crlf" class="crlf">{{ s.text === '\r\n' ? '␍␊' : '␊' }}</span><template v-else>{{ s.text }}</template><template v-if="s.crlf">
</template></template></template><template v-else>{{ text }}</template></pre>
</template>

<style scoped>
/* CRLF 显示开关 */
.toggle { display: inline-flex; align-items: center; gap: 6px; cursor: pointer; color: var(--ink-dim); user-select: none; }
.toggle input { accent-color: var(--phosphor); }
/* 原始字节显示区 */
pre.wire {
  min-height: 0; overflow-y: auto; /* 深色框内部滚动，hint/COPY 固定在框外上方 */
  font-family: var(--mono); font-size: 12.5px; line-height: 1.65;
  white-space: pre-wrap; word-break: break-all; margin: 0;
  background:
    linear-gradient(180deg, var(--panel-2), var(--panel));
  border: 1px solid var(--line); border-radius: 8px;
  padding: 18px 20px; color: var(--ink);
  box-shadow: inset 0 0 30px #00000055;
}
</style>
