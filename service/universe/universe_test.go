package universe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// tempDir is removed best-effort (Windows can keep files busy briefly).
func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "universe-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

const sampleIndex = `[
 {"name":"chess-book","version":"0.1.0","description":"old","categories":["book"],"template":{"path":"t","entrypoint":"main.typ"},"updatedAt":10},
 {"name":"chess-book","version":"0.10.0","description":"Chess book template","keywords":["chess"],"categories":["book"],"template":{"path":"t","entrypoint":"main.typ","thumbnail":"thumb.png"},"updatedAt":30},
 {"name":"chess-book","version":"0.9.0","description":"older","categories":["book"],"updatedAt":20},
 {"name":"cetz","version":"0.4.2","description":"Drawing with Typst","categories":["visualization"],"updatedAt":50},
 {"name":"letter","version":"1.0.0","description":"A letter","categories":["office"],"template":{"path":"template","entrypoint":"letter.typ"},"updatedAt":40}
]`

func indexServer(t *testing.T, body string, status *atomic.Int32, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if code := status.Load(); code != 0 && code != 200 {
			w.WriteHeader(int(code))
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSearchKeepsLatestVersionAndFilters(t *testing.T) {
	var status, hits atomic.Int32
	ix := &Index{URL: indexServer(t, sampleIndex, &status, &hits).URL}
	ctx := context.Background()

	page, err := ix.Search(ctx, Query{Kind: "template"})
	if err != nil {
		t.Fatal(err)
	}
	// Latest version by number (0.10.0 > 0.9.0), templates only, newest first.
	if page.Total != 2 || page.Packages[0].Name != "letter" || page.Packages[1].Version != "0.10.0" {
		t.Fatalf("templates = %+v", page)
	}
	if page, _ = ix.Search(ctx, Query{Kind: "package"}); page.Total != 1 || page.Packages[0].Name != "cetz" {
		t.Fatalf("packages = %+v", page)
	}
	if page, _ = ix.Search(ctx, Query{Text: "CHESS"}); page.Total != 1 || page.Packages[0].Name != "chess-book" {
		t.Fatalf("text = %+v", page)
	}
	if page, _ = ix.Search(ctx, Query{Category: "office"}); page.Total != 1 {
		t.Fatalf("category = %+v", page)
	}
	if page, _ = ix.Search(ctx, Query{Offset: 1, Limit: 1}); page.Total != 3 || len(page.Packages) != 1 {
		t.Fatalf("paging = %+v", page)
	}
	if p, ok, _ := ix.Lookup(ctx, "letter"); !ok || p.Template == nil || p.Template.Entrypoint != "letter.typ" {
		t.Fatalf("lookup = %+v %v", p, ok)
	}
	if hits.Load() != 1 {
		t.Fatalf("index fetched %d times", hits.Load())
	}
}

func TestIndexFallsBackToDiskCopy(t *testing.T) {
	var status, hits atomic.Int32
	status.Store(500)
	cache := filepath.Join(tempDir(t), "index.json")
	if err := os.WriteFile(cache, []byte(sampleIndex), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	_ = os.Chtimes(cache, old, old)

	ix := &Index{URL: indexServer(t, sampleIndex, &status, &hits).URL, CacheFile: cache}
	page, err := ix.Search(context.Background(), Query{})
	if err != nil || page.Total != 3 {
		t.Fatalf("stale disk copy: %+v %v", page, err)
	}
	// The stale copy triggers one background refresh, which fails here.
	deadline := time.Now().Add(5 * time.Second)
	for hits.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if hits.Load() != 1 {
		t.Fatalf("refresh attempts = %d", hits.Load())
	}

	// No copy at all and no network: an error, and the next calls within
	// retryAfter get it again without another request.
	var emptyHits atomic.Int32
	empty := &Index{URL: indexServer(t, sampleIndex, &status, &emptyHits).URL}
	for range 3 {
		if _, err := empty.Search(context.Background(), Query{}); err == nil {
			t.Fatal("expected an error without index")
		}
	}
	if emptyHits.Load() != 1 {
		t.Fatalf("first-load attempts = %d, want 1", emptyHits.Load())
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCopyTemplate(t *testing.T) {
	pkg := tempDir(t)
	write(t, filepath.Join(pkg, "typst.toml"), "[package]\nname = \"demo\"\nversion = \"0.1.0\"\nentrypoint = \"lib.typ\"\n\n[template]\npath = \"template\"\nentrypoint = \"main.typ\"\nthumbnail = \"thumb.png\"\n")
	write(t, filepath.Join(pkg, "lib.typ"), "#let x = 1")
	write(t, filepath.Join(pkg, "template", "main.typ"), `#import "@preview/demo:0.1.0": *`)
	write(t, filepath.Join(pkg, "template", "chapters", "one.typ"), "= One")
	write(t, filepath.Join(pkg, "README.md"), "# Demo")

	dst := tempDir(t)
	write(t, filepath.Join(dst, "keep.txt"), "mine")
	main, err := CopyTemplate(pkg, dst)
	if err != nil || main != "main.typ" {
		t.Fatalf("CopyTemplate = %q, %v", main, err)
	}
	for _, f := range []string{"main.typ", "chapters/one.typ", "keep.txt"} {
		if _, err := os.Stat(filepath.Join(dst, f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
	if _, err := os.Stat(filepath.Join(dst, "lib.typ")); err == nil {
		t.Error("package files outside the template were copied")
	}
	if readme, _ := ReadReadme(pkg); readme != "# Demo" {
		t.Errorf("readme = %q", readme)
	}
}

func TestCopyTemplateRejectsEscapes(t *testing.T) {
	for name, toml := range map[string]string{
		"no template":   "[package]\nname = \"x\"\n",
		"path escapes":  "[template]\npath = \"../outside\"\nentrypoint = \"main.typ\"\n",
		"entry escapes": "[template]\npath = \"t\"\nentrypoint = \"../../x.typ\"\n",
		"absolute path": "[template]\npath = \"/etc\"\nentrypoint = \"main.typ\"\n",
	} {
		pkg := tempDir(t)
		write(t, filepath.Join(pkg, "typst.toml"), toml)
		write(t, filepath.Join(pkg, "t", "main.typ"), "x")
		if _, err := CopyTemplate(pkg, tempDir(t)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestThumbnailsAreCachedOnDisk(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path != "/demo-0.1.0-small.webp" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("WEBP"))
	}))
	defer srv.Close()
	th := &Thumbnails{BaseURL: srv.URL, CacheDir: tempDir(t)}
	for range 2 {
		if data, err := th.Get(context.Background(), "demo", "0.1.0", true); err != nil || string(data) != "WEBP" {
			t.Fatalf("Get = %q, %v", data, err)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("fetched %d times", hits.Load())
	}
	for range 2 {
		if _, err := th.Get(context.Background(), "demo", "0.1.0", false); err != ErrNoThumbnail {
			t.Fatalf("missing thumbnail err = %v", err)
		}
	}
	if hits.Load() != 2 {
		t.Fatalf("a missing thumbnail was asked for again: %d requests", hits.Load())
	}
}

func TestCompareVersions(t *testing.T) {
	if compareVersions("0.10.0", "0.9.9") <= 0 || compareVersions("1.0.0", "1.0.0") != 0 || compareVersions("0.1.0", "0.1.1") >= 0 {
		t.Fatal("compareVersions")
	}
}
