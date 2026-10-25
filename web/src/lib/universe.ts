// Typst Universe (server/universe_api.go, service/universe).

export interface UniverseTemplate {
  path: string
  entrypoint: string
  thumbnail?: string
}

/** A package at its latest version, from packages.typst.org's index. */
export interface UniversePackage {
  name: string
  version: string
  description?: string
  authors?: string[]
  license?: string
  repository?: string
  homepage?: string
  keywords?: string[]
  categories?: string[]
  disciplines?: string[]
  compiler?: string
  template?: UniverseTemplate
  updatedAt?: number
}

export interface UniversePage {
  packages: UniversePackage[]
  total: number
}

export const universeThumbnailUrl = (p: { name: string; version: string }, small = true) =>
  `/api/universe/thumbnail/${encodeURIComponent(p.name)}/${encodeURIComponent(p.version)}${small ? '?size=small' : ''}`

export const importLine = (name: string, version: string) => `#import "@preview/${name}:${version}": *`

const escapeRe = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

/**
 * Where to add `#import "@preview/name:version": *` to doc: after the
 * imports, blank lines and comments the file starts with (so a chessbook
 * import stays on line 1). Null when the package is already imported, in
 * any version.
 */
export function importInsertion(doc: string, name: string, version: string): { at: number; text: string } | null {
  // Only a real import counts, not one inside a // comment.
  if (new RegExp(`^[ \\t]*#import\\s+"@preview/${escapeRe(name)}:`, 'm').test(doc)) return null
  // No import yet: the very top. Otherwise right below the last one. An
  // import whose item list spans lines ends where its parentheses close.
  let at = 0
  let depth = 0
  for (let pos = 0; pos < doc.length; ) {
    const nl = doc.indexOf('\n', pos)
    const end = nl === -1 ? doc.length : nl + 1
    const line = doc.slice(pos, end).trim()
    if (depth > 0 || line.startsWith('#import')) {
      for (const ch of line.replace(/"(?:[^"\\]|\\.)*"/g, '')) {
        if (ch === '(') depth++
        else if (ch === ')') depth = Math.max(0, depth - 1)
      }
      if (depth === 0) at = end
    } else if (line !== '' && !line.startsWith('//')) break
    pos = end
  }
  // An import left open runs to the end: the top is the only safe place.
  if (depth > 0) at = 0
  const needsBreak = at > 0 && !doc.slice(0, at).endsWith('\n')
  return { at, text: `${needsBreak ? '\n' : ''}${importLine(name, version)}\n` }
}
