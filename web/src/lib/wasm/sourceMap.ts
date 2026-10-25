// Preview -> source jumps for the in-browser compiler. typst.ts does not
// expose span resolution, so the clicked text run (the selectable `.tsel`
// layer the renderer emits) is searched for in the source, and the match
// nearest the line estimated from the click's page position wins.

const norm = (s: string) => s.normalize('NFC').toLowerCase()

function lineStarts(source: string): number[] {
  const starts = [0]
  for (let i = 0; i < source.length; i++) if (source.charCodeAt(i) === 10) starts.push(i + 1)
  return starts
}

/** 1-based line of a character offset. */
function lineOf(starts: number[], offset: number): number {
  let lo = 0
  let hi = starts.length - 1
  while (lo < hi) {
    const mid = (lo + hi + 1) >> 1
    if (starts[mid] <= offset) lo = mid
    else hi = mid - 1
  }
  return lo + 1
}

function occurrences(hay: string, needle: string, limit = 400): number[] {
  const out: number[] = []
  if (!needle) return out
  let i = hay.indexOf(needle)
  while (i !== -1 && out.length < limit) {
    out.push(i)
    i = hay.indexOf(needle, i + 1)
  }
  return out
}

const nearest = (lines: number[], estimate: number) =>
  lines.reduce((best, l) => (Math.abs(l - estimate) < Math.abs(best - estimate) ? l : best), lines[0])

/**
 * findSourceLine returns the 1-based source line that most likely produced
 * `text`, or null when it cannot be located. `estimate` (1-based) breaks
 * ties between repeated phrases.
 */
export function findSourceLine(source: string, text: string, estimate: number): number | null {
  const needle = norm(text.replace(/\s+/g, ' ').trim())
  if (needle.length < 2) return null
  const hay = norm(source)
  const starts = lineStarts(source)

  const exact = occurrences(hay, needle).map((o) => lineOf(starts, o))
  if (exact.length) return nearest(exact, estimate)

  // Markup splits the run in the source (*bold*, #emph[...], line breaks):
  // score lines by how many of the run's words they contain.
  const words = [...new Set(needle.split(/[^\p{L}\p{N}]+/u).filter((w) => w.length >= 3))]
  if (!words.length) return null
  const score = new Map<number, number>()
  for (const w of words) {
    for (const line of new Set(occurrences(hay, w).map((o) => lineOf(starts, o)))) {
      score.set(line, (score.get(line) ?? 0) + w.length)
    }
  }
  if (!score.size) return null
  const top = Math.max(...score.values())
  const best = [...score].filter(([, s]) => s === top).map(([l]) => l)
  return nearest(best, estimate)
}
