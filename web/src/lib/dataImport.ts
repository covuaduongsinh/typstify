import { CHESSBOOK_IMPORT, typstString } from './typst'

export interface ParsedPuzzle {
  fen: string
  title: string
  turn: 'w' | 'b' | 'auto'
  difficulty: number
  hint?: string
  solution?: string
}

export interface GenerateTypstOptions {
  title?: string
  subtitle?: string
  author?: string
  layout: 'a4-3x4' | '16x24-2x3'
  paperSize?: 'a4' | '16x24'
  upsideDown?: boolean
  appendix?: boolean
  embedMode?: 'direct' | 'json_reference'
  dataPath?: string
}

/**
 * parseFenList parses a raw text with one FEN per line (optionally separated by tab/comma with title/solution)
 */
export function parseFenList(text: string): ParsedPuzzle[] {
  const lines = text.split(/\r?\n/).map((l) => l.trim()).filter(Boolean)
  const puzzles: ParsedPuzzle[] = []

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    // Check if line is CSV/TSV or pure FEN
    if (line.includes('\t') || line.includes(',')) {
      const parts = line.includes('\t') ? line.split('\t') : splitCsvLine(line)
      if (parts.length > 0 && isLikelyFen(parts[0])) {
        puzzles.push({
          fen: parts[0].trim(),
          title: parts[1]?.trim() || `Bài tập ${i + 1}`,
          turn: parseTurn(parts[2] || parts[0]),
          difficulty: parseInt(parts[3], 10) || 1,
          hint: parts[4]?.trim() || undefined,
          solution: parts[5]?.trim() || undefined,
        })
        continue
      }
    }

    if (isLikelyFen(line)) {
      puzzles.push({
        fen: line,
        title: `Bài tập ${i + 1}`,
        turn: parseTurn(line),
        difficulty: 1,
      })
    }
  }

  return puzzles
}

/**
 * parseJsonPuzzles parses a JSON string (either array of objects or array of arrays)
 */
export function parseJsonPuzzles(jsonText: string): ParsedPuzzle[] {
  try {
    const data = JSON.parse(jsonText)
    if (!Array.isArray(data)) return []

    return data
      .map((item, idx) => {
        if (typeof item === 'string' && isLikelyFen(item)) {
          return {
            fen: item,
            title: `Bài tập ${idx + 1}`,
            turn: parseTurn(item),
            difficulty: 1,
          }
        }
        if (typeof item === 'object' && item !== null) {
          const fen = item.fen || item.FEN || item.Fen || ''
          if (!fen) return null
          return {
            fen,
            title: item.title || item.name || `Bài tập ${idx + 1}`,
            turn: item.turn === 'b' || item.turn === 'black' ? 'b' : item.turn === 'w' || item.turn === 'white' ? 'w' : parseTurn(fen),
            difficulty: typeof item.difficulty === 'number' ? item.difficulty : parseInt(item.difficulty, 10) || 1,
            hint: item.hint || undefined,
            solution: item.solution || item.ans || undefined,
          }
        }
        return null
      })
      .filter((p): p is ParsedPuzzle => p !== null)
  } catch {
    return []
  }
}

/**
 * parseCsvPuzzles parses CSV text into structured puzzles
 */
export function parseCsvPuzzles(csvText: string): ParsedPuzzle[] {
  const lines = csvText.split(/\r?\n/).map((l) => l.trim()).filter(Boolean)
  if (lines.length === 0) return []

  const firstLine = lines[0].toLowerCase()
  const hasHeader = firstLine.includes('fen')

  const startIdx = hasHeader ? 1 : 0
  const puzzles: ParsedPuzzle[] = []

  for (let i = startIdx; i < lines.length; i++) {
    const cols = splitCsvLine(lines[i])
    if (cols.length === 0) continue

    const fen = cols[0]?.trim()
    if (!fen || !isLikelyFen(fen)) continue

    puzzles.push({
      fen,
      title: cols[1]?.trim() || `Bài tập ${puzzles.length + 1}`,
      turn: parseTurn(cols[2] || fen),
      difficulty: parseInt(cols[3], 10) || 1,
      hint: cols[4]?.trim() || undefined,
      solution: cols[5]?.trim() || undefined,
    })
  }

  return puzzles
}

function splitCsvLine(line: string): string[] {
  const result: string[] = []
  let current = ''
  let inQuotes = false

  for (let i = 0; i < line.length; i++) {
    const char = line[i]
    if (char === '"') {
      inQuotes = !inQuotes
    } else if (char === ',' && !inQuotes) {
      result.push(current)
      current = ''
    } else {
      current += char
    }
  }
  result.push(current)
  return result.map((c) => c.replace(/^"|"$/g, '').replace(/""/g, '"'))
}

export function isLikelyFen(str: string): boolean {
  if (!str || typeof str !== 'string') return false
  const trimmed = str.trim()
  const parts = trimmed.split(' ')
  // Standard FEN has 6 space-separated fields, but board-only position has 8 slash-separated rows
  const boardPart = parts[0]
  const rows = boardPart.split('/')
  return rows.length === 8 && /^[rnbqkpRNBQKP1-8]+$/.test(rows[0])
}

function parseTurn(str: string): 'w' | 'b' | 'auto' {
  if (!str) return 'auto'
  const lower = str.toLowerCase().trim()
  if (lower === 'w' || lower === 'white' || lower === 'trắng') return 'w'
  if (lower === 'b' || lower === 'black' || lower === 'đen') return 'b'
  // Try extracting from full FEN (2nd token)
  const parts = str.trim().split(' ')
  if (parts.length >= 2) {
    if (parts[1] === 'w') return 'w'
    if (parts[1] === 'b') return 'b'
  }
  return 'auto'
}

/**
 * generatePuzzleCollectionTypst builds complete Typst code for a collection of puzzles
 */
export function generatePuzzleCollectionTypst(
  puzzles: ParsedPuzzle[],
  options: GenerateTypstOptions,
): string {
  const {
    title = 'BỘ BÀI TẬP CỜ VUA',
    subtitle = '',
    author = 'CLB Cờ vua Dương Sinh',
    layout = 'a4-3x4',
    upsideDown = true,
    appendix = true,
    embedMode = 'direct',
    dataPath = 'data/puzzles.json',
  } = options

  const is16x24 = layout === '16x24-2x3'
  const initFunc = is16x24
    ? `#show: chess-book-init.with(
  title: ${typstString(title)},
  subtitle: ${typstString(subtitle)},
  author: ${typstString(author)},
  paper-size: "16x24",
)`
    : `#show: chess-worksheet-init.with(
  title: ${typstString(title)},
  subtitle: ${typstString(subtitle)},
  author: ${typstString(author)},
  paper-size: "a4",
)`

  if (embedMode === 'json_reference') {
    return `${CHESSBOOK_IMPORT}

${initFunc}

#let puzzle-data = json(${typstString(dataPath)})

#render-puzzle-collection(
  puzzle-data,
  layout: "${layout}",
  upside-down-solutions: ${upsideDown ? 'true' : 'false'},
  render-appendix-at-end: ${appendix ? 'true' : 'false'},
)
`
  }

  // Direct embed: render array of dictionary records
  const puzzleItemsStr = puzzles
    .map((p, idx) => {
      const fields = [
        `fen: ${typstString(p.fen)}`,
        `number: ${idx + 1}`,
        `title: ${typstString(p.title)}`,
        `turn: ${typstString(p.turn)}`,
        `difficulty: ${p.difficulty}`,
      ]
      if (p.hint) fields.push(`hint: ${typstString(p.hint)}`)
      if (p.solution) fields.push(`solution: ${typstString(p.solution)}`)
      return `    (${fields.join(', ')}),`
    })
    .join('\n')

  return `${CHESSBOOK_IMPORT}

${initFunc}

#render-puzzle-collection(
  (
${puzzleItemsStr}
  ),
  layout: "${layout}",
  upside-down-solutions: ${upsideDown ? 'true' : 'false'},
  render-appendix-at-end: ${appendix ? 'true' : 'false'},
)
`
}
