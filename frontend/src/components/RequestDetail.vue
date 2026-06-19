<script setup lang="ts">
import { ref, computed } from 'vue'
import { useCaptureStore } from '@/stores/captures'
import { fmtBytes } from '@/utils/bytes'
import RawView from './detail/RawView.vue'
import HeadersView from './detail/HeadersView.vue'
import BodyView from './detail/BodyView.vue'
import HexView from './detail/HexView.vue'

const store = useCaptureStore()
const tab = ref<'raw' | 'headers' | 'body' | 'hex'>('raw')
const d = computed(() => store.current)

function at(t: string): string {
  return new Date(t).toLocaleString('en-GB')
}
</script>

<template>
  <section class="readout">
    <template v-if="d">
      <div class="detail-bar">
        <div class="detail-line">
          <span
            class="chip"
            :data-m="d.method || '?'"
          >{{ d.method || '?' }}</span>
          <span class="detail-target">{{ d.target || '/' }}</span>
          <span class="detail-proto">{{ d.proto }}</span>
        </div>
        <div class="detail-meta">
          <span><b>#</b>{{ d.id }}</span>
          <span><b>FROM</b> {{ d.remoteAddr }}</span>
          <span><span
            v-if="d.tls"
            class="tls-on"
          >🔒 TLS</span><span v-else>明文 cleartext</span></span>
          <span><b>HEADERS</b> {{ d.headerCount }}</span>
          <span><b>BODY</b> {{ fmtBytes(d.bodySize) }}</span>
          <span><b>RAW</b> {{ fmtBytes(d.rawSize) }}</span>
          <span><b>AT</b> {{ at(d.time) }}</span>
        </div>
        <div class="tabs">
          <div
            class="tab"
            :class="{ active: tab === 'raw' }"
            @click="tab = 'raw'"
          >
            RAW
          </div>
          <div
            class="tab"
            :class="{ active: tab === 'headers' }"
            @click="tab = 'headers'"
          >
            HEADERS
          </div>
          <div
            class="tab"
            :class="{ active: tab === 'body' }"
            @click="tab = 'body'"
          >
            BODY
          </div>
          <div
            class="tab"
            :class="{ active: tab === 'hex' }"
            @click="tab = 'hex'"
          >
            HEX
          </div>
        </div>
      </div>
      <div class="pane">
        <RawView
          v-if="tab === 'raw'"
          :detail="d"
        />
        <HeadersView
          v-else-if="tab === 'headers'"
          :detail="d"
        />
        <BodyView
          v-else-if="tab === 'body'"
          :detail="d"
        />
        <HexView
          v-else
          :detail="d"
        />
      </div>
    </template>
    <div
      v-else
      class="awaiting"
    >
      <div
        class="awaiting-art"
        aria-hidden="true"
      >
        ⌁ ⌁ ⌁
      </div>
      <p class="awaiting-title">
        AWAITING SIGNAL<span class="cursor">▌</span>
      </p>
      <p class="awaiting-sub">
        把请求发到抓包端口，原始字节会在此显形<br>顺序 · 大小写 · 重复 header · body —— 一字不改
      </p>
    </div>
  </section>
</template>

<style scoped>
/* ---- readout 容器 ---- */
.readout { overflow-y: auto; min-height: 0; position: relative; }

/* ---- 空态等待 ---- */
.awaiting {
  height: 100%; display: flex; flex-direction: column; align-items: center; justify-content: center;
  text-align: center; gap: 14px; color: var(--muted);
}
.awaiting-art { font-size: 30px; color: var(--phosphor); letter-spacing: 10px; opacity: .55; animation: drift 3s ease-in-out infinite; }
@keyframes drift { 50% { opacity: .9; letter-spacing: 16px; } }
.awaiting-title { font-family: var(--mono); font-size: 15px; letter-spacing: 4px; color: var(--ink-dim); margin: 0; }
.awaiting-sub { font-family: var(--mono); font-size: 11.5px; line-height: 1.9; color: var(--muted); margin: 0; }
.cursor { color: var(--phosphor); animation: blink 1.1s steps(1) infinite; }
@keyframes blink { 50% { opacity: 0; } }

/* ---- 详情头 ---- */
.detail-bar {
  position: sticky; top: 0; z-index: 5;
  padding: 16px 24px 0; background: linear-gradient(180deg, var(--bg) 70%, transparent);
  border-bottom: 1px solid var(--line);
  overflow: hidden;
}
.detail-bar::before {
  content: ""; position: absolute; top: 0; left: -40%; width: 40%; height: 2px;
  background: linear-gradient(90deg, transparent, var(--phosphor), transparent);
  animation: sweep 3.4s linear infinite; opacity: .7;
}
@keyframes sweep { to { left: 110%; } }

.detail-line { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.detail-line .chip { font-size: 11px; padding: 3px 9px; }
.detail-target {
  font-family: var(--mono); font-size: 15px; color: var(--ink); font-weight: 500;
  word-break: break-all;
}
.detail-proto { font-family: var(--mono); font-size: 11px; color: var(--muted); margin-left: auto; }

.detail-meta {
  display: flex; flex-wrap: wrap; gap: 6px 20px;
  margin: 12px 0 14px; font-family: var(--mono); font-size: 11px; color: var(--ink-dim);
}
.detail-meta b { color: var(--muted); font-weight: 400; }
.tls-on { color: var(--phosphor); }

/* ---- tabs ---- */
.tabs { display: flex; gap: 2px; }
.tab {
  font-family: var(--mono); font-size: 11px; letter-spacing: 1.5px;
  padding: 8px 16px; cursor: pointer; color: var(--muted);
  border: 1px solid transparent; border-bottom: none; border-radius: 5px 5px 0 0;
  transition: color .12s, background .12s;
}
.tab:hover { color: var(--ink-dim); }
.tab.active { color: var(--phosphor); background: var(--panel-2); border-color: var(--line); }

/* ---- 内容区 ---- */
.pane { padding: 20px 24px 40px; }
</style>
