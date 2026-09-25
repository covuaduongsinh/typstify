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

/** The import line that brings in the chessbook library (installed as a
 * local Typst package on the server and, via scripts/, on desktops). */
export const CHESSBOOK_IMPORT = '#import "@local/chessbook:0.1.0": *'

/** hasChessbookImport reports whether a document already imports the
 * library, either as the package or via the in-repo relative path. */
export function hasChessbookImport(doc: string): boolean {
  return /@local\/chessbook:|lib\/lib\.typ/.test(doc)
}
