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
 * chess library or lib/lib.typ. */
export function hasChessbookImport(doc: string): boolean {
  return /@local\/chessbook:|lib\/lib\.typ|chess_template\.typ/.test(doc)
}

const CHESS_FUNCTION_NAMES = [
  'game-header',
  'puzzle-card',
  'eco-header',
  'eco-table',
  'opening-diagram-box',
  'lesson-header',
  'concept-box',
  'teaching-diagram',
  'practice-question',
  'instructor-note',
  'column-diagram',
  'chess-quote',
  'chess-board',
  'chess-book-init',
  'chess-magazine-init',
  'difficulty-stars',
  'upside-down-solutions',
  'render-puzzle-solutions',
  'turn-indicator',
  'note-num',
  'nag',
  'w[KQBNRP]',
  'b[KQBNRP]',
]

const CHESS_REGEX = new RegExp(`#(?:${CHESS_FUNCTION_NAMES.join('|')})\\b`)

/** requiresChessImport detects if the document uses chessbook functions or figurines. */
export function requiresChessImport(doc: string): boolean {
  return CHESS_REGEX.test(doc)
}

/** repairChessImports ensures that if a document uses chess functions, the required import is placed at the top. */
export function repairChessImports(doc: string): string {
  if (hasChessbookImport(doc)) {
    return doc
  }
  if (!requiresChessImport(doc)) {
    return doc
  }
  return `${CHESSBOOK_IMPORT}\n\n${doc.trimStart()}`
}
