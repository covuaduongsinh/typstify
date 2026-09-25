// PGN (Portable Game Notation) → Typst for the chessbook library.
//
// Move text is emitted as Typst string literals or structured tables
// to ensure valid Typst syntax while supporting magazine-quality layouts.
import { typstString } from './typst'

export interface PgnGame {
  headers: Record<string, string>
  /** Move text with the result token removed. */
  movetext: string
  /** Result from the Result tag, else the trailing result token, else ''. */
  result: string
  /** Structured moves parsed from movetext */
  moves?: PgnMove[]
}

export interface PgnMove {
  num: number
  white: string
  whiteNag?: string
  whiteComment?: string
  black?: string
  blackNag?: string
  blackComment?: string
}

export type PgnLayout = 'magazine' | 'inline' | 'header_only' | 'annotated'

const RESULT_TOKEN = /^(1-0|0-1|1\/2-1\/2|\*)$/

/** parsePgn splits a PGN file into games. A new game starts at a tag line
 * that follows move text; a file with only move text is one game. */
export function parsePgn(text: string): PgnGame[] {
  const games: PgnGame[] = []
  let headers: Record<string, string> = {}
  let moveLines: string[] = []

  const flush = () => {
    const movetext = moveLines.join(' ').trim()
    if (movetext || Object.keys(headers).length > 0) games.push(finishGame(headers, movetext))
    headers = {}
    moveLines = []
  }

  for (const raw of text.split(/\r?\n/)) {
    const line = raw.trim()
    if (!line || line.startsWith('%')) continue
    const tag = line.match(/^\[([A-Za-z0-9_]+)\s+"((?:[^"\\]|\\.)*)"\s*\]$/)
    if (tag) {
      if (moveLines.length > 0) flush()
      headers[tag[1]] = tag[2].replace(/\\(["\\])/g, '$1')
    } else {
      moveLines.push(lineCommentToBraces(line))
    }
  }
  flush()
  return games
}

/** A `;` comment runs to the end of its line. Lines are joined into one
 * move text below, so turn it into a `{...}` comment first -- otherwise it
 * would swallow every later move. A `;` inside an open `{...}` comment is
 * plain text. */
export function lineCommentToBraces(line: string): string {
  let inBrace = false
  for (let i = 0; i < line.length; i++) {
    const ch = line[i]
    if (ch === '{') inBrace = true
    else if (ch === '}') inBrace = false
    else if (ch === ';' && !inBrace) {
      const comment = line.slice(i + 1).replace(/[{}]/g, '').trim()
      return `${line.slice(0, i)}${comment ? ` {${comment}}` : ''}`
    }
  }
  return line
}

function finishGame(headers: Record<string, string>, movetext: string): PgnGame {
  const tokens = movetext.split(/\s+/)
  let trailing = ''
  if (tokens.length > 0 && RESULT_TOKEN.test(tokens[tokens.length - 1])) {
    trailing = tokens.pop()!
  }
  const cleanMovetext = tokens.join(' ').trim()
  const result = headers['Result'] && headers['Result'] !== '*' ? headers['Result'] : trailing === '*' ? '' : trailing
  return {
    headers,
    movetext: cleanMovetext,
    result,
    moves: parseMoves(cleanMovetext),
  }
}

/** parseMoves decomposes movetext into a structured list of move pairs (1. e4 e5, 2. Nf3 Nc6...). */
export function parseMoves(movetext: string): PgnMove[] {
  if (!movetext.trim()) return []

  const moves: PgnMove[] = []
  let currentNum = 1
  let currentMove: PgnMove | null = null

  // Tokenize preserving comments and variations
  const tokens: string[] = []
  let i = 0
  while (i < movetext.length) {
    const ch = movetext[i]
    if (ch === '{') {
      const end = movetext.indexOf('}', i + 1)
      const comment = end < 0 ? movetext.slice(i) : movetext.slice(i, end + 1)
      tokens.push(comment)
      i = end < 0 ? movetext.length : end + 1
    } else if (ch === '(') {
      // Find matching closing paren
      let depth = 1
      let j = i + 1
      while (j < movetext.length && depth > 0) {
        if (movetext[j] === '(') depth++
        else if (movetext[j] === ')') depth--
        j++
      }
      tokens.push(movetext.slice(i, j))
      i = j
    } else if (/\s/.test(ch)) {
      i++
    } else {
      let j = i
      while (j < movetext.length && !/\s|[{}()]/.test(movetext[j])) {
        j++
      }
      tokens.push(movetext.slice(i, j))
      i = j
    }
  }

  let expectingBlack = false

  for (const token of tokens) {
    if (token.startsWith('{') && token.endsWith('}')) {
      const comment = token.slice(1, -1).trim()
      if (currentMove) {
        if (expectingBlack) {
          currentMove.whiteComment = (currentMove.whiteComment ? currentMove.whiteComment + ' ' : '') + comment
        } else {
          currentMove.blackComment = (currentMove.blackComment ? currentMove.blackComment + ' ' : '') + comment
        }
      }
      continue
    }

    if (token.startsWith('(')) {
      // Inline variation ignored in strict table, but could attach to comment
      continue
    }

    // Check move number: e.g. "1.", "1...", "24."
    const numMatch = token.match(/^(\d+)\.+$/)
    if (numMatch) {
      const n = parseInt(numMatch[1], 10)
      if (token.includes('...')) {
        expectingBlack = true
        if (!currentMove || currentMove.num !== n) {
          currentMove = { num: n, white: '...' }
          moves.push(currentMove)
        }
      } else {
        currentNum = n
        expectingBlack = false
      }
      continue
    }

    // Check NAG or annotation symbol: $14, !, ?, !!, ??, !?, ?!
    if (token.startsWith('$') || /^(!|\?|!!|\?\?|!\?|\?!)$/.test(token)) {
      const nagCode = token.startsWith('$') ? token.slice(1) : token
      if (currentMove) {
        if (expectingBlack) {
          currentMove.whiteNag = nagCode
        } else {
          currentMove.blackNag = nagCode
        }
      }
      continue
    }

    // Result tokens
    if (RESULT_TOKEN.test(token)) {
      continue
    }

    // It's a SAN move (e.g. e4, Nf3, O-O, etc.)
    if (!expectingBlack) {
      currentMove = {
        num: currentNum,
        white: token,
      }
      moves.push(currentMove)
      expectingBlack = true
    } else {
      if (currentMove) {
        currentMove.black = token
      } else {
        currentMove = {
          num: currentNum,
          white: '...',
          black: token,
        }
        moves.push(currentMove)
      }
      expectingBlack = false
      currentNum++
    }
  }

  return moves
}

/** formatResult turns a PGN result into the book's display form. */
export function formatResult(result: string): string {
  switch (result) {
    case '1-0':
      return '1 - 0'
    case '0-1':
      return '0 - 1'
    case '1/2-1/2':
      return '½ - ½'
    default:
      return ''
  }
}

/** formatDate drops unknown PGN date parts: "2024.??.??" → "2024". */
export function formatDate(date: string | undefined): string {
  if (!date) return ''
  return date
    .split('.')
    .filter((part) => part && !/^\?+$/.test(part))
    .join('.')
}

/** movetextToTypst renders move text: moves as string literals, {comments}
 * in italics, (variations) kept inline, $n NAGs through the library's #nag. */
export function movetextToTypst(movetext: string): string {
  const out: string[] = []
  let plain = ''
  const flushPlain = () => {
    const text = plain.replace(/\s+/g, ' ')
    if (text.trim()) out.push(`#${typstString(text)}`)
    plain = ''
  }

  let i = 0
  while (i < movetext.length) {
    const ch = movetext[i]
    if (ch === '{') {
      const end = movetext.indexOf('}', i + 1)
      const comment = movetext.slice(i + 1, end < 0 ? undefined : end).trim()
      flushPlain()
      if (comment) out.push(`#emph(${typstString(comment)})`)
      plain = ' '
      i = end < 0 ? movetext.length : end + 1
    } else if (ch === ';') {
      const comment = movetext.slice(i + 1).trim()
      flushPlain()
      if (comment) out.push(`#emph(${typstString(comment)})`)
      i = movetext.length
    } else if (ch === '$') {
      const m = movetext.slice(i).match(/^\$(\d+)/)
      if (m) {
        flushPlain()
        out.push(`#nag("${m[1]}")`)
        plain = ' '
        i += m[0].length
      } else {
        plain += ch
        i++
      }
    } else {
      plain += ch
      i++
    }
  }
  flushPlain()
  return out.join('').trim()
}

/** formatMovesAsColumns renders moves into an elegant 2-column table in Typst. */
export function formatMovesAsColumns(moves: PgnMove[]): string {
  if (moves.length === 0) return ''

  const rows = moves.map((m) => {
    const wNag = m.whiteNag ? ` #nag("${m.whiteNag}")` : ''
    const wComm = m.whiteComment ? ` #emph(${typstString(m.whiteComment)})` : ''
    const wText = `[#strong[${m.white}]${wNag}${wComm}]`

    const bNag = m.blackNag ? ` #nag("${m.blackNag}")` : ''
    const bComm = m.blackComment ? ` #emph(${typstString(m.blackComment)})` : ''
    const bText = m.black ? `[#strong[${m.black}]${bNag}${bComm}]` : '[]'

    return `  [${m.num}.], ${wText}, ${bText},`
  })

  return [
    '#table(',
    '  columns: (22pt, 1fr, 1fr),',
    '  stroke: none,',
    '  inset: (x: 4pt, y: 3pt),',
    '  fill: (col, row) => if calc.odd(row) { rgb("#f8fafc") } else { none },',
    '  table.header([*\\#*], [*Trắng*], [*Đen*]),',
    ...rows,
    ')',
  ].join('\n')
}

export interface GameToTypstOptions {
  layout?: PgnLayout
  diagramFen?: string
  diagramCaption?: string
}

/** gameToTypst renders one game according to layout. */
export function gameToTypst(game: PgnGame, options: GameToTypstOptions = {}): string {
  const layout = options.layout ?? 'magazine'
  const h = game.headers
  const field = (name: string, value: string) => `  ${name}: ${typstString(value)},`
  const header = [
    '#game-header(',
    field('white', h['White'] ?? ''),
    field('white-title', h['WhiteTitle'] ?? ''),
    field('white-elo', h['WhiteElo'] ?? ''),
    field('white-fed', h['WhiteFed'] ?? h['WhiteTeam'] ?? ''),
    field('black', h['Black'] ?? ''),
    field('black-title', h['BlackTitle'] ?? ''),
    field('black-elo', h['BlackElo'] ?? ''),
    field('black-fed', h['BlackFed'] ?? h['BlackTeam'] ?? ''),
    field('event', h['Event'] && h['Event'] !== '?' ? h['Event'] : ''),
    field('site', h['Site'] && h['Site'] !== '?' ? h['Site'] : ''),
    field('date', formatDate(h['Date'])),
    field('round', h['Round'] && h['Round'] !== '?' && h['Round'] !== '-' ? h['Round'] : ''),
    field('result', formatResult(game.result)),
    field('eco', h['ECO'] ?? ''),
    field('opening', h['Opening'] ?? ''),
    ')',
  ].join('\n')

  const title = `${h['White'] ?? '?'} – ${h['Black'] ?? '?'}`
  const titleComment = `// Ván đấu: ${title.replace(/\n/g, ' ')}\n`

  if (layout === 'header_only') {
    return `${titleComment}${header}\n`
  }

  let body = ''
  if (layout === 'magazine' && game.moves && game.moves.length > 0) {
    body = formatMovesAsColumns(game.moves)
  } else if (layout === 'annotated' && options.diagramFen) {
    const movesText = game.movetext ? movetextToTypst(game.movetext) : ''
    const diagram = `#column-diagram(\n  "${options.diagramFen}",\n  caption: ${typstString(options.diagramCaption || 'Thế cờ then chốt')},\n  turn: "w"\n)`
    body = `${movesText}\n\n#v(8pt)\n${diagram}`
  } else {
    body = game.movetext ? movetextToTypst(game.movetext) : ''
  }

  return `${titleComment}${header}\n\n#v(6pt)\n\n${body}\n`
}

/** pgnToTypst converts every game in a PGN file. */
export function pgnToTypst(text: string, options: GameToTypstOptions = {}): string {
  return parsePgn(text)
    .map((g) => gameToTypst(g, options))
    .join('\n#v(12pt)\n\n')
}
