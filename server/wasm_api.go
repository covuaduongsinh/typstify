package server

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"looz.ws/typstify/service/fonts"
	"looz.ws/typstify/service/wasmassets"
)

// Transport for the in-browser compiler (web/src/lib/wasm): font files and
// package archives. The lookup logic lives in service/wasmassets.

// newWasmPackages searches the same package roots as the native compile:
// the configured/env package dir (the Docker image bundles @local/chessbook
// there), typst's own data dir, then the package cache.
func (s *Server) newWasmPackages() *wasmassets.Packages {
	return &wasmassets.Packages{
		SearchDirs: func() []string {
			var dirs []string
			if s.appSrv != nil && s.appSrv.Settings() != nil {
				dirs = append(dirs, s.appSrv.Settings().Typst().EffectivePackageDir())
			}
			return append(dirs, wasmassets.TypstDataPackageDir())
		},
		CacheDir: func() string {
			if s.appSrv == nil || s.appSrv.PkgService() == nil {
				return ""
			}
			return s.appSrv.PkgService().CacheDir()
		},
	}
}

// wasmFontSources mirrors where the native typst finds fonts: the extra
// font dir (uploads), TYPSTIFY_WASM_FONT_DIRS, and -- unless system fonts
// are disabled in Settings -- the system fonts matching
// TYPSTIFY_WASM_SYSTEM_FONTS (default: the fonts chessbook uses).
func (s *Server) wasmFontSources() wasmassets.FontSources {
	var src wasmassets.FontSources
	ignoreSystem := false
	if s.appSrv != nil && s.appSrv.Settings() != nil {
		if dir, err := fonts.Dir(s.appSrv.Settings(), false); err == nil && dir != "" {
			src.Dirs = append(src.Dirs, dir)
		}
		ignoreSystem = s.appSrv.Settings().Typst().IgnoreSystemFonts == 1
	}
	src.Dirs = append(src.Dirs, wasmassets.SplitList(os.Getenv("TYPSTIFY_WASM_FONT_DIRS"), filepath.ListSeparator)...)
	if !ignoreSystem {
		src.SystemDirs = wasmassets.SystemFontDirs()
		src.Patterns = wasmassets.DefaultSystemFontPatterns
		if v := os.Getenv("TYPSTIFY_WASM_SYSTEM_FONTS"); v != "" {
			src.Patterns = wasmassets.SplitList(v, ',')
		}
	}
	return src
}

const wasmFontRescanInterval = 30 * time.Second

type wasmFontView struct {
	wasmassets.Font
	URL string `json:"url"`
}

// handleWasmFonts lists the fonts the browser compiler should load.
func (s *Server) handleWasmFonts(w http.ResponseWriter, r *http.Request) {
	list := s.wasmFonts.Scan(s.wasmFontSources())
	out := make([]wasmFontView, 0, len(list))
	for _, f := range list {
		out = append(out, wasmFontView{Font: f, URL: "/api/wasm/fonts/" + f.ID + "?v=" + f.Version})
	}
	w.Header().Set("Cache-Control", "no-cache")
	writeJSON(w, http.StatusOK, out)
}

// handleWasmFont serves one font listed by handleWasmFonts. Ids are opaque
// hashes resolved against the last scan, so no path ever comes from the
// client.
func (s *Server) handleWasmFont(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, ok := s.wasmFonts.Lookup(id)
	// An id from a list fetched before a restart or a font upload needs a
	// fresh scan; RescanAllowed limits how often unknown ids can force one.
	if !ok && s.wasmFonts.RescanAllowed(wasmFontRescanInterval) {
		s.wasmFonts.Scan(s.wasmFontSources())
		p, ok = s.wasmFonts.Lookup(id)
	}
	if !ok {
		writeError(w, http.StatusNotFound, "không tìm thấy font")
		return
	}
	f, err := os.Open(p)
	if err != nil {
		writeError(w, http.StatusNotFound, "không tìm thấy font")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	http.ServeContent(w, r, "", info.ModTime(), f)
}

// handleWasmPackage serves a package as .tar.gz, downloading @preview
// packages into the server's cache on first use.
func (s *Server) handleWasmPackage(w http.ResponseWriter, r *http.Request) {
	spec, err := wasmassets.ParseSpec(r.PathValue("namespace"), r.PathValue("name"), r.PathValue("version"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	a, err := s.wasmPackages.Archive(r.Context(), spec)
	if errors.Is(err, wasmassets.ErrNotFound) {
		writeError(w, http.StatusNotFound, "không tìm thấy package "+spec.String())
		return
	}
	if err != nil {
		// The error may carry server paths; log it, answer generically.
		log.Printf("wasm package %s: %v", spec, err)
		writeError(w, http.StatusBadGateway, "không lấy được package "+spec.String())
		return
	}
	if a.Immutable {
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "private, no-cache")
	}
	w.Header().Set("ETag", a.ETag)
	if match := r.Header.Get("If-None-Match"); match != "" && strings.Contains(match, a.ETag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/gzip")
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(a.Data))
}
