import { describe, it, expect } from 'vitest'
import { b64ToBytes, bytesToText, toHexLines, fmtBytes, meterPct } from './bytes'

describe('b64ToBytes / bytesToText 往返保真', () => {
  it('round-trips ASCII 文本', () => {
    expect(bytesToText(b64ToBytes(btoa('GET /a HTTP/1.1')))).toBe('GET /a HTTP/1.1')
  })
  it('保留原始字节（含 CRLF）', () => {
    const bytes = b64ToBytes(btoa('a\r\nb'))
    expect(Array.from(bytes)).toEqual([97, 13, 10, 98])
  })
})

describe('toHexLines', () => {
  it('空输入返回空数组', () => {
    expect(toHexLines(new Uint8Array())).toEqual([])
  })
  it('每行含 offset / hex / ascii，非可见字节用点', () => {
    const lines = toHexLines(new Uint8Array([0x41, 0x42, 0x00]))
    expect(lines).toHaveLength(1)
    expect(lines[0]).toContain('00000000')
    expect(lines[0]).toContain('41 42 00')
    expect(lines[0]).toContain('|AB.|')
  })
})

describe('fmtBytes', () => {
  it('分级格式化', () => {
    expect(fmtBytes(512)).toBe('512 B')
    expect(fmtBytes(2048)).toBe('2.0 KB')
    expect(fmtBytes(3 * 1048576)).toBe('3.0 MB')
  })
})

describe('meterPct', () => {
  it('0 及以下为 0，单调递增且封顶 100', () => {
    expect(meterPct(0)).toBe(0)
    expect(meterPct(10)).toBeGreaterThan(0)
    expect(meterPct(1e9)).toBe(100)
  })
})
