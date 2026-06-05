'use strict';

const listEl = document.getElementById('list');
const detailEl = document.getElementById('detail');
const countEl = document.getElementById('count');
const statusText = document.getElementById('statusText');

let activeId = null;
let activeTab = 'raw';
let showCRLF = false;
let knownIds = new Set();
let firstLoad = true;
let current = null; // 当前详情数据

/* ---------- 工具 ---------- */
function b64ToBytes(b64) {
  const bin = atob(b64);
  const arr = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) arr[i] = bin.charCodeAt(i);
  return arr;
}
const decoder = new TextDecoder('utf-8', { fatal: false });
const bytesToText = (b) => decoder.decode(b);
const esc = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

function markCRLF(text) {
  return esc(text)
    .replace(/\r\n/g, '<span class="crlf">␍␊</span>\n')
    .replace(/(?<!␊)\n/g, '<span class="crlf">␊</span>\n');
}

function toHex(bytes) {
  if (!bytes.length) return '<span class="off">(空)</span>';
  let out = '';
  for (let off = 0; off < bytes.length; off += 16) {
    const slice = bytes.slice(off, off + 16);
    const hex = [...slice].map((b) => b.toString(16).padStart(2, '0')).join(' ').padEnd(47, ' ');
    const ascii = [...slice].map((b) => (b >= 32 && b < 127) ? String.fromCharCode(b) : '.').join('');
    out += `<span class="off">${off.toString(16).padStart(8, '0')}</span>  ` +
           `<span class="hx">${hex}</span>  ` +
           `<span class="as">|${esc(ascii)}|</span>\n`;
  }
  return out;
}

function fmtBytes(n) {
  if (n < 1024) return n + ' B';
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB';
  return (n / 1048576).toFixed(1) + ' MB';
}

/* meter 宽度：对字节数取对数，0–100% */
function meterPct(n) {
  if (n <= 0) return 0;
  return Math.min(100, (Math.log10(n) / 5) * 100);
}

/* ---------- 列表 ---------- */
async function refresh() {
  let items;
  try {
    items = await (await fetch('/api/requests')).json();
    statusText.textContent = 'CAPTURING';
  } catch {
    statusText.textContent = 'OFFLINE';
    return;
  }
  countEl.textContent = items.length;

  const ids = new Set(items.map((i) => i.id));
  listEl.innerHTML = items.map((it) => {
    const isNew = !firstLoad && !knownIds.has(it.id);
    const t = new Date(it.time).toLocaleTimeString('en-GB');
    const lock = it.tls ? '<span class="lock" title="TLS">🔒</span>' : '';
    return `<div class="item ${it.id === activeId ? 'active' : ''} ${isNew ? 'is-new' : ''}" data-id="${it.id}">
      <div class="item-top">
        <span class="chip" data-m="${esc(it.method || '?')}">${esc(it.method || '?')}</span>
        <span class="item-target">${esc(it.target || '/')}</span>
      </div>
      <div class="item-meta">
        <span class="id">#${it.id}</span>${lock}
        <span>${t}</span>
        <span class="meter"><span style="width:${meterPct(it.rawSize)}%"></span></span>
        <span>${fmtBytes(it.rawSize)}</span>
      </div>
    </div>`;
  }).join('');

  for (const el of listEl.querySelectorAll('.item')) {
    el.onclick = () => loadDetail(Number(el.dataset.id));
  }
  knownIds = ids;
  firstLoad = false;
}

/* ---------- 详情 ---------- */
async function loadDetail(id) {
  activeId = id;
  for (const el of listEl.querySelectorAll('.item')) {
    el.classList.toggle('active', Number(el.dataset.id) === id);
  }
  current = await (await fetch('/api/requests/' + id)).json();
  render();
}

function render() {
  const d = current;
  if (!d) return;
  const rawBytes = b64ToBytes(d.rawBase64);
  const bodyBytes = b64ToBytes(d.bodyBase64);
  const tab = (n, label) => `<div class="tab ${activeTab === n ? 'active' : ''}" data-tab="${n}">${label}</div>`;

  let pane = '';
  if (activeTab === 'raw') {
    const txt = bytesToText(rawBytes);
    pane =
      `<div class="hint">连接上读到的原始字节，顺序 / 大小写 / 空白完全保真
        <label class="toggle"><input type="checkbox" id="crlfToggle" ${showCRLF ? 'checked' : ''}>显示换行符</label>
        <button class="copy" id="copyRaw">COPY</button></div>
       <pre class="wire">${showCRLF ? markCRLF(txt) : esc(txt)}</pre>`;
  } else if (activeTab === 'headers') {
    const seen = new Map();
    let rows = `<tr class="reqline"><td class="idx"></td><td class="hname">⟶</td><td class="hval">${esc(d.requestLine)}</td></tr>`;
    d.headers.forEach((h, i) => {
      const key = h.name.toLowerCase();
      const dup = seen.has(key) ? `<span class="dupe">DUP·重复名</span>` : '';
      seen.set(key, true);
      rows += `<tr><td class="idx">${i + 1}</td><td class="hname">${esc(h.name)}</td><td class="hval">${esc(h.value)}${dup}</td></tr>`;
    });
    pane = `<div class="hint">按收到顺序排列，header 名保留原始大小写，重复名不合并</div><table class="htable">${rows}</table>`;
  } else if (activeTab === 'body') {
    pane = `<div class="hint">${d.bodySize} 字节 body · 文本视图</div>
            <pre class="wire">${esc(bytesToText(bodyBytes)) || '<span style="color:var(--muted)">(无 body)</span>'}</pre>`;
  } else {
    pane = `<div class="hint">完整原始字节 · 十六进制视图</div><pre class="hex">${toHex(rawBytes)}</pre>`;
  }

  const lock = d.tls
    ? '<span class="tls-on">🔒 TLS</span>'
    : '<span>明文 cleartext</span>';

  detailEl.innerHTML = `
    <div class="detail-bar">
      <div class="detail-line">
        <span class="chip" data-m="${esc(d.method || '?')}">${esc(d.method || '?')}</span>
        <span class="detail-target">${esc(d.target || '/')}</span>
        <span class="detail-proto">${esc(d.proto || '')}</span>
      </div>
      <div class="detail-meta">
        <span><b>#</b>${d.id}</span>
        <span><b>FROM</b> ${esc(d.remoteAddr)}</span>
        <span>${lock}</span>
        <span><b>HEADERS</b> ${d.headerCount}</span>
        <span><b>BODY</b> ${fmtBytes(d.bodySize)}</span>
        <span><b>RAW</b> ${fmtBytes(d.rawSize)}</span>
        <span><b>AT</b> ${new Date(d.time).toLocaleString('en-GB')}</span>
      </div>
      <div class="tabs">${tab('raw', 'RAW')}${tab('headers', 'HEADERS')}${tab('body', 'BODY')}${tab('hex', 'HEX')}</div>
    </div>
    <div class="pane">${pane}</div>`;

  for (const el of detailEl.querySelectorAll('.tab')) {
    el.onclick = () => { activeTab = el.dataset.tab; render(); };
  }
  const ct = document.getElementById('crlfToggle');
  if (ct) ct.onchange = () => { showCRLF = ct.checked; render(); };
  const cp = document.getElementById('copyRaw');
  if (cp) cp.onclick = () => {
    navigator.clipboard.writeText(bytesToText(rawBytes)).then(() => {
      cp.textContent = 'COPIED'; setTimeout(() => (cp.textContent = 'COPY'), 1200);
    });
  };
}

/* ---------- 清空 ---------- */
document.getElementById('clear').onclick = async () => {
  await fetch('/api/clear', { method: 'POST' });
  activeId = null; current = null;
  detailEl.innerHTML = `<div class="awaiting">
      <div class="awaiting-art">⌁ ⌁ ⌁</div>
      <p class="awaiting-title">AWAITING SIGNAL<span class="cursor">▌</span></p>
      <p class="awaiting-sub">已清空</p></div>`;
  refresh();
};

refresh();
setInterval(refresh, 1500);
