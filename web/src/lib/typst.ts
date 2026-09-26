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

/** Function names provided by @local/chessbook that must NOT be redefined
 *  with `#let` inside user documents. AI assistants sometimes generate
 *  these "mock" definitions, which shadow the library versions and cause
 *  compile errors due to signature mismatches. */
const CHESSBOOK_DEFINED_FUNCTIONS = [
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
  'turn-box',
  'note-num',
  'nag',
]

const CHESS_FUNCTION_NAMES = [
  ...CHESSBOOK_DEFINED_FUNCTIONS,
  'w[KQBNRP]',
  'b[KQBNRP]',
]

const CHESS_REGEX = new RegExp(`#(?:${CHESS_FUNCTION_NAMES.join('|')})\\b`)

/** requiresChessImport detects if the document uses chessbook functions or figurines. */
export function requiresChessImport(doc: string): boolean {
  return CHESS_REGEX.test(doc)
}

/**
 * stripChessbookMockDefinitions removes `#let <name>(...) = { ... }` blocks
 * for any function that is already provided by @local/chessbook. These mock
 * definitions are injected by AI assistants when they fail to resolve the
 * chessbook import, and they shadow the real library definitions causing
 * signature mismatches (e.g. `unexpected argument: turn`).
 *
 * The algorithm walks line-by-line, tracking brace depth to find the closing
 * `}` of each `#let name(` block, then removes the entire range.
 */
export function stripChessbookMockDefinitions(doc: string): string {
  const namePattern = new RegExp(
    `^#let\\s+(?:${CHESSBOOK_DEFINED_FUNCTIONS.map((n) => n.replace(/-/g, '-')).join('|')})\\s*[\\(]`,
  )

  const lines = doc.split('\n')
  const removeRanges: Array<[number, number]> = []
  let i = 0

  while (i < lines.length) {
    const line = lines[i]
    if (namePattern.test(line.trim())) {
      // Found a mock `#let`. Walk forward counting braces to find the
      // matching close of `= { ... }`.
      const start = i
      let depth = 0
      let foundOpen = false
      let j = i

      for (; j < lines.length; j++) {
        for (const ch of lines[j]) {
          if (ch === '{') {
            depth++
            foundOpen = true
          } else if (ch === '}') {
            depth--
          }
        }
        if (foundOpen && depth <= 0) break
      }
      // Remove trailing blank lines after the block.
      let end = j
      while (end + 1 < lines.length && lines[end + 1].trim() === '') end++
      removeRanges.push([start, end])
      i = end + 1
    } else {
      i++
    }
  }

  if (removeRanges.length === 0) return doc

  const kept: string[] = []
  let cursor = 0
  for (const [start, end] of removeRanges) {
    for (let k = cursor; k < start; k++) kept.push(lines[k])
    cursor = end + 1
  }
  for (let k = cursor; k < lines.length; k++) kept.push(lines[k])

  return kept.join('\n')
}

/** repairChessImports ensures that if a document uses chess functions, the
 *  required import is placed at the top. It also removes any inline `#let`
 *  mock definitions that shadow the real chessbook library functions. */
export function repairChessImports(doc: string): string {
  // Step 1: strip mock #let definitions that override chessbook functions.
  let result = hasChessbookImport(doc) ? stripChessbookMockDefinitions(doc) : doc

  // Step 2: add the import if it's still missing.
  if (!hasChessbookImport(result) && requiresChessImport(result)) {
    result = `${CHESSBOOK_IMPORT}\n\n${result.trimStart()}`
  }
  return result
}
