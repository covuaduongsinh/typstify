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
  'chess-worksheet-init',
  'puzzle-grid-a4',
  'puzzle-grid-16x24',
  'render-puzzle-collection',
  'csv-to-puzzles',
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

const CHESS_IMPORT_LINE_REGEX =
  /^[ \t]*#import[ \t]+["'](?:@local\/chessbook:[^"']+|lib\/lib\.typ|chess_template\.typ)["'][^\r\n]*\r?\n?/gm

/**
 * hoistChessbookImport ensures that if a document imports or requires chessbook,
 * all duplicate or misplaced chessbook import lines are removed from the body and
 * a single `#import "@local/chessbook:0.1.0": *` is placed at the very top of the file.
 */
export function hoistChessbookImport(doc: string): string {
  const hasImport = hasChessbookImport(doc)
  const needsImport = hasImport || requiresChessImport(doc)
  if (!needsImport) return doc

  // Remove all existing chessbook import lines from the document body
  let cleaned = doc.replace(CHESS_IMPORT_LINE_REGEX, '').trimStart()
  cleaned = cleaned.replace(/^\n+/, '')

  return `${CHESSBOOK_IMPORT}\n\n${cleaned}`
}

/**
 * repairEmbeddedChessBlocks converts raw markdown chess fences (```chessboard, ```fen, ```pgn)
 * embedded inside a Typst document into proper chessbook function calls.
 */
export function repairEmbeddedChessBlocks(doc: string): string {
  // 1. Transform ```chessboard / ```fen / ```diagram blocks
  const fenBlockRegex =
    /```(?:chessboard|chess-board|chess_board|fen|chess-fen|chess_fen|diagram|teaching-diagram|board)\s*\r?\n([\s\S]*?)```/gi

  let res = doc.replace(fenBlockRegex, (_, content) => {
    const lines = content.split(/\r?\n/).map((l: string) => l.trim()).filter(Boolean)
    let fen = ''
    let title = 'Thế cờ'
    let turn = ''
    let caption = ''

    for (const line of lines) {
      if (/^(?:fen|FEN|Fen|thế cờ|Thế cờ)\s*:/i.test(line)) {
        fen = line.replace(/^(?:fen|FEN|Fen|thế cờ|Thế cờ)\s*:\s*/i, '').trim().replace(/^["']|["']$/g, '')
        continue
      }
      const parts = line.split(' ')
      const rows = parts[0].split('/')
      if (rows.length === 8 && /^[rnbqkpRNBQKP1-8]+$/.test(rows[0])) {
        fen = line.trim().replace(/^["']|["']$/g, '')
        continue
      }
      const colonIdx = line.indexOf(':')
      if (colonIdx > 0) {
        const key = line.slice(0, colonIdx).trim().toLowerCase()
        const val = line.slice(colonIdx + 1).trim().replace(/^["']|["']$/g, '')
        if (key === 'title') title = val
        else if (key === 'turn') turn = val === 'b' || val === 'black' || val === 'đen' ? 'b' : 'w'
        else if (key === 'caption') caption = val
      }
    }

    if (!fen && lines.length > 0) fen = lines[0]
    if (!turn && fen) {
      const parts = fen.split(' ')
      if (parts.length >= 2 && (parts[1] === 'w' || parts[1] === 'b')) turn = parts[1]
    }
    if (!turn) turn = 'w'
    if (!fen) fen = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'

    return `#teaching-diagram(\n  ${typstString(fen)},\n  title: ${typstString(title)},\n  turn: "${turn}",\n  size: 16pt,\n  caption: ${typstString(caption)}\n)`
  })

  return res
}

/** repairChessImports ensures that if a document uses chess functions:
 *  1. Any mock `#let` definitions that shadow the library are removed.
 *  2. Any misplaced or duplicate `#import` lines are hoisted to line 1.
 *  3. Any residual Markdown chess blocks (```chessboard, ```fen) are converted to #teaching-diagram. */
export function repairChessImports(doc: string): string {
  // Step 1: strip mock #let definitions that override chessbook functions.
  let result = stripChessbookMockDefinitions(doc)

  // Step 2: convert raw markdown chess blocks (```chessboard, ```fen)
  result = repairEmbeddedChessBlocks(result)

  // Step 3: hoist/ensure chessbook import is at the very top (line 1).
  result = hoistChessbookImport(result)

  return result
}

/** detectDocumentFontSize finds the global font size in pt from #set text. Defaults to 11. */
export function detectDocumentFontSize(doc: string): number {
  const match = doc.match(/#set\s+text\s*\(\s*(?:[^)]*?\b)?size\s*:\s*(\d+(?:\.\d+)?)\s*pt/i)
  if (match) {
    const val = parseFloat(match[1])
    if (Number.isFinite(val) && val > 0) return val
  }
  const simpleMatch = doc.match(/#set\s+text\s*\(\s*(\d+(?:\.\d+)?)\s*pt\s*\)/i)
  if (simpleMatch) {
    const val = parseFloat(simpleMatch[1])
    if (Number.isFinite(val) && val > 0) return val
  }
  return 11
}

/** detectDocumentColumns finds whether the document uses 1 or 2 columns globally. */
export function detectDocumentColumns(doc: string): 1 | 2 {
  const match = doc.match(/#set\s+page\s*\([^)]*?\bcolumns\s*:\s*(\d+)/i)
  if (match && match[1] === '2') return 2
  return 1
}

/**
 * applyGlobalFontSize updates or inserts the global `#set text(size: ...)`
 * at the top of the document, and removes orphan `#set text(size: ...)`
 * scattered down in the body.
 */
export function applyGlobalFontSize(doc: string, sizePt: number | string): string {
  const sizeNum = typeof sizePt === 'number' ? sizePt : parseFloat(sizePt)
  const formattedSize = Number.isFinite(sizeNum) ? `${sizeNum}pt` : (String(sizePt).endsWith('pt') ? String(sizePt) : `${sizePt}pt`)

  let result = doc

  // 1. Remove any stray `#set text(size: ...)` lines deeper in the body
  const lines = result.split('\n')
  const cleanedLines: string[] = []
  let foundTopSetText = false

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const isSetText = /^\s*#set\s+text\s*\(\s*(?:[^)]*?\b)?size\s*:\s*\d+(?:\.\d+)?\s*pt/i.test(line)

    if (isSetText) {
      if (!foundTopSetText && i < 25) {
        // Update top-level set text
        const updated = line.replace(/size\s*:\s*\d+(?:\.\d+)?\s*pt/i, `size: ${formattedSize}`)
        cleanedLines.push(updated)
        foundTopSetText = true
      } else {
        // Strip duplicate or body-level #set text
        continue
      }
    } else {
      cleanedLines.push(line)
    }
  }

  result = cleanedLines.join('\n')

  // 2. If no top-level #set text was found, insert it at the proper header position
  if (!foundTopSetText) {
    const textDirective = `#set text(size: ${formattedSize})`
    result = insertAtDocumentHeader(result, textDirective)
  }

  return result
}

/**
 * applyGlobalColumns updates or inserts the global `#set page(columns: ...)`
 * at the top of the document, and removes orphan `#set page(columns: ...)`
 * scattered down in the body.
 */
export function applyGlobalColumns(doc: string, columns: 1 | 2, gutter: string = '14pt'): string {
  let result = doc

  // 1. Remove any stray `#set page(columns: ...)` lines deeper in the body
  const lines = result.split('\n')
  const cleanedLines: string[] = []
  let foundTopSetPage = false

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const isSetPageCols = /^\s*#set\s+page\s*\([^)]*?\bcolumns\s*:\s*\d+/i.test(line)

    if (isSetPageCols) {
      if (!foundTopSetPage && i < 25) {
        // Update top-level set page
        let updated = line
        if (columns === 2) {
          updated = updated.replace(/columns\s*:\s*\d+/i, `columns: 2`)
          if (!updated.includes('gutter:')) {
            updated = updated.replace(/columns\s*:\s*2/i, `columns: 2, gutter: ${gutter}`)
          }
        } else {
          updated = updated.replace(/columns\s*:\s*\d+(?:\s*,\s*gutter\s*:\s*[^,\)]+)?/i, `columns: 1`)
        }
        cleanedLines.push(updated)
        foundTopSetPage = true
      } else {
        // Strip duplicate or body-level #set page columns
        continue
      }
    } else {
      cleanedLines.push(line)
    }
  }

  result = cleanedLines.join('\n')

  // 2. If no top-level #set page was found, insert it at the proper header position
  if (!foundTopSetPage) {
    const colDirective = columns === 2
      ? `#set page(columns: 2, gutter: ${gutter})`
      : `#set page(columns: 1)`
    result = insertAtDocumentHeader(result, colDirective)
  }

  return result
}

/**
 * Helper to insert a configuration directive (#set text / #set page)
 * right after #import lines and #show: ... init blocks at the top of the document.
 */
function insertAtDocumentHeader(doc: string, directive: string): string {
  const lines = doc.split('\n')
  let insertIdx = 0

  // Walk past initial imports and show init blocks
  let inShowBlock = false
  for (let i = 0; i < Math.min(lines.length, 35); i++) {
    const trimmed = lines[i].trim()
    if (trimmed.startsWith('#import')) {
      insertIdx = i + 1
      continue
    }
    if (trimmed.startsWith('#show:') || trimmed.startsWith('#show ')) {
      inShowBlock = true
      insertIdx = i + 1
      continue
    }
    if (inShowBlock) {
      insertIdx = i + 1
      if (trimmed.endsWith(')') || trimmed === ')') {
        inShowBlock = false
      }
      continue
    }
    if (trimmed.startsWith('#set ') && (trimmed.includes('text') || trimmed.includes('page'))) {
      insertIdx = i + 1
      continue
    }
    // First non-header line reached
    if (trimmed !== '' && !trimmed.startsWith('//')) {
      break
    }
  }

  lines.splice(insertIdx, 0, directive)
  return lines.join('\n')
}


