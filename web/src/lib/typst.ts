// Helpers for generating Typst source from user-entered text.

/** typstString returns s as a Typst string literal ("..."), escaped so any
 * user text (player names, hints, comments) can be embedded safely. Inside
 * a string literal Typst interprets nothing but backslash escapes, so this
 * also sidesteps markup pitfalls like `*`, `$`, `_`, `#` or smart quotes. */
export function typstString(s: string): string {
  const escaped = s
    .replace(/\\/g, '\\\\')
    .replace(/"/g, '\\"')
    .replace(/\r?\n/g, ' ')
  return `"${escaped}"`
}

/** The primary import line that brings in the chessbook library. */
export const CHESSBOOK_IMPORT = '#import "@local/chessbook:0.1.0": *'

/** hasChessbookImport reports whether a document already imports the
 * chess library or defines required helpers locally. */
export function hasChessbookImport(doc: string): boolean {
  return /@local\/chessbook:|lib\/lib\.typ|chess_template\.typ|#let game-header|#let puzzle-card/.test(doc)
}

/** requiresChessImport detects if the document uses chessbook functions. */
export function requiresChessImport(doc: string): boolean {
  return /#(game-header|puzzle-card|eco-header|lesson-header|column-diagram|teaching-diagram|opening-diagram-box|nag|w[KQBNRP]|b[KQBNRP])\b/.test(
    doc,
  )
}

/** repairChessImports ensures that if a document uses chess functions, the required import is placed at the top. */
export function repairChessImports(doc: string): string {
  if (hasChessbookImport(doc) || !requiresChessImport(doc)) {
    return doc
  }
  return `${CHESSBOOK_IMPORT}\n\n${doc.trimStart()}`
}
