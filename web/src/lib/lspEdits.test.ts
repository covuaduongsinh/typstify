import { EditorState } from '@codemirror/state'
import { describe, expect, it } from 'vitest'
import { lspEditsToChanges, lspOffset } from './lspEdits'

const apply = (doc: string, edits: Parameters<typeof lspEditsToChanges>[1]) => {
  const state = EditorState.create({ doc })
  return state.update({ changes: lspEditsToChanges(state.doc, edits) }).state.doc.toString()
}

describe('lspEdits', () => {
  it('maps positions, clamping past-the-end lines to the document end', () => {
    const doc = EditorState.create({ doc: 'ab\ncd' }).doc
    expect(lspOffset(doc, 1, 1)).toBe(4)
    expect(lspOffset(doc, 1, 99)).toBe(5)
    expect(lspOffset(doc, 2, 0)).toBe(5)
  })

  it('applies a whole-document replacement the way formatters send it', () => {
    const out = apply('#let  x=1\n= Tiêu đề\n', [
      { range: { start: { line: 0, character: 0 }, end: { line: 2, character: 0 } }, newText: '#let x = 1\n= Tiêu đề\n' },
    ])
    expect(out).toBe('#let x = 1\n= Tiêu đề\n')
  })

  it('applies several edits against the original document', () => {
    const out = apply('a  b\nc  d', [
      { range: { start: { line: 0, character: 1 }, end: { line: 0, character: 3 } }, newText: ' ' },
      { range: { start: { line: 1, character: 1 }, end: { line: 1, character: 3 } }, newText: ' ' },
    ])
    expect(out).toBe('a b\nc d')
  })
})
