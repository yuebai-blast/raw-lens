<script setup lang="ts">
import { useCaptureStore } from '@/stores/captures'
const store = useCaptureStore()
</script>

<template>
  <header class="masthead">
    <div class="brand">
      <span
        class="brand-mark"
        aria-hidden="true"
      />
      <div class="brand-text">
        <h1>raw<span class="sep">·</span>lens</h1>
        <p>WIRE-LEVEL HTTP INSPECTOR</p>
      </div>
    </div>
    <div class="readouts">
      <div class="gauge">
        <span class="gauge-label">CAPTURED</span>
        <span class="gauge-value">{{ store.list.length }}</span>
      </div>
      <div class="gauge status">
        <span
          class="signal"
          aria-hidden="true"
        />
        <span class="gauge-value">{{ store.status }}</span>
      </div>
      <button
        class="btn-clear"
        title="清空所有记录"
        @click="store.clear()"
      >
        PURGE
      </button>
    </div>
  </header>
</template>

<style scoped>
/* ---- masthead ---- */
.masthead {
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 22px;
  border-bottom: 1px solid var(--line);
  background: linear-gradient(180deg, var(--panel-2), var(--panel));
  position: relative;
}
.masthead::after {
  content: ""; position: absolute; left: 0; right: 0; bottom: -1px; height: 1px;
  background: linear-gradient(90deg, transparent, var(--phosphor-soft) 30%, var(--phosphor-soft) 70%, transparent);
}

.brand { display: flex; align-items: center; gap: 14px; }
.brand-mark {
  width: 30px; height: 30px; flex: none; border-radius: 4px;
  background:
    radial-gradient(circle at 50% 50%, var(--phosphor) 0 3px, transparent 4px),
    conic-gradient(from 0deg, transparent 0 80%, var(--phosphor-soft) 100%);
  border: 1px solid #23332e;
  box-shadow: 0 0 12px #34e0a155, inset 0 0 8px #000;
  animation: spin 8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.brand-text h1 {
  margin: 0; font-family: var(--mono); font-weight: 500;
  font-size: 21px; letter-spacing: 1px; line-height: 1;
  color: var(--ink);
  text-shadow: 0 0 14px #34e0a140;
}
.brand-text h1 .sep { color: var(--phosphor); }
.brand-text p {
  margin: 4px 0 0; font-family: var(--mono); font-size: 10px;
  letter-spacing: 3.5px; color: var(--muted); font-weight: 400;
}

.readouts { display: flex; align-items: stretch; gap: 10px; }
.gauge {
  display: flex; flex-direction: column; gap: 3px; justify-content: center;
  padding: 6px 14px; min-width: 78px;
  border: 1px solid var(--line); border-radius: 5px;
  background: #0a0f0e;
}
.gauge-label { font-family: var(--mono); font-size: 9px; letter-spacing: 2px; color: var(--muted); }
.gauge-value { font-family: var(--mono); font-size: 16px; font-weight: 500; color: var(--ink); }
.gauge.status { flex-direction: row; align-items: center; gap: 8px; }
.gauge.status .gauge-value { font-size: 12px; letter-spacing: 1.5px; color: var(--phosphor); }

.signal {
  width: 8px; height: 8px; border-radius: 50%; background: var(--phosphor); flex: none;
  box-shadow: 0 0 0 0 #34e0a166; animation: pulse 1.8s ease-out infinite;
}
@keyframes pulse {
  0%   { box-shadow: 0 0 0 0 #34e0a17a; }
  70%  { box-shadow: 0 0 0 7px transparent; }
  100% { box-shadow: 0 0 0 0 transparent; }
}

.btn-clear {
  font-family: var(--mono); font-size: 11px; letter-spacing: 2px; font-weight: 500;
  color: var(--ink-dim); background: #0a0f0e;
  border: 1px solid var(--line); border-radius: 5px;
  padding: 0 16px; cursor: pointer; transition: all .15s;
}
.btn-clear:hover { color: var(--red); border-color: #4a2a2a; box-shadow: 0 0 12px #ef6f6c22; }
</style>
