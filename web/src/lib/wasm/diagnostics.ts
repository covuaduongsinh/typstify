import type { WasmDiagnostic } from './protocol'

export interface DiagnosticPosition {
  line: number
  col: number
}

/** Parses typst.ts' "startLine:startCol-endLine:endCol" range (0-based)
 * into a 1-based start position. */
export function diagnosticStart(range: string): DiagnosticPosition | null {
  const m = /^(\d+):(\d+)/.exec(range.trim())
  if (!m) return null
  return { line: Number(m[1]) + 1, col: Number(m[2]) + 1 }
}

export const isError = (d: WasmDiagnostic) => d.severity.toLowerCase() === 'error'

/** Formats diagnostics like the CLI does: `file:line:col: error: message`. */
export function formatDiagnostics(diags: WasmDiagnostic[], max = 8): string {
  const lines = diags.slice(0, max).map((d) => {
    const pos = diagnosticStart(d.range)
    const file = (d.package ? `${d.package}/` : '') + d.path.replace(/^\//, '')
    const where = pos ? `${file}:${pos.line}:${pos.col}` : file
    return `${where}: ${d.severity.toLowerCase()}: ${d.message}`
  })
  if (diags.length > max) lines.push(`… và ${diags.length - max} thông báo khác`)
  return lines.join('\n')
}
