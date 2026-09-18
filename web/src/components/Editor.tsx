import { autocompletion, type CompletionContext, type CompletionResult } from '@codemirror/autocomplete'
import { linter, lintGutter, setDiagnostics, type Diagnostic as CmDiagnostic } from '@codemirror/lint'
import { EditorState, type Extension, type Text } from '@codemirror/state'
import { EditorView, hoverTooltip, keymap } from '@codemirror/view'
import { basicSetup } from 'codemirror'
import { typst_lezer } from 'codemirror-lang-typst/lezer'
import { useEffect, useRef } from 'react'
import { LspClient, type LspDiagnostic } from '../lib/lspClient'

interface EditorProps {
  path: string
  initialContent: string
  onDirtyChange?: (dirty: boolean) => void
  onSave?: (content: string) => void
}

function posFromLineChar(doc: Text, line: number, character: number): number {
  if (line < 0 || line >= doc.lines) return 0
  const docLine = doc.line(line + 1)
  return docLine.from + Math.min(character, docLine.length)
}

function lineCharFromPos(doc: Text, pos: number): { line: number; character: number } {
  const docLine = doc.lineAt(pos)
  return { line: docLine.number - 1, character: pos - docLine.from }
}

function severityFromLsp(sev?: number): CmDiagnostic['severity'] {
  switch (sev) {
    case 1:
      return 'error'
    case 2:
      return 'warning'
    default:
      return 'info'
  }
}

function toCmDiagnostics(doc: Text, diags: LspDiagnostic[]): CmDiagnostic[] {
  return diags
    .map((d): CmDiagnostic | null => {
      const from = posFromLineChar(doc, d.range.start.line, d.range.start.character)
      const to = posFromLineChar(doc, d.range.end.line, d.range.end.character)
      if (to < from) return null
      return { from, to, severity: severityFromLsp(d.severity), message: d.message, source: d.source }
    })
    .filter((d): d is CmDiagnostic => d !== null)
}

/** Editor is a CodeMirror 6 wrapper providing Typst syntax highlighting
 * (via codemirror-lang-typst's WASM-free Lezer grammar) plus live
 * completion/hover/diagnostics sourced from the tinymist LSP over
 * /ws/lsp (see server/lsp_ws.go and lib/lspClient.ts). */
export function Editor({ path, initialContent, onDirtyChange, onSave }: EditorProps) {
  const hostRef = useRef<HTMLDivElement | null>(null)
  const viewRef = useRef<EditorView | null>(null)
  const lspRef = useRef<LspClient | null>(null)

  useEffect(() => {
    const lsp = new LspClient()
    lspRef.current = lsp

    let changeTimer: number | undefined
    const scheduleChange = (content: string) => {
      window.clearTimeout(changeTimer)
      changeTimer = window.setTimeout(() => {
        lsp.didChange(path, content)
        onDirtyChange?.(true)
      }, 250)
    }

    const completionSource = async (context: CompletionContext): Promise<CompletionResult | null> => {
      const word = context.matchBefore(/[\w.-]*/)
      if (!word && !context.explicit) return null
      const { line, character } = lineCharFromPos(context.state.doc, context.pos)
      try {
        const items = await lsp.complete(path, line, character)
        return {
          from: word ? word.from : context.pos,
          options: items.map((item) => ({
            label: item.label,
            detail: item.detail,
            apply: item.insertText ?? item.label,
          })),
        }
      } catch {
        return null
      }
    }

    const hover = hoverTooltip(async (view, pos) => {
      const { line, character } = lineCharFromPos(view.state.doc, pos)
      let contents: string | null
      try {
        contents = await lsp.hover(path, line, character)
      } catch {
        return null
      }
      if (!contents) return null
      return {
        pos,
        create: () => {
          const dom = document.createElement('div')
          dom.className = 'cm-typstify-hover'
          dom.textContent = contents
          return { dom }
        },
      }
    })

    const extensions: Extension[] = [
      basicSetup,
      typst_lezer(),
      autocompletion({ override: [completionSource] }),
      hover,
      linter(() => []), // registers the lint state field; diagnostics are pushed via setDiagnostics below
      lintGutter(),
      keymap.of([
        {
          key: 'Mod-s',
          preventDefault: true,
          run: (view) => {
            // Flush any pending debounced didChange synchronously so the LSP's
            // document cache (which didSave relies on) has the latest content,
            // and so the debounce timer doesn't fire afterwards and re-mark
            // the file dirty right after we just cleared that below.
            window.clearTimeout(changeTimer)
            const text = view.state.doc.toString()
            lsp.didChange(path, text)
            onSave?.(text)
            lsp.didSave(path)
            onDirtyChange?.(false)
            return true
          },
        },
      ]),
      EditorView.updateListener.of((update) => {
        if (update.docChanged) scheduleChange(update.state.doc.toString())
      }),
    ]

    const state = EditorState.create({ doc: initialContent, extensions })
    const view = new EditorView({ state, parent: hostRef.current! })
    viewRef.current = view

    lsp.didOpen(path, initialContent)
    const unsubscribe = lsp.onDiagnostics((diagPath, diags) => {
      if (diagPath !== path) return
      view.dispatch(setDiagnostics(view.state, toCmDiagnostics(view.state.doc, diags)))
    })

    return () => {
      window.clearTimeout(changeTimer)
      unsubscribe()
      lsp.didClose(path)
      lsp.close()
      view.destroy()
      viewRef.current = null
      lspRef.current = null
    }
    // Re-create the whole editor session when the open file changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [path])

  return <div className="editor-host" ref={hostRef} />
}
