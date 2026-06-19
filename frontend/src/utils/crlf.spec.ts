import { describe, it, expect } from 'vitest'
import { markCRLFSegments } from './crlf'

describe('markCRLFSegments', () => {
  it('CRLF 与裸 LF 都标记为换行片段，文本片段原样保留', () => {
    const segs = markCRLFSegments('a\r\nb\nc')
    const joined = segs.map((s) => s.text).join('')
    expect(joined).toBe('a\r\nb\nc')
    expect(segs.some((s) => s.crlf)).toBe(true)
  })
  it('无换行时整段为单个非 crlf 片段', () => {
    expect(markCRLFSegments('abc')).toEqual([{ text: 'abc', crlf: false }])
  })
})
