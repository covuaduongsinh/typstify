import { StateEffect, StateField, type Extension } from '@codemirror/state'
import { Decoration, EditorView, type DecorationSet } from '@codemirror/view'

export interface CommentRange {
  id: string
  from: number
  to: number
}

/** Replaces the highlighted comment ranges (open threads only). */
export const setCommentRanges = StateEffect.define<CommentRange[]>()

const mark = Decoration.mark({ class: 'cm-comment-range' })

function build(ranges: CommentRange[], length: number): DecorationSet {
  const valid = ranges
    .map((r) => ({ from: Math.max(0, Math.min(r.from, length)), to: Math.max(0, Math.min(r.to, length)) }))
    .filter((r) => r.to > r.from)
    .sort((a, b) => a.from - b.from)
  return Decoration.set(
    valid.map((r) => mark.range(r.from, r.to)),
    true,
  )
}

const field = StateField.define<DecorationSet>({
  create: () => Decoration.none,
  update(deco, tr) {
    deco = deco.map(tr.changes)
    for (const e of tr.effects) if (e.is(setCommentRanges)) deco = build(e.value, tr.state.doc.length)
    return deco
  },
  provide: (f) => EditorView.decorations.from(f),
})

/** Underlines text that open comment threads are attached to. */
export const commentHighlights = (): Extension => field
