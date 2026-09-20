package server

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"looz.ws/typstify/typst"
	"looz.ws/typstify/typst/export"
)

// handleExport compiles a Typst document to PDF/PNG/SVG (reusing
// typst/export.CompileHelper, the same helper the desktop app's export
// dialog uses -- see ui/dialog/export.go) and streams the result back as a
// file download. PNG/SVG export can produce one file per page; when it
// does, the files are zipped together.
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	targetFile, err := resolveInRoot(root, r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := os.Stat(targetFile); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	format := typst.OutFormat(r.URL.Query().Get("format"))
	if format == "" {
		format = typst.PDF
	}
	switch format {
	case typst.PDF, typst.PNG, typst.SVG:
	default:
		writeError(w, http.StatusBadRequest, "unsupported format: "+string(format))
		return
	}

	outDir, err := os.MkdirTemp("", "typstify-export-*")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer os.RemoveAll(outDir)

	outName := strings.TrimSuffix(filepath.Base(targetFile), filepath.Ext(targetFile))

	helper := export.NewCompileHelper(root, s.appSrv.Settings().Typst())
	helper.Format = format
	helper.Pages = r.URL.Query().Get("pages")
	helper.PPI = 144

	params, err := helper.BuildParams(targetFile, outName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	params.OutDir = outDir // override CompileHelper's project-relative default

	if err := helper.Compile(params); err != nil {
		writeError(w, http.StatusInternalServerError, "export failed: "+err.Error())
		return
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, filepath.Join(outDir, e.Name()))
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		writeError(w, http.StatusInternalServerError, "export produced no output")
		return
	}

	if len(files) == 1 {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(files[0])))
		http.ServeFile(w, r, files[0])
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", outName+".zip"))
	zw := zip.NewWriter(w)
	defer zw.Close()
	for _, f := range files {
		if err := addFileToZip(zw, f); err != nil {
			return // headers/partial body already sent; best-effort only
		}
	}
}

// handlePreviewPdf compiles a Typst document to PDF and serves it inline
// (no Content-Disposition) so a browser <iframe>/<embed> renders it directly
// with the browser's native PDF viewer, instead of the "download" behavior
// handleExport wants for its explicit Export button.
//
// This exists because the tinymist live-preview WebSocket (server/
// preview_proxy.go, /preview/) does not work through this deployment's
// reverse proxy chain (confirmed: the browser's WebSocket-over-HTTP/2
// upgrade to it fails with code 1006 on every attempt, while a plain
// HTTP/1.1 WS handshake to the same backend succeeds -- an HTTP/2-vs-
// WebSocket interaction in the shared Traefik instance in front of this
// server, not something fixable from here without touching config that
// also serves other unrelated apps). A plain PDF fetched over a normal GET
// request needs no WebSocket at all, so it sidesteps the problem entirely.
// Same tradeoff as before: refreshes on save (via Workspace.tsx bumping
// previewVersion), not on every keystroke -- matching what the WS preview
// only ever did too (docs/web-server.md's known v1 limitation).
func (s *Server) handlePreviewPdf(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	targetFile, err := resolveInRoot(root, r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := os.Stat(targetFile); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	outDir, err := os.MkdirTemp("", "typstify-preview-*")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer os.RemoveAll(outDir)

	outName := strings.TrimSuffix(filepath.Base(targetFile), filepath.Ext(targetFile))

	helper := export.NewCompileHelper(root, s.appSrv.Settings().Typst())
	helper.Format = typst.PDF
	helper.PPI = 144

	params, err := helper.BuildParams(targetFile, outName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	params.OutDir = outDir

	if err := helper.Compile(params); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "compile failed: "+err.Error())
		return
	}

	pdfPath := filepath.Join(outDir, outName+".pdf")
	if _, err := os.Stat(pdfPath); err != nil {
		writeError(w, http.StatusInternalServerError, "compile produced no PDF")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	// Best-effort: lets a plain reload notice the file changed instead of
	// showing a cached copy of the previous compile.
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, pdfPath)
}

func addFileToZip(zw *zip.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	entry, err := zw.Create(filepath.Base(path))
	if err != nil {
		return err
	}
	_, err = io.Copy(entry, f)
	return err
}
