package server

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"looz.ws/typstify/service/universe"
	"looz.ws/typstify/service/wasmassets"
)

// universeCacheDir keeps the Universe index and thumbnails across restarts.
func universeCacheDir(storageDir string) string {
	if storageDir != "" {
		return filepath.Join(storageDir, "universe")
	}
	if dir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(dir, "typstify", "universe")
	}
	return ""
}

func newUniverse(storageDir string) (*universe.Index, *universe.Thumbnails) {
	dir := universeCacheDir(storageDir)
	ix := &universe.Index{}
	th := &universe.Thumbnails{}
	if dir != "" {
		ix.CacheFile = filepath.Join(dir, "index.json")
		th.CacheDir = filepath.Join(dir, "thumbnails")
	}
	return ix, th
}

// handleUniverseSearch lists Typst Universe packages:
// ?kind=template|package&q=&category=&offset=&limit=.
func (s *Server) handleUniverseSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	offset, _ := strconv.Atoi(q.Get("offset"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	page, err := s.universe.Search(r.Context(), universe.Query{
		Kind:     q.Get("kind"),
		Text:     q.Get("q"),
		Category: q.Get("category"),
		Offset:   offset,
		Limit:    limit,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// universeSpec validates {name}/{version} of a @preview package.
func universeSpec(w http.ResponseWriter, r *http.Request) (wasmassets.Spec, bool) {
	spec, err := wasmassets.ParseSpec("preview", r.PathValue("name"), r.PathValue("version"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return spec, false
	}
	return spec, true
}

// handleUniverseThumbnail serves a template thumbnail (WebP), ?size=small
// for the gallery size.
func (s *Server) handleUniverseThumbnail(w http.ResponseWriter, r *http.Request) {
	spec, ok := universeSpec(w, r)
	if !ok {
		return
	}
	data, err := s.universeThumbs.Get(r.Context(), spec.Name, spec.Version, r.URL.Query().Get("size") == "small")
	if errors.Is(err, universe.ErrNoThumbnail) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	// A published version never changes.
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.Header().Set("Content-Type", "image/webp")
	_, _ = w.Write(data)
}

// handleUniverseReadme returns {readme} of a @preview package, downloading
// it into the package cache (as the in-browser compiler would).
func (s *Server) handleUniverseReadme(w http.ResponseWriter, r *http.Request) {
	spec, ok := universeSpec(w, r)
	if !ok {
		return
	}
	dir, err := s.wasmPackages.Dir(r.Context(), spec)
	if errors.Is(err, wasmassets.ErrNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	readme, err := universe.ReadReadme(dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=86400")
	writeJSON(w, http.StatusOK, map[string]string{"readme": readme})
}
