import { useState } from 'react'
import { useTranslations } from '../lib/i18n'
import { Icon } from './Icon'
import { Modal } from './Modal'

const FORMATS = ['pdf', 'png', 'svg', 'html'] as const

// Mirrors typst.PdfVersions (typst/pdf.go) -- the desktop export dialog's
// version radio buttons use these same strings verbatim.
const PDF_VERSIONS = ['PDF 1.4', 'PDF 1.5', 'PDF 1.6', 'PDF 1.7', 'PDF 2.0'] as const

// Mirrors typst.PdfStandard.Compatible (typst/pdf.go) -- which standards
// BuildParams will actually keep for a given version (an incompatible pair
// is silently dropped server-side, so we don't offer it in the first place).
const COMPATIBLE_STANDARDS: Record<string, string[]> = {
  'PDF 1.4': ['PDF/A-1b', 'PDF/A-1a'],
  'PDF 1.7': ['PDF/A-2b', 'PDF/A-2u', 'PDF/A-2a', 'PDF/A-3b', 'PDF/A-3u', 'PDF/A-3a', 'PDF/UA-1'],
  'PDF 2.0': ['PDF/A-4', 'PDF/A-4f', 'PDF/A-4e'],
}

const PAGES_HELP =
  'Chọn trang cần xuất. Có thể dùng số trang cách nhau bởi dấu phẩy và khoảng trang, ví dụ: 1,3,5-9. Để trống để xuất toàn bộ tài liệu.'

/** ExportButton downloads the currently open Typst file as PDF/PNG/SVG/HTML
 * via GET /api/export (server/export_api.go), which reuses the same
 * typst/export.CompileHelper the desktop app's export dialog uses. */
export function ExportButton({ path }: { path: string }) {
  const [format, setFormat] = useState<(typeof FORMATS)[number]>('pdf')
  const [pages, setPages] = useState('')
  const [showOptions, setShowOptions] = useState(false)

  const [ppi, setPpi] = useState(144)
  const [pdfVersion, setPdfVersion] = useState('')
  const [pdfStandard, setPdfStandard] = useState('')
  const [noPdfTags, setNoPdfTags] = useState(false)
  const [filename, setFilename] = useState('')

  const t = useTranslations(['Export'])

  const download = () => {
    const params = new URLSearchParams({ path, format })
    if (pages.trim()) params.set('pages', pages.trim())
    if (filename.trim()) params.set('filename', filename.trim())
    if (format === 'png' && ppi !== 144) params.set('ppi', String(ppi))
    if (format === 'pdf') {
      if (pdfVersion) params.set('pdfVersion', pdfVersion)
      if (pdfVersion && pdfStandard) params.set('pdfStandard', pdfStandard)
      if (noPdfTags) params.set('noPdfTags', '1')
    }
    window.open(`/api/export?${params.toString()}`, '_blank')
  }

  const compatibleStandards = pdfVersion ? (COMPATIBLE_STANDARDS[pdfVersion] ?? []) : []

  return (
    <div className="export-control">
      <select aria-label="Định dạng xuất" value={format} onChange={(e) => setFormat(e.target.value as (typeof FORMATS)[number])}>
        {FORMATS.map((f) => (
          <option key={f} value={f}>
            {f.toUpperCase()}
          </option>
        ))}
      </select>
      <input
        className="export-pages-input"
        aria-label="Trang cần xuất"
        placeholder="Tất cả trang"
        title={PAGES_HELP}
        value={pages}
        onChange={(e) => setPages(e.target.value)}
      />
      <button className="btn-ghost hdr-btn" onClick={() => setShowOptions(true)} title="Tuỳ chọn xuất nâng cao">
        <Icon name="sliders" size={14} />
      </button>
      <button className="btn-primary hdr-btn" onClick={download} title="Tải file đã biên dịch">
        <Icon name="download" />
        <span className="hdr-label">{t('Export')}</span>
      </button>

      {showOptions && (
        <Modal title={<><Icon name="sliders" /> Tuỳ chọn xuất nâng cao</>} onClose={() => setShowOptions(false)}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
            <div className="config-group">
              <label>Tên file xuất</label>
              <input
                className="chess-input"
                value={filename}
                onChange={(e) => setFilename(e.target.value)}
                placeholder="Mặc định: theo tên tệp nguồn"
              />
            </div>

            {format === 'png' && (
              <div className="config-group">
                <label>PPI (độ phân giải PNG): {ppi}</label>
                <input
                  type="range"
                  min={72}
                  max={300}
                  step={1}
                  value={ppi}
                  onChange={(e) => setPpi(Number(e.target.value))}
                />
              </div>
            )}

            {format === 'pdf' && (
              <>
                <div className="config-group">
                  <label>Phiên bản PDF</label>
                  <select
                    className="chess-input"
                    value={pdfVersion}
                    onChange={(e) => {
                      setPdfVersion(e.target.value)
                      setPdfStandard('')
                    }}
                  >
                    <option value="">Mặc định</option>
                    {PDF_VERSIONS.map((v) => (
                      <option key={v} value={v}>
                        {v}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="config-group">
                  <label>Chuẩn PDF/A (archival)</label>
                  <select
                    className="chess-input"
                    value={pdfStandard}
                    onChange={(e) => setPdfStandard(e.target.value)}
                    disabled={compatibleStandards.length === 0}
                  >
                    <option value="">Không</option>
                    {compatibleStandards.map((s) => (
                      <option key={s} value={s}>
                        {s}
                      </option>
                    ))}
                  </select>
                  {pdfVersion && compatibleStandards.length === 0 && (
                    <span className="export-option-hint">Phiên bản này không có chuẩn PDF/A tương ứng.</span>
                  )}
                </div>
                <label className="export-checkbox-row">
                  <input type="checkbox" checked={noPdfTags} onChange={(e) => setNoPdfTags(e.target.checked)} />
                  Tắt PDF tags (giảm dung lượng, mất khả năng đọc trợ năng)
                </label>
              </>
            )}
          </div>
        </Modal>
      )}
    </div>
  )
}
