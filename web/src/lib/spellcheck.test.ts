import { ensureSyntaxTree } from '@codemirror/language'
import { EditorState } from '@codemirror/state'
import { typst_lezer } from 'codemirror-lang-typst/lezer'
import { describe, expect, it } from 'vitest'
import { proseRanges } from './spellcheck'

function prose(doc: string) {
  const state = EditorState.create({ doc, extensions: [typst_lezer()] })
  ensureSyntaxTree(state, doc.length, 5000)
  return proseRanges(state, 0, doc.length).map((r) => doc.slice(r.from, r.to))
}

describe('proseRanges', () => {
  it('keeps markup text and skips code, math, raw, labels and references', () => {
    const doc = [
      '#import "@local/chessbook:0.1.0": *',
      '= Tiêu đề chương',
      'Khai cuộc *Ý* và $x^2$ với `mã` <nhan> @tham-chieu',
      '#let a = "chuỗi"',
      '#text(red)[nội dung]',
    ].join('\n')
    expect(prose(doc)).toEqual(['Tiêu đề chương', 'Khai cuộc', 'Ý', 'và', 'với', 'nội dung'])
  })
})
