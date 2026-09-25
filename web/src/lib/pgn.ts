// PGN (Portable Game Notation) → Typst for the chessbook library.
//
// Move text is emitted as Typst string literals (#"...") rather than raw
// markup: PGN is full of characters that mean something in Typst markup
// (`*` result, `$n` NAGs, `...` black move numbers, a leading `1.` that
// would start a numbered list), and a string literal renders them verbatim.
import { typstString } from './typst'

export interface PgnGame {
  headers: Record<string, string>
  /** Move text with the result token removed. */
  movetext: string
  /** Result from the Result tag, else the trailing result token, else ''. */
  result: string
}

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
  const result = headers['Result'] && headers['Result'] !== '*' ? headers['Result'] : trailing === '*' ? '' : trailing
  return { headers, movetext: tokens.join(' ').trim(), result }
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
      // rest-of-line comment; lines were already joined, so it runs to the end
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

/** gameToTypst renders one game: a #game-header card plus its moves. Tags
 * missing from the PGN are left empty -- never filled with placeholder
 * data, which would end up printed as if it were real. */
export function gameToTypst(game: PgnGame): string {
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
  const moves = game.movetext ? movetextToTypst(game.movetext) : ''
  return `// Ván đấu: ${title.replace(/\n/g, ' ')}\n${header}\n\n#v(6pt)\n\n${moves}\n`
}

/** pgnToTypst converts every game in a PGN file. */
export function pgnToTypst(text: string): string {
  return parsePgn(text).map(gameToTypst).join('\n#v(12pt)\n\n')
}
