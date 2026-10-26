package server

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticSPAFallbackOnlyForRoutes(t *testing.T) {
	dir := tempStorage(t)
	for name, content := range map[string]string{
		"index.html":       "<!doctype html><title>app</title>",
		"assets/main-1.js": "console.log(1)",
		"sw.js":            "self",
	} {
		p := filepath.Join(dir, filepath.FromSlash(name))
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := &Server{opts: Options{StaticDir: dir}}
	h := s.staticHandler()
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}

	if w := get("/assets/main-1.js"); w.Code != 200 || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("asset: %d %q", w.Code, w.Header().Get("Cache-Control"))
	}
	if w := get("/sw.js"); w.Code != 200 || w.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("sw.js: %d %q", w.Code, w.Header().Get("Cache-Control"))
	}
	// Client routes get the app.
	for _, p := range []string{"/", "/projects/abc", "/?join=x"} {
		if w := get(p); w.Code != 200 || !strings.Contains(w.Body.String(), "<title>app") {
			t.Errorf("%s: %d %q", p, w.Code, w.Body.String())
		}
	}
	// Missing files are 404, never index.html under a script's URL.
	for _, p := range []string{"/assets/main-old.js", "/assets/x", "/missing.js", "/icons/none.png"} {
		if w := get(p); w.Code != 404 {
			t.Errorf("%s: %d, want 404", p, w.Code)
		}
	}
}
