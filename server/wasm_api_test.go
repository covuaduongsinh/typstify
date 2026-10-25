package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWasmAssetsRequireLoginAndValidate(t *testing.T) {
	pkgRoot := t.TempDir()
	dir := filepath.Join(pkgRoot, "local", "demo", "0.1.0")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "typst.toml"), []byte("[package]"), 0o644); err != nil {
		t.Fatal(err)
	}
	fontDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(fontDir, "Demo.ttf"), []byte("font-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TYPSTIFY_WASM_FONT_DIRS", fontDir)
	t.Setenv("TYPSTIFY_WASM_SYSTEM_FONTS", "no-such-font-*")

	s := New(testService(), Options{StorageDir: tempStorage(t)})
	s.wasmPackages.SearchDirs = func() []string { return []string{pkgRoot} }
	s.wasmPackages.CacheDir = func() string { return "" }
	h := s.Handler()

	c := &apiClient{t: t, h: h}
	mustStatus(t, c.do("POST", "/api/auth/register", `{"username":"wasmuser","password":"pw1234"}`), 200, "register")

	// Once an account exists, every asset route needs a session.
	anon := &apiClient{t: t, h: h}
	mustStatus(t, anon.do("GET", "/api/wasm/fonts", ""), 401, "anonymous font list")
	mustStatus(t, anon.do("GET", "/api/wasm/packages/local/demo/0.1.0", ""), 401, "anonymous package")

	w := c.do("GET", "/api/wasm/fonts", "")
	mustStatus(t, w, 200, "font list")
	var fonts []struct {
		ID, Name, URL string
	}
	if err := json.Unmarshal(w.Body.Bytes(), &fonts); err != nil {
		t.Fatal(err)
	}
	var url string
	for _, f := range fonts {
		if f.Name == "Demo.ttf" {
			url = f.URL
		}
	}
	if url == "" {
		t.Fatalf("Demo.ttf not listed: %s", w.Body.String())
	}
	fw := c.do("GET", url, "")
	mustStatus(t, fw, 200, "font download")
	if fw.Body.String() != "font-bytes" {
		t.Fatalf("font body = %q", fw.Body.String())
	}
	mustStatus(t, c.do("GET", "/api/wasm/fonts/0011223344556677", ""), 404, "unknown font id")

	pw := c.do("GET", "/api/wasm/packages/local/demo/0.1.0", "")
	mustStatus(t, pw, 200, "local package")
	if pw.Header().Get("Content-Type") != "application/gzip" || pw.Header().Get("ETag") == "" {
		t.Fatalf("package headers: %v", pw.Header())
	}
	mustStatus(t, c.do("GET", "/api/wasm/packages/local/missing/0.1.0", ""), 404, "missing package")
	mustStatus(t, c.do("GET", "/api/wasm/packages/local/Bad_Name/0.1.0", ""), 400, "invalid package name")
	mustStatus(t, c.do("GET", "/api/wasm/packages/local/demo/latest", ""), 400, "invalid version")
}
