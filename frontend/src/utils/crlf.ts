export interface CRLFSegment {
  text: string
  crlf: boolean
}

// 把文本切成片段：换行（\r\n 或裸 \n）单独成 crlf 片段，便于组件高亮显示 ␍␊ / ␊。
export function markCRLFSegments(text: string): CRLFSegment[] {
  const segs: CRLFSegment[] = []
  const re = /\r\n|\n/g
  let last = 0
  let m: RegExpExecArray | null
  while ((m = re.exec(text)) !== null) {
    if (m.index > last) segs.push({ text: text.slice(last, m.index), crlf: false })
    segs.push({ text: m[0], crlf: true })
    last = m.index + m[0].length
  }
  if (last < text.length) segs.push({ text: text.slice(last), crlf: false })
  return segs
}
