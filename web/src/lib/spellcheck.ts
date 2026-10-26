import { syntaxTree } from '@codemirror/language'
import type { EditorState, Range } from '@codemirror/state'
import { Decoration, EditorView, ViewPlugin, type DecorationSet, type ViewUpdate } from '@codemirror/view'

const prose = Decoration.mark({ attributes: { spellcheck: 'true', lang: 'vi' } })

/** Ranges of prose (markup Text nodes outside raw blocks) within [from, to). */
export function proseRanges(state: EditorState, from: number, to: number): { from: number; to: number }[] {
  const out: { from: number; to: number }[] = []
  syntaxTree(state).iterate({
    from,
    to,
    enter(n) {
      // Code, math, labels and references carry no prose; raw text is code.
      if (n.name === 'Raw' || n.name === 'Equation' || n.name === 'Label' || n.name === 'Ref') return false
      if (n.name === 'Text' && n.to > n.from) out.push({ from: n.from, to: n.to })
    },
  })
  return out
}

function build(view: EditorView): DecorationSet {
  const marks: Range<Decoration>[] = []
  for (const r of view.visibleRanges) {
    for (const p of proseRanges(view.state, r.from, r.to)) marks.push(prose.range(p.from, p.to))
  }
  return Decoration.set(marks, true)
}

/**
 * The browser's own spellchecker, limited to prose: the editor content keeps
 * spellcheck off and only markup text is marked spellcheck="true" lang="vi",
 * so function names, math and raw blocks are never underlined. Which
 * dictionary is used is up to the browser (Chrome: Settings → Languages).
 */
export function proseSpellcheck() {
  return [
    EditorView.contentAttributes.of({ spellcheck: 'false', lang: 'vi' }),
    ViewPlugin.fromClass(
      class {
        decorations: DecorationSet
        constructor(view: EditorView) {
          this.decorations = build(view)
        }
        update(u: ViewUpdate) {
          if (u.docChanged || u.viewportChanged || syntaxTree(u.state) !== syntaxTree(u.startState))
            this.decorations = build(u.view)
        }
      },
      { decorations: (v) => v.decorations },
    ),
  ]
}
