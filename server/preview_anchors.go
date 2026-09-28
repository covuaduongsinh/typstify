package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"looz.ws/typstify/typst"
	"looz.ws/typstify/typst/export"
)

type previewAnchorsRequest struct {
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
}

type previewAnchorsResponse struct {
	OK    bool   `json:"ok"`
	Pages []int  `json:"pages,omitempty"`
	Error string `json:"error,omitempty"`
}

// handlePreviewAnchors returns the real physical page number of every
// heading in the document, in document order, via Typst's own query()
// introspection (see typst.QueryHeadingPages). This feeds real anchors into
// the frontend's line->page interpolation instead of relying purely on
// #pagebreak() + a linear-lines-per-page guess.
//
// Deliberately a separate endpoint from /api/preview/render (not fused into
// the same request): an eval+query pass costs about as much as a full
// compile, and the frontend only needs fresh anchors when the document's
// heading structure actually changes -- far less often than every keystroke
// that triggers a render.
func (s *Server) handlePreviewAnchors(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeJSON(w, http.StatusOK, previewAnchorsResponse{OK: false, Error: err.Error()})
		return
	}

	var req previewAnchorsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusOK, previewAnchorsResponse{OK: false, Error: "invalid request body: " + err.Error()})
		return
	}

	if req.Path == "" {
		writeJSON(w, http.StatusOK, previewAnchorsResponse{OK: false, Error: "path parameter is required"})
		return
	}

	targetFile, err := resolveInRoot(root, req.Path)
	if err != nil {
		writeJSON(w, http.StatusOK, previewAnchorsResponse{OK: false, Error: err.Error()})
		return
	}

	// Same shadow-file trick as handlePreviewRender: write unsaved live
	// content next to the real file so relative imports still resolve.
	var compileInputFile string
	var cleanupInput func()
	if req.Content != "" {
		targetDir := filepath.Dir(targetFile)
		tempFile, err := os.CreateTemp(targetDir, ".live_preview_*.typ")
		if err != nil {
			writeJSON(w, http.StatusOK, previewAnchorsResponse{OK: false, Error: "failed to create live preview temp file: " + err.Error()})
			return
		}
		if _, err := tempFile.WriteString(req.Content); err != nil {
			_ = tempFile.Close()
			_ = os.Remove(tempFile.Name())
			writeJSON(w, http.StatusOK, previewAnchorsResponse{OK: false, Error: "failed to write live preview content: " + err.Error()})
			return
		}
		_ = tempFile.Close()
		compileInputFile = tempFile.Name()
		cleanupInput = func() { _ = os.Remove(compileInputFile) }
	} else {
		if _, err := os.Stat(targetFile); err != nil {
			writeJSON(w, http.StatusOK, previewAnchorsResponse{OK: false, Error: "file not found: " + req.Path})
			return
		}
		compileInputFile = targetFile
		cleanupInput = func() {}
	}
	defer cleanupInput()

	ctx, release, ok := s.acquireCompile(w, r)
	if !ok {
		return
	}
	defer release()

	// Reuse CompileHelper.BuildParams purely to get the same root/font-path/
	// package-path/features/input a real compile of this file would use --
	// Format/PPI/OutDir are irrelevant here and left at zero values.
	helper := export.NewCompileHelper(root, s.appSrv.Settings().Typst())
	params, err := helper.BuildParams(compileInputFile, "")
	if err != nil {
		writeJSON(w, http.StatusOK, previewAnchorsResponse{OK: false, Error: "build query params error: " + err.Error()})
		return
	}
	params.Options.RootDir = root

	pages, err := typst.QueryHeadingPages(ctx, &params.Options, compileInputFile)
	if err != nil {
		writeJSON(w, http.StatusOK, previewAnchorsResponse{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, previewAnchorsResponse{OK: true, Pages: pages})
}
