import type * as Y from 'yjs'

const isHighSurrogate = (c: number) => c >= 0xd800 && c <= 0xdbff
const isLowSurrogate = (c: number) => c >= 0xdc00 && c <= 0xdfff

/**
 * The smallest single replacement turning `prev` into `next`: the common
 * prefix and suffix are kept. Boundaries never split a surrogate pair, so
 * an emoji is replaced whole.
 */
export function minimalChange(prev: string, next: string): { from: number; to: number; insert: string } | null {
  if (prev === next) return null
  let start = 0
  const max = Math.min(prev.length, next.length)
  while (start < max && prev.charCodeAt(start) === next.charCodeAt(start)) start++
  if (start > 0 && isHighSurrogate(prev.charCodeAt(start - 1))) start--

  let endPrev = prev.length
  let endNext = next.length
  while (endPrev > start && endNext > start && prev.charCodeAt(endPrev - 1) === next.charCodeAt(endNext - 1)) {
    endPrev--
    endNext--
  }
  if (endPrev < prev.length && isLowSurrogate(prev.charCodeAt(endPrev))) {
    endPrev++
    endNext++
  }
  return { from: start, to: endPrev, insert: next.slice(start, endNext) }
}

/** Rewrites `ytext` to `next` with a minimal edit, so collaborators'
 * cursors and comment anchors outside the changed span stay put. */
export function applyTextDiff(ytext: Y.Text, next: string) {
  const change = minimalChange(ytext.toString(), next)
  if (!change) return
  ytext.doc?.transact(() => {
    if (change.to > change.from) ytext.delete(change.from, change.to - change.from)
    if (change.insert) ytext.insert(change.from, change.insert)
  })
}

export type Rebase =
  | { kind: 'none' }
  | { kind: 'apply'; from: number; to: number; insert: string }
  | { kind: 'conflict' }

/**
 * rebaseChange replays an outside edit of the file (base -> disk) on top
 * of the shared text `cur`, which may hold edits that never reached the
 * disk. When the outside edit and the local ones touch the same span the
 * result is a conflict for the user to settle, never a silent overwrite.
 */
export function rebaseChange(base: string, disk: string, cur: string): Rebase {
  const outside = minimalChange(base, disk)
  if (!outside || cur === disk) return { kind: 'none' }
  const local = minimalChange(base, cur)
  if (!local) return { kind: 'apply', ...outside }
  const sameSpot = outside.from === local.from
  if (outside.to <= local.from && !sameSpot) return { kind: 'apply', ...outside }
  if (outside.from >= local.to && !sameSpot) {
    const shift = local.insert.length - (local.to - local.from)
    return { kind: 'apply', from: outside.from + shift, to: outside.to + shift, insert: outside.insert }
  }
  return { kind: 'conflict' }
}
