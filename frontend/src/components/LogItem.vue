<script setup lang="ts">
import type { Summary } from '@/types/api'
import { fmtBytes, meterPct } from '@/utils/bytes'

const props = defineProps<{ item: Summary; active: boolean; isNew: boolean }>()
const emit = defineEmits<{ select: [id: number]; delete: [id: number] }>()

function time(t: string): string {
  return new Date(t).toLocaleTimeString('en-GB')
}
</script>

<template>
  <div
    class="item"
    :class="{ active: props.active, 'is-new': props.isNew }"
    @click="emit('select', props.item.id)"
  >
    <div
      v-if="props.item.name"
      class="item-name"
      :title="props.item.name"
    >
      {{ props.item.name }}
    </div>
    <div class="item-top">
      <span
        class="chip"
        :data-m="props.item.method || '?'"
      >{{ props.item.method || '?' }}</span>
      <span class="item-target">{{ props.item.target || '/' }}</span>
      <button
        class="item-del"
        title="删除此记录"
        @click.stop="emit('delete', props.item.id)"
      >
        ✕
      </button>
    </div>
    <div class="item-meta">
      <span class="id">#{{ props.item.id }}</span>
      <span
        v-if="props.item.tls"
        class="lock"
        title="TLS"
      >🔒</span>
      <span>{{ time(props.item.time) }}</span>
      <span class="meter"><span :style="{ width: meterPct(props.item.rawSize) + '%' }" /></span>
      <span>{{ fmtBytes(props.item.rawSize) }}</span>
    </div>
  </div>
</template>

<style scoped>
/* ---- 列表项 ---- */
.item {
  position: relative; display: grid; gap: 6px;
  padding: 12px 18px 13px; cursor: pointer;
  border-bottom: 1px solid var(--line-soft);
  border-left: 2px solid transparent;
  transition: background .12s, border-color .12s;
}
.item:hover { background: #0e1715; }
.item.active { background: #0f1d18; border-left-color: var(--phosphor); }
.item.active::after {
  content: ""; position: absolute; right: 14px; top: 50%; transform: translateY(-50%);
  width: 5px; height: 5px; border-radius: 50%; background: var(--phosphor);
  box-shadow: 0 0 8px var(--phosphor);
}
.item.is-new { animation: flash 1.2s ease-out; }
@keyframes flash {
  0%   { background: #143228; box-shadow: inset 2px 0 0 var(--phosphor); }
  100% { background: transparent; box-shadow: inset 2px 0 0 transparent; }
}

.item-name {
  font-family: var(--mono); font-size: 11px; color: var(--phosphor);
  letter-spacing: .5px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}

.item-top { display: flex; align-items: baseline; gap: 9px; min-width: 0; }

/* 删除按钮：默认隐藏，hover 列表项时显现 */
.item-del {
  flex: none; margin-left: auto; padding: 2px 6px;
  font-family: var(--mono); font-size: 11px; line-height: 1;
  color: var(--muted); background: transparent;
  border: 1px solid var(--line); border-radius: 4px; cursor: pointer;
  opacity: 0; transition: opacity .12s, color .12s, border-color .12s;
}
.item:hover .item-del { opacity: 1; }
.item-del:hover { color: var(--red); border-color: #4a2a2a; }

/* 注：.chip 基础样式在 global.css，此处仅 override detail-bar 内尺寸（若需要） */

.item-target {
  font-family: var(--mono); font-size: 12.5px; color: var(--ink);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; flex: 1; min-width: 0;
}
.item-meta {
  display: flex; align-items: center; gap: 10px;
  font-family: var(--mono); font-size: 10.5px; color: var(--muted);
}
.item-meta .id { color: var(--ink-dim); }
.item-meta .lock { color: var(--phosphor); }
.meter { flex: 1; height: 2px; background: var(--line); border-radius: 2px; overflow: hidden; }
.meter > span { display: block; height: 100%; background: linear-gradient(90deg, var(--phosphor), #1f8c63); }
</style>
