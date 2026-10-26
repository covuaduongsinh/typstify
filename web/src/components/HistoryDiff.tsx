import { MergeView } from '@codemirror/merge'
import { EditorState } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { basicSetup } from 'codemirror'
import { typst_lezer } from 'codemirror-lang-typst/lezer'
import { useEffect, useRef } from 'react'
import { typstifyTheme } from '../lib/editorTheme'

/** Side-by-side, read-only comparison of a file in a version (left) and
 * now (right). Loaded lazily: the merge view is only needed here. */
export function HistoryDiff({ before, after, typst }: { before: string; after: string; typst: boolean }) {
  const hostRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    const host = hostRef.current
    if (!host) return
    const extensions = [
      basicSetup,
      typstifyTheme,
      ...(typst ? [typst_lezer()] : []),
      EditorState.readOnly.of(true),
      EditorView.editable.of(false),
    ]
    const view = new MergeView({
      a: { doc: before, extensions },
      b: { doc: after, extensions },
      parent: host,
      collapseUnchanged: { margin: 3, minSize: 6 },
      gutter: true,
    })
    return () => view.destroy()
  }, [before, after, typst])

  return <div className="history-diff" ref={hostRef} />
}
