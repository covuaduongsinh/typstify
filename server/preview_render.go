package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"looz.ws/typstify/typst"
	"looz.ws/typstify/typst/export"
)

type previewRenderRequest struct {
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
	Format  string `json:"format,omitempty"`
}

type previewRenderResponse struct {
	OK        bool     `json:"ok"`
	Pages     []string `json:"pages,omitempty"`
	PageCount int      `json:"pageCount"`
	Error     string   `json:"error,omitempty"`
}

var pageNumRegex = regexp.MustCompile(`(\d+)\.svg$`)

func extractPageNum(filename string) int {
	matches := pageNumRegex.FindStringSubmatch(filename)
	if len(matches) > 1 {
		if n, err := strconv.Atoi(matches[1]); err == nil {
			return n
		}
	}
	return 0
}

// handlePreviewRender compiles the Typst document (either from disk or from
// live unsaved editor content) and returns SVG pages for instant live preview.
func (s *Server) handlePreviewRender(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeJSON(w, http.StatusOK, previewRenderResponse{
			OK:    false,
			Error: err.Error(),
		})
		return
	}

	var req previewRenderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusOK, previewRenderResponse{
			OK:    false,
			Error: "invalid request body: " + err.Error(),
		})
		return
	}

	if req.Path == "" {
		writeJSON(w, http.StatusOK, previewRenderResponse{
			OK:    false,
			Error: "path parameter is required",
		})
		return
	}

	targetFile, err := resolveInRoot(root, req.Path)
	if err != nil {
		writeJSON(w, http.StatusOK, previewRenderResponse{
			OK:    false,
			Error: err.Error(),
		})
		return
	}

	// Prepare compile input file: if Content is provided, create a shadow file
	// in target's directory to ensure relative imports inside the project resolve.
	var compileInputFile string
	var cleanupInput func()
	if req.Content != "" {
		targetDir := filepath.Dir(targetFile)
		tempFile, err := os.CreateTemp(targetDir, ".live_preview_*.typ")
		if err != nil {
			writeJSON(w, http.StatusOK, previewRenderResponse{
				OK:    false,
				Error: "failed to create live preview temp file: " + err.Error(),
			})
			return
		}
		if _, err := tempFile.WriteString(req.Content); err != nil {
			_ = tempFile.Close()
			_ = os.Remove(tempFile.Name())
			writeJSON(w, http.StatusOK, previewRenderResponse{
				OK:    false,
				Error: "failed to write live preview content: " + err.Error(),
			})
			return
		}
		_ = tempFile.Close()
		compileInputFile = tempFile.Name()
		cleanupInput = func() {
			_ = os.Remove(compileInputFile)
		}
	} else {
		if _, err := os.Stat(targetFile); err != nil {
			writeJSON(w, http.StatusOK, previewRenderResponse{
				OK:    false,
				Error: "file not found: " + req.Path,
			})
			return
		}
		compileInputFile = targetFile
		cleanupInput = func() {}
	}
	defer cleanupInput()

	outDir, err := os.MkdirTemp("", "typstify-preview-live-*")
	if err != nil {
		writeJSON(w, http.StatusOK, previewRenderResponse{
			OK:    false,
			Error: "failed to create output temp dir: " + err.Error(),
		})
		return
	}
	defer os.RemoveAll(outDir)

	ctx, release, ok := s.acquireCompile(w, r)
	if !ok {
		return
	}
	defer release()

	format := typst.SVG
	if req.Format == "pdf" {
		format = typst.PDF
	}

	outName := "preview_out"

	helper := export.NewCompileHelper(root, s.appSrv.Settings().Typst())
	helper.Ctx = ctx
	helper.Format = format
	helper.PPI = 144

	params, err := helper.BuildParams(compileInputFile, outName)
	if err != nil {
		writeJSON(w, http.StatusOK, previewRenderResponse{
			OK:    false,
			Error: "build compile params error: " + err.Error(),
		})
		return
	}
	params.OutDir = outDir

	if err := helper.Compile(params); err != nil {
		writeJSON(w, http.StatusOK, previewRenderResponse{
			OK:    false,
			Error: err.Error(),
		})
		return
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		writeJSON(w, http.StatusOK, previewRenderResponse{
			OK:    false,
			Error: "read output dir error: " + err.Error(),
		})
		return
	}

	var svgFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".svg") {
			svgFiles = append(svgFiles, filepath.Join(outDir, e.Name()))
		}
	}

	// Sort numerically so page 10 comes after page 9 instead of page 1.
	sort.Slice(svgFiles, func(i, j int) bool {
		numI := extractPageNum(svgFiles[i])
		numJ := extractPageNum(svgFiles[j])
		if numI != numJ {
			return numI < numJ
		}
		return svgFiles[i] < svgFiles[j]
	})

	var pages []string
	for _, f := range svgFiles {
		contentBytes, err := os.ReadFile(f)
		if err == nil {
			pages = append(pages, string(contentBytes))
		}
	}

	if len(pages) == 0 {
		writeJSON(w, http.StatusOK, previewRenderResponse{
			OK:    false,
			Error: "Không tạo được trang xem trước nào từ tài liệu.",
		})
		return
	}

	writeJSON(w, http.StatusOK, previewRenderResponse{
		OK:        true,
		Pages:     pages,
		PageCount: len(pages),
	})
}
