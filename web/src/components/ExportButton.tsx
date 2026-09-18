import { useState } from 'react'
import { useTranslations } from '../lib/i18n'

const FORMATS = ['pdf', 'png', 'svg'] as const

/** ExportButton downloads the currently open Typst file as PDF/PNG/SVG via
 * GET /api/export (server/export_api.go), which reuses the same
 * typst/export.CompileHelper the desktop app's export dialog uses. */
export function ExportButton({ path }: { path: string }) {
  const [format, setFormat] = useState<(typeof FORMATS)[number]>('pdf')
  const t = useTranslations(['Export'])

  const download = () => {
    const url = `/api/export?path=${encodeURIComponent(path)}&format=${format}`
    window.open(url, '_blank')
  }

  return (
    <div className="export-control">
      <select value={format} onChange={(e) => setFormat(e.target.value as (typeof FORMATS)[number])}>
        {FORMATS.map((f) => (
          <option key={f} value={f}>
            {f.toUpperCase()}
          </option>
        ))}
      </select>
      <button onClick={download}>{t('Export')}</button>
    </div>
  )
}
