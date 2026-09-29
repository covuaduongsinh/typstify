import { autocompletion, snippetCompletion, type CompletionContext, type CompletionResult } from '@codemirror/autocomplete'
import { linter, lintGutter, setDiagnostics, type Diagnostic as CmDiagnostic } from '@codemirror/lint'
import { EditorState, type Extension, type Text } from '@codemirror/state'
import { EditorView, hoverTooltip } from '@codemirror/view'
import { basicSetup } from 'codemirror'
import { typst_lezer } from 'codemirror-lang-typst/lezer'
import { forwardRef, useEffect, useImperativeHandle, useRef } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import ReactMarkdown from 'react-markdown'
import { typstifyTheme } from '../lib/editorTheme'
import { LspClient, type LspDiagnostic, type LspDocumentSymbol } from '../lib/lspClient'
import {
  applyGlobalColumns,
  applyGlobalFontSize,
  CHESSBOOK_IMPORT,
  detectDocumentColumns,
  detectDocumentFontSize,
  hasChessbookImport,
  repairChessImports,
} from '../lib/typst'

// Exposes an imperative save() so a toolbar button (and the global
// keyboard-shortcut dispatcher, see lib/shortcuts.ts) can trigger the same
// save path -- a shortcut alone isn't discoverable (observed: a user who
// pasted content in couldn't find any way to save it).
export interface EditorHandle {
  /** Saves the document; resolves true once the server confirmed it. */
  save: () => Promise<boolean>
  insertText: (text: string) => void
  /** Inserts a chess snippet atomically, ensuring #import is hoisted to line 1. */
  insertChessSnippet: (text: string) => void
  getContent: () => string
  /** Inserts `line` at the top of the document unless `present(doc)`. */
  ensureLineAtTop: (line: string, present: (doc: string) => boolean) => void
  /** Auto repairs missing chess library imports and saves */
  autoFixImports: () => void
  /** Applies font size globally across the entire document */
  setGlobalFontSize: (size: string | number) => void
  /** Applies column count (1 or 2) globally across the entire document */
  setGlobalColumns: (columns: 1 | 2) => void
  /** Adjusts global font size by delta (+0.5, -0.5, etc.) */
  adjustGlobalFontSize: (delta: number) => void
  /** Gets active document formatting settings */
  getGlobalSettings: () => { fontSize: number; columns: 1 | 2 }
  /** Fetches the document outline (LSP textDocument/documentSymbol). */
  getOutline: () => Promise<LspDocumentSymbol[]>
  /** Moves the caret to a 0-based line/character, scrolling it into view. */
  goToPosition: (line: number, character: number) => void
  /** Scrolls a 1-based line into view WITHOUT moving the caret or stealing
   * focus -- used by preview->editor scroll sync, which must not fight the
   * editor->preview direction (that one is driven by caret/selection
   * changes; this one deliberately avoids touching either). */
  scrollToLine: (line: number) => void
}

interface EditorProps {
  path: string
  initialContent: string
  /** The LSP connection for the whole workspace session (see Workspace.tsx) --
   * shared across file switches instead of one WebSocket per open file, so
   * changing files doesn't pay for a fresh /ws/lsp handshake + tinymist
   * re-init every time. Editor only sends didOpen/didChange/didClose for its
   * own path; it never creates or closes the connection itself. */
  lsp: LspClient
  onDirtyChange?: (dirty: boolean) => void
  /** Persists the content; a rejected promise keeps the file dirty. */
  onSave?: (content: string) => Promise<void> | void
  /** Reports a failed save (message) or a later successful one (null). */
  onSaveError?: (message: string | null) => void
  /** 1-based line/column of the main cursor and total document lines. */
  onCursorChange?: (pos: { line: number; col: number; totalLines: number }) => void
  /** Triggered when document content changes in the editor. */
  onDocChange?: (content: string) => void
  /** Error/warning counts of the latest LSP diagnostics for this file. */
  onDiagnosticsChange?: (counts: { errors: number; warnings: number }) => void
}

// LSP snippets allow bare $1/$0 tabstops (e.g. "${1:name}(${2:args})$0");
// CM6's snippet() only recognizes the braced ${1}/${1:default} form (or
// #{...}), so normalize bare tabstops to braced ones. `${1:x}` is already
// braced -- the digit there follows "{", not "$", so this regex (which only
// matches a digit run directly after "$") leaves it untouched.
function lspSnippetToCm6(template: string): string {
  return template.replace(/\$(\d+)/g, (_, n: string) => `\${${n}}`)
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

// tinymist hover content links a local resource (e.g. an image the doc
// references) as `command:tinymist.open{Internal,External}?["file:///..."]`.
// A browser can't open an arbitrary local path, so this just extracts a
// readable path to show instead of a dead link (mirrors the extraction in
// editor/hovertips.go's parseLink, minus the desktop-only "open it" part).
function extractTinymistLocalPath(href: string): string {
  const qIdx = href.indexOf('?')
  if (qIdx === -1) return href
  let raw = href.slice(qIdx + 1)
  try {
    raw = decodeURIComponent(raw)
  } catch {
    // leave raw as-is
  }
  raw = raw.replace(/^\[|\]$/g, '').replace(/^"|"$/g, '')
  raw = raw.replace(/^file:\/\//, '')
  return raw
}

function HoverLink({ href, children }: { href?: string; children?: React.ReactNode }) {
  if (!href) return <>{children}</>
  if (href.startsWith('http://') || href.startsWith('https://')) {
    return (
      <a href={href} target="_blank" rel="noopener noreferrer">
        {children}
      </a>
    )
  }
  if (href.startsWith('command:tinymist.open')) {
    const path = extractTinymistLocalPath(href)
    return (
      <span className="cm-typstify-hover-localpath" title={path}>
        {children}
      </span>
    )
  }
  return <>{children}</>
}

const HOVER_MARKDOWN_COMPONENTS = { a: HoverLink }

/** Editor is a CodeMirror 6 wrapper providing Typst syntax highlighting
 * (via codemirror-lang-typst's WASM-free Lezer grammar) plus live
 * completion/hover/diagnostics sourced from the tinymist LSP over
 * /ws/lsp (see server/lsp_ws.go and lib/lspClient.ts). */
export const Editor = forwardRef<EditorHandle, EditorProps>(function Editor(
  { path, initialContent, lsp, onDirtyChange, onSave, onSaveError, onCursorChange, onDocChange, onDiagnosticsChange },
  ref,
) {
  const hostRef = useRef<HTMLDivElement | null>(null)
  const viewRef = useRef<EditorView | null>(null)
  const lspRef = useRef<LspClient | null>(null)
  const changeTimerRef = useRef<number | undefined>(undefined)

  // Flushes any pending debounced didChange synchronously so the LSP's
  // document cache (which didSave relies on) has the latest content, and so
  // the debounce timer doesn't fire afterwards and re-mark the file dirty
  // right after this just cleared it. Exposed via the imperative handle,
  // driven by both a toolbar Save button and the global Ctrl+S shortcut
  // (Workspace.tsx / lib/shortcuts.ts).
  //
  // The file only counts as saved once onSave resolves: a failed PUT keeps
  // it dirty and is reported through onSaveError instead of the UI claiming
  // "Đã lưu" for content that never reached the server.
  const saveRef = useRef<() => Promise<boolean>>(async () => false)
  saveRef.current = async () => {
    const view = viewRef.current
    const lsp = lspRef.current
    if (!view || !lsp) return false
    window.clearTimeout(changeTimerRef.current)
    const text = view.state.doc.toString()
    lsp.didChange(path, text)
    try {
      await onSave?.(text)
    } catch (err) {
      onSaveError?.(err instanceof Error ? err.message : String(err))
      return false
    }
    lsp.didSave(path)
    onSaveError?.(null)
    // Keystrokes typed while the request was in flight are not saved yet.
    onDirtyChange?.(viewRef.current !== null && viewRef.current.state.doc.toString() !== text)
    return true
  }

  const insertTextRef = useRef<(textToInsert: string) => void>(() => {})
  insertTextRef.current = (textToInsert: string) => {
    const view = viewRef.current
    if (!view) return
    const sel = view.state.selection.main
    view.dispatch({
      changes: { from: sel.from, to: sel.to, insert: textToInsert },
      selection: { anchor: sel.from + textToInsert.length },
    })
    view.focus()
  }

  useImperativeHandle(ref, () => ({
    save: () => saveRef.current(),
    insertText: (text: string) => insertTextRef.current(text),
    insertChessSnippet: (textToInsert: string) => {
      const view = viewRef.current
      if (!view) return
      const currentDoc = view.state.doc.toString()
      const hasImport = hasChessbookImport(currentDoc)

      if (!hasImport) {
        if (currentDoc.trim() === '') {
          // Empty document: replace with import at top and snippet
          const newDoc = `${CHESSBOOK_IMPORT}\n\n${textToInsert}\n`
          view.dispatch({
            changes: { from: 0, to: currentDoc.length, insert: newDoc },
            selection: { anchor: newDoc.length },
          })
        } else {
          // Non-empty document: insert import at line 1, insert snippet at cursor
          const importPrefix = `${CHESSBOOK_IMPORT}\n\n`
          const sel = view.state.selection.main
          const insertPos = sel.from
          const snippetPos = insertPos === 0 ? importPrefix.length : importPrefix.length + insertPos
          view.dispatch({
            changes: [
              { from: 0, to: 0, insert: importPrefix },
              { from: sel.from, to: sel.to, insert: textToInsert },
            ],
            selection: { anchor: snippetPos + textToInsert.length },
          })
        }
      } else {
        // Document already has import: insert at current selection
        insertTextRef.current(textToInsert)
      }
      view.focus()
    },
    getContent: () => viewRef.current?.state.doc.toString() ?? '',
    ensureLineAtTop: (line: string, present: (doc: string) => boolean) => {
      const view = viewRef.current
      if (!view || present(view.state.doc.toString())) return
      view.dispatch({ changes: { from: 0, to: 0, insert: `${line}\n` } })
    },
    autoFixImports: () => {
      const view = viewRef.current
      if (!view) return
      const current = view.state.doc.toString()
      const fixed = repairChessImports(current)
      if (fixed !== current) {
        view.dispatch({
          changes: { from: 0, to: current.length, insert: fixed },
        })
        void saveRef.current()
      }
    },
    setGlobalFontSize: (size: string | number) => {
      const view = viewRef.current
      if (!view) return
      const current = view.state.doc.toString()
      const updated = applyGlobalFontSize(current, size)
      if (updated !== current) {
        view.dispatch({
          changes: { from: 0, to: current.length, insert: updated },
        })
        onDocChange?.(updated)
      }
      view.focus()
    },
    setGlobalColumns: (columns: 1 | 2) => {
      const view = viewRef.current
      if (!view) return
      const current = view.state.doc.toString()
      const updated = applyGlobalColumns(current, columns)
      if (updated !== current) {
        view.dispatch({
          changes: { from: 0, to: current.length, insert: updated },
        })
        onDocChange?.(updated)
      }
      view.focus()
    },
    adjustGlobalFontSize: (delta: number) => {
      const view = viewRef.current
      if (!view) return
      const current = view.state.doc.toString()
      const curSize = detectDocumentFontSize(current)
      const newSize = Math.max(6, Math.min(48, Math.round((curSize + delta) * 10) / 10))
      const updated = applyGlobalFontSize(current, newSize)
      if (updated !== current) {
        view.dispatch({
          changes: { from: 0, to: current.length, insert: updated },
        })
        onDocChange?.(updated)
      }
      view.focus()
    },
    getGlobalSettings: () => {
      const doc = viewRef.current?.state.doc.toString() ?? ''
      return {
        fontSize: detectDocumentFontSize(doc),
        columns: detectDocumentColumns(doc),
      }
    },
    getOutline: () => {
      const lsp = lspRef.current
      if (!lsp) return Promise.resolve([])
      return lsp.documentSymbols(path)
    },
    goToPosition: (line: number, character: number) => {
      const view = viewRef.current
      if (!view) return
      const pos = posFromLineChar(view.state.doc, line, character)
      view.dispatch({ selection: { anchor: pos }, effects: EditorView.scrollIntoView(pos, { y: 'center' }) })
      view.focus()
    },
    scrollToLine: (line: number) => {
      const view = viewRef.current
      if (!view) return
      const pos = posFromLineChar(view.state.doc, Math.max(0, line - 1), 0)
      view.dispatch({ effects: EditorView.scrollIntoView(pos, { y: 'center' }) })
    },
  }))

  useEffect(() => {
    lspRef.current = lsp

    const scheduleChange = (content: string) => {
      window.clearTimeout(changeTimerRef.current)
      changeTimerRef.current = window.setTimeout(() => {
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
          options: items.map((item) =>
            item.insertTextFormat === 2 && item.insertText
              ? snippetCompletion(lspSnippetToCm6(item.insertText), { label: item.label, detail: item.detail })
              : { label: item.label, detail: item.detail, apply: item.insertText ?? item.label },
          ),
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
          dom.className = 'cm-typstify-hover chat-text-markdown'
          let root: Root | null = createRoot(dom)
          root.render(<ReactMarkdown components={HOVER_MARKDOWN_COMPONENTS}>{contents}</ReactMarkdown>)
          return {
            dom,
            destroy: () => {
              // Deferred: CodeMirror may call destroy() synchronously while
              // still inside React's render/commit phase for this same
              // tree, and unmounting a root mid-render throws.
              const r = root
              root = null
              if (r) setTimeout(() => r.unmount(), 0)
            },
          }
        },
      }
    })

    const reportCursor = (state: EditorState) => {
      const head = state.selection.main.head
      const line = state.doc.lineAt(head)
      onCursorChange?.({ line: line.number, col: head - line.from + 1, totalLines: state.doc.lines })
    }

    const extensions: Extension[] = [
      basicSetup,
      typstifyTheme,
      typst_lezer(),
      autocompletion({ override: [completionSource] }),
      hover,
      linter(() => []), // registers the lint state field; diagnostics are pushed via setDiagnostics below
      lintGutter(),
      // Save is bound globally (Workspace.tsx's shortcut dispatcher, see
      // lib/shortcuts.ts) instead of here, so it works with the same
      // user-configurable key regardless of whether the editor has focus,
      // and so it fires exactly once instead of twice (a window-level
      // listener still sees a keydown CodeMirror's own keymap already
      // handled -- preventDefault() doesn't stop propagation).
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          const docStr = update.state.doc.toString()
          scheduleChange(docStr)
          onDocChange?.(docStr)
        }
        if (update.docChanged || update.selectionSet) reportCursor(update.state)
      }),
    ]

    const state = EditorState.create({ doc: initialContent, extensions })
    const view = new EditorView({ state, parent: hostRef.current! })
    viewRef.current = view
    reportCursor(state)
    onDiagnosticsChange?.({ errors: 0, warnings: 0 })

    lsp.didOpen(path, initialContent)
    // After a dropped /ws/lsp reconnects, the server-side session is new:
    // open the document again with what the user currently has.
    const unsubscribeReconnect = lsp.onReconnect(() => lsp.didOpen(path, view.state.doc.toString()))
    const unsubscribe = lsp.onDiagnostics((diagPath, diags) => {
      if (diagPath !== path) return
      view.dispatch(setDiagnostics(view.state, toCmDiagnostics(view.state.doc, diags)))
      onDiagnosticsChange?.({
        errors: diags.filter((d) => d.severity === 1).length,
        warnings: diags.filter((d) => d.severity === 2).length,
      })
    })

    return () => {
      window.clearTimeout(changeTimerRef.current)
      unsubscribe()
      unsubscribeReconnect()
      // Tells the shared LSP session this document is no longer open. The
      // WebSocket itself belongs to Workspace (one per project session, not
      // per file) and stays open across this file switch.
      lsp.didClose(path)
      view.destroy()
      viewRef.current = null
      lspRef.current = null
    }
    // Re-create the whole CodeMirror session when the open file changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [path, lsp])

  return <div className="editor-host" ref={hostRef} />
})
