// Renders the current .typ file as a plain PDF fetched over a normal GET
// request (server/export_api.go's handlePreviewPdf), instead of tinymist's
// live WebSocket preview (/preview/). The WS approach doesn't work through
// this deployment's reverse proxy chain (a browser WebSocket-over-HTTP/2
// upgrade failing against the shared Traefik instance in front of it -- see
// handlePreviewPdf's doc comment), and a plain PDF fetch needs no WebSocket
// at all. Workspace.tsx remounts this component (via its `key`) whenever
// the active file changes or gets saved, which is what makes the <embed>
// reload -- same "updates on save, not on keystroke" behavior the WS
// preview had anyway.
export function PreviewPane({ path }: { path: string | null }) {
  if (!path || !path.endsWith('.typ')) {
    return (
      <div className="preview-pane">
        <div className="preview-loading">Select a Typst file to preview</div>
      </div>
    )
  }

  return (
    <div className="preview-pane">
      <embed className="preview-pdf" src={`/api/preview/pdf?path=${encodeURIComponent(path)}`} type="application/pdf" />
    </div>
  )
}
