// Markdown to Typst converter specialized for Chess publishing.
// Supports standard Markdown elements plus FEN diagrams, PGN games,
// puzzle cards/grids, ECO headers, lesson plans, callouts, and NAG annotations.

import { isLikelyFen, parseFenList, parseJsonPuzzles } from './dataImport'
import { gameToTypst, parsePgn } from './pgn'
import { repairChessImports, typstString } from './typst'

export interface MarkdownToTypstOptions {
  /** Target layout/paper template: 'book' (16x24), 'worksheet' (a4), 'magazine' (a4), 'article' (a5), or 'none' */
  template?: 'book' | 'worksheet' | 'magazine' | 'article' | 'none'
  title?: string
  subtitle?: string
  author?: string
  date?: string
  /** Auto-detect and transform FEN/PGN/Chess blocks */
  enableChessFeatures?: boolean
  /** Auto-convert NAG notations (e.g. $14, (±), !?, ??) to #nag(...) */
  convertNags?: boolean
}

export interface FrontmatterMeta {
  title?: string
  subtitle?: string
  author?: string
  date?: string
  layout?: string
  paperSize?: string
  [key: string]: string | undefined
}

/**
 * parseFrontmatter extracts YAML frontmatter from markdown text if present
 */
export function parseFrontmatter(markdown: string): { meta: FrontmatterMeta; content: string } {
  const trimmed = markdown.trimStart()
  if (!trimmed.startsWith('---')) {
    return { meta: {}, content: markdown }
  }

  const endIdx = trimmed.indexOf('\n---', 3)
  if (endIdx === -1) {
    return { meta: {}, content: markdown }
  }

  const yamlBlock = trimmed.slice(3, endIdx).trim()
  const content = trimmed.slice(endIdx + 4).replace(/^\r?\n/, '')

  const meta: FrontmatterMeta = {}
  for (const line of yamlBlock.split(/\r?\n/)) {
    const colonIdx = line.indexOf(':')
    if (colonIdx > 0) {
      const key = line.slice(0, colonIdx).trim()
      let val = line.slice(colonIdx + 1).trim()
      if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
        val = val.slice(1, -1)
      }
      meta[key] = val
    }
  }

  return { meta, content }
}

/**
 * convertMarkdownToTypst converts Markdown text to Typst with chessbook enhancements
 */
export function convertMarkdownToTypst(markdown: string, options: MarkdownToTypstOptions = {}): string {
  const { meta, content } = parseFrontmatter(markdown)

  const template =
    options.template ??
    (meta.layout === 'worksheet' || meta.paperSize === 'a4-worksheet'
      ? 'worksheet'
      : meta.layout === 'magazine'
        ? 'magazine'
        : meta.layout === 'article' || meta.paperSize === 'a5'
          ? 'article'
          : meta.layout === 'book' || meta.paperSize === '16x24'
            ? 'book'
            : meta.title
              ? 'book'
              : 'none')

  const title = options.title ?? meta.title
  const subtitle = options.subtitle ?? meta.subtitle ?? ''
  const author = options.author ?? meta.author ?? 'CLB Cờ vua Dương Sinh'
  const date = options.date ?? meta.date ?? ''
  const enableChess = options.enableChessFeatures ?? true
  const convertNags = options.convertNags ?? true

  const lines = content.split(/\r?\n/)
  const outLines: string[] = []

  let i = 0
  while (i < lines.length) {
    const line = lines[i]

    // 1. Code blocks (``` or ~~~)
    if (line.trim().startsWith('```') || line.trim().startsWith('~~~')) {
      const fenceMatch = line.trim().match(/^(`{3,}|~{3,})\s*(.*)$/)
      const fence = fenceMatch ? fenceMatch[1] : '```'
      const info = fenceMatch ? fenceMatch[2].trim() : ''

      const codeLines: string[] = []
      i++
      while (i < lines.length && !lines[i].trim().startsWith(fence)) {
        codeLines.push(lines[i])
        i++
      }
      i++ // Skip closing fence

      const codeContent = codeLines.join('\n')
      const lang = info.toLowerCase().split(/\s+/)[0]

      if (enableChess && isChessFence(lang)) {
        outLines.push(renderChessCodeBlock(lang, info, codeContent))
      } else if (lang === 'typst' || lang === 'typ') {
        // Raw Typst code embedded in markdown
        outLines.push(codeContent)
      } else {
        // Regular markdown code fence -> Typst code block
        outLines.push(`\`\`\`${lang}\n${codeContent}\n\`\`\``)
      }
      continue
    }

    // 2. Markdown Tables
    if (isTableStart(lines, i)) {
      const { tableTypst, nextIdx } = renderMarkdownTable(lines, i)
      outLines.push(tableTypst)
      i = nextIdx
      continue
    }

    // 3. Blockquotes / Callouts
    if (line.trim().startsWith('>')) {
      const quoteLines: string[] = []
      while (i < lines.length && lines[i].trim().startsWith('>')) {
        quoteLines.push(lines[i].replace(/^\s*>\s?/, ''))
        i++
      }
      outLines.push(renderBlockquote(quoteLines, convertNags))
      continue
    }

    // 4. Headings (# Heading)
    const headingMatch = line.match(/^(#{1,6})\s+(.*)$/)
    if (headingMatch) {
      const level = headingMatch[1].length
      const headingText = transformInlineFormatting(headingMatch[2], convertNags)
      outLines.push(`${'='.repeat(level)} ${headingText}`)
      i++
      continue
    }

    // 5. Horizontal rule
    if (/^(?:---|\*\*\*|___)\s*$/.test(line.trim())) {
      outLines.push('#v(8pt)\n#line(length: 100%, stroke: 0.5pt + luma(200))\n#v(8pt)')
      i++
      continue
    }

    // 6. Lists
    const listMatch = line.match(/^(\s*)([-*+]|\d+\.)\s+(.*)$/)
    if (listMatch) {
      const indent = listMatch[1]
      const bullet = listMatch[2]
      const text = transformInlineFormatting(listMatch[3], convertNags)
      const isNumbered = /^\d+\./.test(bullet)
      const prefix = isNumbered ? '+' : '-'
      outLines.push(`${indent}${prefix} ${text}`)
      i++
      continue
    }

    // 7. Check if line is a standalone raw FEN string
    if (enableChess && isLikelyFen(line.trim())) {
      outLines.push(
        `#teaching-diagram(\n  "${line.trim()}",\n  title: "Thế cờ",\n  turn: "${line.trim().split(' ')[1] || 'w'}"\n)`,
      )
      i++
      continue
    }

    // 8. Regular paragraph text
    if (line.trim() === '') {
      outLines.push('')
    } else {
      outLines.push(transformInlineFormatting(line, convertNags))
    }
    i++
  }

  let body = outLines.join('\n')

  // Prepend document initialization template if requested
  let header = ''
  if (template === 'book' && title) {
    header = `#show: chess-book-init.with(
  title: ${typstString(title)},
  subtitle: ${typstString(subtitle)},
  author: ${typstString(author)},
  paper-size: "16x24",
)\n\n`
  } else if (template === 'worksheet' && title) {
    header = `#show: chess-worksheet-init.with(
  title: ${typstString(title)},
  subtitle: ${typstString(subtitle)},
  author: ${typstString(author)},
  date: ${typstString(date)},
  paper-size: "a4",
)\n\n`
  } else if (template === 'magazine' && title) {
    header = `#show: chess-magazine-init.with(
  magazine-title: ${typstString(title)},
  issue: ${typstString(subtitle || 'Số 1')},
)\n\n`
  } else if (template === 'article' && title) {
    header = `#show: chess-book-init.with(
  title: ${typstString(title)},
  subtitle: ${typstString(subtitle)},
  author: ${typstString(author)},
  paper-size: "a5",
)\n\n`
  }

  const fullDoc = header + body

  // Ensure line 1 has #import "@local/chessbook:0.1.0": * if chess elements are present
  return repairChessImports(fullDoc)
}

function isChessFence(lang: string): boolean {
  const clean = lang.toLowerCase()
  return (
    clean === 'fen' ||
    clean === 'chess-fen' ||
    clean === 'diagram' ||
    clean === 'board' ||
    clean === 'pgn' ||
    clean === 'chess-pgn' ||
    clean === 'chess' ||
    clean === 'puzzle' ||
    clean === 'puzzles' ||
    clean === 'tactics' ||
    clean === 'eco' ||
    clean === 'lesson'
  )
}

function renderChessCodeBlock(lang: string, fullInfo: string, content: string): string {
  const cleanLang = lang.toLowerCase()

  // 1. FEN / Diagram / Board
  if (cleanLang === 'fen' || cleanLang === 'chess-fen' || cleanLang === 'diagram' || cleanLang === 'board') {
    return renderFenBlock(fullInfo, content)
  }

  // 2. PGN Games
  if (cleanLang === 'pgn' || cleanLang === 'chess-pgn' || cleanLang === 'chess') {
    const games = parsePgn(content)
    if (games.length > 0) {
      return games.map((g) => gameToTypst(g, { layout: 'magazine' })).join('\n#v(12pt)\n\n')
    }
  }

  // 3. Puzzle & Tactics
  if (cleanLang === 'puzzle' || cleanLang === 'puzzles' || cleanLang === 'tactics') {
    return renderPuzzleBlock(content)
  }

  // 4. ECO Header
  if (cleanLang === 'eco') {
    return renderEcoBlock(content)
  }

  // 5. Lesson Plan Header
  if (cleanLang === 'lesson') {
    return renderLessonBlock(content)
  }

  return `\`\`\`${lang}\n${content}\n\`\`\``
}

function renderFenBlock(_info: string, content: string): string {
  const lines = content.split(/\r?\n/).map((l) => l.trim()).filter(Boolean)
  if (lines.length === 0) return ''

  let fen = ''
  let title = 'Thế cờ'
  let turn = 'w'
  let caption = ''
  let arrows: string[] = []
  let evalText = ''

  // Look for FEN in lines or key-values
  for (const line of lines) {
    if (isLikelyFen(line)) {
      fen = line
      const parts = line.split(' ')
      if (parts.length >= 2 && (parts[1] === 'w' || parts[1] === 'b')) {
        turn = parts[1]
      }
      continue
    }

    const colonIdx = line.indexOf(':')
    if (colonIdx > 0) {
      const key = line.slice(0, colonIdx).trim().toLowerCase()
      let val = line.slice(colonIdx + 1).trim()
      if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
        val = val.slice(1, -1)
      }

      if (key === 'fen') fen = val
      else if (key === 'title') title = val
      else if (key === 'turn') turn = val === 'b' || val === 'black' || val === 'đen' ? 'b' : 'w'
      else if (key === 'caption') caption = val
      else if (key === 'eval') evalText = val
      else if (key === 'arrows' || key === 'arrow') {
        arrows = val.split(/[,;\s]+/).filter(Boolean)
      }
    }
  }

  if (!fen && lines.length > 0 && isLikelyFen(lines[0])) {
    fen = lines[0]
  }

  if (!fen) {
    fen = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'
  }

  if (evalText) {
    return `#opening-diagram-box(
  ${typstString(fen)},
  title: ${typstString(title)},
  turn: "${turn}",
  eval-text: ${typstString(evalText)},
  caption: ${typstString(caption)}
)`
  }

  const arrowParam =
    arrows.length > 0 ? `,\n  arrows: (${arrows.map((a) => typstString(a)).join(', ')}${arrows.length === 1 ? ',' : ''})` : ''

  return `#teaching-diagram(
  ${typstString(fen)},
  title: ${typstString(title)},
  turn: "${turn}",
  size: 16pt,
  caption: ${typstString(caption)}${arrowParam}
)`
}

function renderPuzzleBlock(content: string): string {
  const trimmed = content.trim()
  let puzzles = parseJsonPuzzles(trimmed)
  if (puzzles.length === 0) {
    puzzles = parseFenList(trimmed)
  }

  if (puzzles.length === 0) return ''

  if (puzzles.length === 1) {
    const p = puzzles[0]
    return `#puzzle-card(
  ${typstString(p.fen)},
  number: 1,
  title: ${typstString(p.title)},
  turn: "${p.turn === 'auto' ? 'w' : p.turn}",
  difficulty: ${p.difficulty}${p.hint ? `,\n  hint: ${typstString(p.hint)}` : ''}${p.solution ? `,\n  solution: ${typstString(p.solution)}` : ''}
)`
  }

  if (puzzles.length <= 6) {
    const items = puzzles
      .map(
        (p, idx) =>
          `    (fen: ${typstString(p.fen)}, number: ${idx + 1}, title: ${typstString(p.title)}, turn: "${p.turn === 'auto' ? 'w' : p.turn}", difficulty: ${p.difficulty}${p.hint ? `, hint: ${typstString(p.hint)}` : ''}${p.solution ? `, solution: ${typstString(p.solution)}` : ''}),`,
      )
      .join('\n')
    return `#puzzle-grid-16x24(
  puzzles: (
${items}
  )
)`
  }

  // 7-12 puzzles -> A4 grid
  const items = puzzles
    .slice(0, 12)
    .map(
      (p, idx) =>
        `    (fen: ${typstString(p.fen)}, number: ${idx + 1}, title: ${typstString(p.title)}, turn: "${p.turn === 'auto' ? 'w' : p.turn}", difficulty: ${p.difficulty}${p.hint ? `, hint: ${typstString(p.hint)}` : ''}${p.solution ? `, solution: ${typstString(p.solution)}` : ''}),`,
    )
    .join('\n')
  return `#puzzle-grid-a4(
  puzzles: (
${items}
  )
)`
}

function renderEcoBlock(content: string): string {
  const lines = content.split(/\r?\n/)
  let code = 'C 58'
  let name = ''
  let subname = ''
  let introMoves = ''

  for (const line of lines) {
    const colonIdx = line.indexOf(':')
    if (colonIdx > 0) {
      const key = line.slice(0, colonIdx).trim().toLowerCase()
      let val = line.slice(colonIdx + 1).trim()
      if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
        val = val.slice(1, -1)
      }
      if (key === 'code' || key === 'eco') code = val
      else if (key === 'name' || key === 'title') name = val
      else if (key === 'subname' || key === 'variant') subname = val
      else if (key === 'intro-moves' || key === 'moves' || key === 'intromoves') introMoves = val
    }
  }

  return `#eco-header(
  code: ${typstString(code)},
  name: ${typstString(name)},
  subname: ${typstString(subname)},
  intro-moves: ${typstString(introMoves)}
)`
}

function renderLessonBlock(content: string): string {
  const lines = content.split(/\r?\n/)
  let lessonNum = 1
  let title = 'Bài giảng'
  let level = 'Cơ bản'
  let duration = '45 phút'
  let objective = ''

  for (const line of lines) {
    const colonIdx = line.indexOf(':')
    if (colonIdx > 0) {
      const key = line.slice(0, colonIdx).trim().toLowerCase()
      let val = line.slice(colonIdx + 1).trim()
      if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
        val = val.slice(1, -1)
      }
      if (key === 'lesson-num' || key === 'number' || key === 'lesson') lessonNum = parseInt(val, 10) || 1
      else if (key === 'title') title = val
      else if (key === 'level') level = val
      else if (key === 'duration') duration = val
      else if (key === 'objective' || key === 'goal') objective = val
    }
  }

  return `#lesson-header(
  lesson-num: ${lessonNum},
  title: ${typstString(title)},
  level: ${typstString(level)},
  duration: ${typstString(duration)},
  objective: ${typstString(objective)}
)`
}

function renderBlockquote(lines: string[], convertNags: boolean): string {
  if (lines.length === 0) return ''
  const firstLine = lines[0].trim()

  // GitHub / Markdown Callouts
  const calloutMatch = firstLine.match(/^\[!(NOTE|INFO|TIP|IMPORTANT|WARNING|CAUTION|CONCEPT|THEORY|KEY|QUOTE)\](?:\s*(.*))?$/i)
  if (calloutMatch) {
    const type = calloutMatch[1].toUpperCase()
    const customTitle = calloutMatch[2]?.trim()
    const bodyLines = lines.slice(1).map((l) => transformInlineFormatting(l, convertNags)).join('\n')

    if (type === 'CONCEPT' || type === 'THEORY' || type === 'KEY') {
      return `#concept-box(title: ${typstString(customTitle || 'Khái niệm Then chốt')})[\n${bodyLines}\n]`
    }
    if (type === 'NOTE' || type === 'INFO' || type === 'TIP') {
      return `#instructor-note[\n${bodyLines}\n]`
    }
    if (type === 'IMPORTANT' || type === 'WARNING' || type === 'CAUTION') {
      return `#concept-box(title: ${typstString(customTitle || 'Lưu ý Quan trọng')})[\n${bodyLines}\n]`
    }
    if (type === 'QUOTE') {
      return `#chess-quote(author: ${typstString(customTitle || '')})[\n${bodyLines}\n]`
    }
  }

  // Quote with author attribution: `> "quote" -- Author`
  const fullText = lines.join('\n')
  const authorMatch = fullText.match(/^([\s\S]+?)(?:\r?\n\s*)?(?:--|—)\s*([^\n\r]+)$/)
  if (authorMatch) {
    const quoteBody = transformInlineFormatting(authorMatch[1].trim().replace(/^["“]|["”]$/g, ''), convertNags)
    const author = authorMatch[2].trim()
    return `#chess-quote(author: ${typstString(author)})[\n${quoteBody}\n]`
  }

  const content = lines.map((l) => transformInlineFormatting(l, convertNags)).join('\n')
  return `#quote[\n${content}\n]`
}

function isTableStart(lines: string[], idx: number): boolean {
  if (idx + 1 >= lines.length) return false
  const line1 = lines[idx].trim()
  const line2 = lines[idx + 1].trim()
  return line1.includes('|') && /^\|?[\s:-]+(?:\|[\s:-]+)+\|?$/.test(line2)
}

function renderMarkdownTable(lines: string[], startIdx: number): { tableTypst: string; nextIdx: number } {
  const headerLine = lines[startIdx].trim()
  const headers = splitTableRow(headerLine)
  const colCount = headers.length

  let i = startIdx + 2 // Skip separator line
  const rows: string[][] = []

  while (i < lines.length && lines[i].trim().includes('|')) {
    const row = splitTableRow(lines[i].trim())
    // Pad or slice to colCount
    while (row.length < colCount) row.push('')
    rows.push(row.slice(0, colCount))
    i++
  }

  const columnsDef = `(${Array(colCount).fill('1fr').join(', ')})`
  // In Typst, table.header(...) takes the headers
  const headerArgs = headers.map((h) => `[*${escapeTableText(h.trim())}*]`).join(', ')
  const rowCells: string[] = []

  for (const r of rows) {
    const cells = r.map((c) => `  [${transformInlineFormatting(c.trim(), true)}]`).join(', ')
    rowCells.push(`  ${cells},`)
  }

  const typstTable = `#table(
  columns: ${columnsDef},
  stroke: 0.5pt + luma(200),
  inset: (x: 6pt, y: 5pt),
  fill: (col, row) => if row == 0 { rgb("#f1f5f9") } else if calc.odd(row) { rgb("#fafbfc") } else { none },
  table.header(${headerArgs}),
${rowCells.join('\n')}
)`

  return { tableTypst: typstTable, nextIdx: i }
}

function splitTableRow(row: string): string[] {
  let cleaned = row
  if (cleaned.startsWith('|')) cleaned = cleaned.slice(1)
  if (cleaned.endsWith('|')) cleaned = cleaned.slice(0, -1)
  return cleaned.split('|')
}

function escapeTableText(text: string): string {
  // In table headers/cells, literal '#' must be escaped as '\#'
  return text.replace(/#/g, '\\#')
}

/**
 * transformInlineFormatting converts bold, italic, inline code, links, images, figurines, and NAGs
 */
export function transformInlineFormatting(text: string, convertNags = true): string {
  let res = text

  // 1. Inline code `code` (preserve without formatting inside)
  const codeSpans: string[] = []
  res = res.replace(/`([^`]+)`/g, (_, code) => {
    codeSpans.push(`\`${code}\``)
    return `__CODE_SPAN_${codeSpans.length - 1}__`
  })

  // 2. Images: ![alt](url)
  res = res.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, (_, _alt, url) => {
    return `#image(${typstString(url)})`
  })

  // 3. Links: [text](url)
  res = res.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_, linkText, url) => {
    return `#link(${typstString(url)})[${linkText}]`
  })

  // 4. Strikethrough: ~~text~~
  res = res.replace(/~~([^~]+)~~/g, '#strike[$1]')

  // 5. Bold & Italic: ***text*** or ___text___
  const biMatches: string[] = []
  res = res.replace(/(?:\*\*\*|___)([\s\S]+?)(?:\*\*\*|___)/g, (_, text) => {
    biMatches.push(text)
    return `__TYP_BI_${biMatches.length - 1}__`
  })

  // 6. Bold: **text** or __text__
  const boldMatches: string[] = []
  res = res.replace(/(?:\*\*|__)([\s\S]+?)(?:\*\*|__)/g, (_, text) => {
    boldMatches.push(text)
    return `__TYP_B_${boldMatches.length - 1}__`
  })

  // 7. Italic: *text* or _text_
  const italicMatches: string[] = []
  res = res.replace(/(?:(?<=\s|^|[^\w*])\*([^*\n]+?)\*(?=\s|$|[^\w*])|(?<=\s|^|[^\w_])_([^_\n]+?)_(?=\s|$|[^\w_]))/g, (_, t1, t2) => {
    const text = t1 || t2
    italicMatches.push(text)
    return `__TYP_I_${italicMatches.length - 1}__`
  })

  // Restore Bold/Italic to Typst syntax (*bold*, _italic_, *_bold italic_*)
  res = res.replace(/__TYP_BI_(\d+)__/g, (_, idx) => `*_${biMatches[parseInt(idx, 10)]}_*`)
  res = res.replace(/__TYP_B_(\d+)__/g, (_, idx) => `*${boldMatches[parseInt(idx, 10)]}*`)
  res = res.replace(/__TYP_I_(\d+)__/g, (_, idx) => `_${italicMatches[parseInt(idx, 10)]}_`)

  // 8. Figurine shortcodes: :wK:, :wQ:, :wR:, :wB:, :wN:, :wP:, :bK:, :bQ:, :bR:, :bB:, :bN:, :bP:
  res = res.replace(/:w([KQBNRP]):/g, '#w$1')
  res = res.replace(/:b([KQBNRP]):/g, '#b$1')

  // 9. Chess NAG annotations
  if (convertNags) {
    res = res.replace(/\$14\b|\(⩲\)/g, '#nag("14")')
    res = res.replace(/\$15\b|\(⩱\)/g, '#nag("15")')
    res = res.replace(/\$16\b|\(±\)/g, '#nag("16")')
    res = res.replace(/\$17\b|\(∓\)/g, '#nag("17")')
    res = res.replace(/\$10\b|\(=\)/g, '#nag("10")')
    res = res.replace(/\$13\b|\(∞\)/g, '#nag("13")')
    res = res.replace(/\$7\b|\(□\)/g, '#nag("7")')
    res = res.replace(/\$1\b/g, '#nag("1")')
    res = res.replace(/\$2\b/g, '#nag("2")')
    res = res.replace(/\$3\b/g, '#nag("3")')
    res = res.replace(/\$4\b/g, '#nag("4")')
  }

  // Restore code spans
  res = res.replace(/__CODE_SPAN_(\d+)__/g, (_, idx) => {
    return codeSpans[parseInt(idx, 10)] || ''
  })

  return res
}
