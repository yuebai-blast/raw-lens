export function tryParseJSON(text: string): unknown | null {
  const trimmed = text.trim()
  if (!trimmed) return null
  try {
    return JSON.parse(trimmed)
  } catch {
    return null
  }
}

export interface JSONToken {
  text: string
  cls: string
}

const tokenRe =
  /("(?:\\.|[^"\\])*"(?=\s*:))|("(?:\\.|[^"\\])*")|(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)|\b(true|false)\b|\bnull\b|([{}[\],:])/g

// 把缩进后的 JSON 切成带类名的 token 序列，由组件渲染（不拼 HTML，交给 Vue 转义）。
export function highlightTokens(value: unknown): JSONToken[] {
  const json = JSON.stringify(value, null, 2)
  const tokens: JSONToken[] = []
  let last = 0
  let m: RegExpExecArray | null
  tokenRe.lastIndex = 0
  while ((m = tokenRe.exec(json)) !== null) {
    if (m.index > last) tokens.push({ text: json.slice(last, m.index), cls: '' })
    const [token, key, string, number, bool] = m
    const cls = key
      ? 'json-key'
      : string
        ? 'json-string'
        : number
          ? 'json-number'
          : bool
            ? 'json-boolean'
            : token === 'null'
              ? 'json-null'
              : 'json-punct'
    tokens.push({ text: token, cls })
    last = m.index + token.length
  }
  if (last < json.length) tokens.push({ text: json.slice(last), cls: '' })
  return tokens
}
