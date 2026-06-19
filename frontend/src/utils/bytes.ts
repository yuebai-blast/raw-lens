export function b64ToBytes(b64: string): Uint8Array {
  const bin = atob(b64)
  const arr = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) arr[i] = bin.charCodeAt(i)
  return arr
}

const decoder = new TextDecoder('utf-8', { fatal: false })
export function bytesToText(b: Uint8Array): string {
  return decoder.decode(b)
}

export function toHexLines(bytes: Uint8Array): string[] {
  const lines: string[] = []
  for (let off = 0; off < bytes.length; off += 16) {
    const slice = bytes.slice(off, off + 16)
    const hex = [...slice].map((b) => b.toString(16).padStart(2, '0')).join(' ').padEnd(47, ' ')
    const ascii = [...slice].map((b) => (b >= 32 && b < 127 ? String.fromCharCode(b) : '.')).join('')
    lines.push(`${off.toString(16).padStart(8, '0')}  ${hex}  |${ascii}|`)
  }
  return lines
}

export function fmtBytes(n: number): string {
  if (n < 1024) return n + ' B'
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB'
  return (n / 1048576).toFixed(1) + ' MB'
}

// 对字节数取对数映射到 0–100%，用于列表里的大小条。
export function meterPct(n: number): number {
  if (n <= 0) return 0
  return Math.min(100, (Math.log10(n) / 5) * 100)
}
