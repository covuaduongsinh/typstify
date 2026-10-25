// Typst's sym.* table (server/symbols_api.go, from tinymist).
export interface TypstSymbol {
  name: string
  char: string
  category?: string
}

const norm = (s: string) => s.toLowerCase().replace(/^#?sym\./, '')

/** Symbols matching a query: the character itself, a name prefix, then a
 * name or category substring, at most `limit` of them. */
export function filterSymbols(list: TypstSymbol[], query: string, limit = 400): TypstSymbol[] {
  const q = norm(query.trim())
  if (!q) return list.slice(0, limit)
  const exact: TypstSymbol[] = []
  const prefix: TypstSymbol[] = []
  const rest: TypstSymbol[] = []
  for (const s of list) {
    const name = s.name.toLowerCase()
    if (s.char === query.trim()) exact.push(s)
    else if (name.startsWith(q)) prefix.push(s)
    else if (name.includes(q) || (s.category ?? '').toLowerCase().includes(q)) rest.push(s)
  }
  return [...exact, ...prefix, ...rest].slice(0, limit)
}
