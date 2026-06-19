import { describe, it, expect } from 'vitest'
import { tryParseJSON, highlightTokens } from './json'

describe('tryParseJSON', () => {
  it('合法 JSON 返回对象，非法返回 null', () => {
    expect(tryParseJSON('{"a":1}')).toEqual({ a: 1 })
    expect(tryParseJSON('not json')).toBeNull()
    expect(tryParseJSON('   ')).toBeNull()
  })
})

describe('highlightTokens', () => {
  it('键 / 字符串 / 数字 / 布尔 / null 各有类名', () => {
    const tokens = highlightTokens({ a: 'x', b: 1, c: true, d: null })
    const classes = new Set(tokens.map((t) => t.cls))
    expect(classes.has('json-key')).toBe(true)
    expect(classes.has('json-string')).toBe(true)
    expect(classes.has('json-number')).toBe(true)
    expect(classes.has('json-boolean')).toBe(true)
    expect(classes.has('json-null')).toBe(true)
  })
  it('拼回原文等于 JSON.stringify 两空格缩进', () => {
    const value = { a: [1, 'two', false] }
    const joined = highlightTokens(value).map((t) => t.text).join('')
    expect(joined).toBe(JSON.stringify(value, null, 2))
  })
})
