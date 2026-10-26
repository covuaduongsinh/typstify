import type { ChangeSpec, Text } from '@codemirror/state'

export interface LspTextEdit {
  range: { start: { line: number; character: number }; end: { line: number; character: number } }
  newText: string
}

/** Offset of an LSP position (0-based line, UTF-16 character). A position
 * past the last line (formatters replace "the whole document" up to
 * {line: lineCount, character: 0}) maps to the end of the document. */
export function lspOffset(doc: Text, line: number, character: number): number {
  if (line < 0) return 0
  if (line >= doc.lines) return doc.length
  const l = doc.line(line + 1)
  return l.from + Math.min(character, l.length)
}

/** CodeMirror changes for a set of LSP edits, all expressed against `doc`
 * (the LSP spec's semantics, which CodeMirror's change sets share). */
export function lspEditsToChanges(doc: Text, edits: LspTextEdit[]): ChangeSpec[] {
  return edits.map((e) => ({
    from: lspOffset(doc, e.range.start.line, e.range.start.character),
    to: lspOffset(doc, e.range.end.line, e.range.end.character),
    insert: e.newText,
  }))
}
