import { useState } from 'react'
import { useTranslations } from '../lib/i18n'
import { Icon } from './Icon'

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
      <select aria-label="Định dạng xuất" value={format} onChange={(e) => setFormat(e.target.value as (typeof FORMATS)[number])}>
        {FORMATS.map((f) => (
          <option key={f} value={f}>
            {f.toUpperCase()}
          </option>
        ))}
      </select>
      <button className="btn-primary hdr-btn" onClick={download} title="Tải file đã biên dịch">
        <Icon name="download" />
        <span className="hdr-label">{t('Export')}</span>
      </button>
    </div>
  )
}
