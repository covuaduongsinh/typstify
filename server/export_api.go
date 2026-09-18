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
