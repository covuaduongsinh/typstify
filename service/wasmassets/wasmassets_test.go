package wasmassets

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFontScanFiltersAndDedupes(t *testing.T) {
	up := t.TempDir()
	sys := t.TempDir()
	writeFile(t, filepath.Join(up, "Roboto-Regular.ttf"), "uploaded")
	writeFile(t, filepath.Join(up, "nested", "Custom.otf"), "c")
	writeFile(t, filepath.Join(up, "notes.txt"), "x")
	writeFile(t, filepath.Join(sys, "roboto", "Roboto-Regular.ttf"), "system copy")
	writeFile(t, filepath.Join(sys, "roboto", "Roboto-Bold.TTF"), "b")
	writeFile(t, filepath.Join(sys, "Arial.ttf"), "not selected")

	x := NewFontIndex()
	fonts := x.Scan(FontSources{Dirs: []string{up}, SystemDirs: []string{sys}, Patterns: DefaultSystemFontPatterns})
	var names []string
	for _, f := range fonts {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	want := []string{"Custom.otf", "Roboto-Bold.TTF", "Roboto-Regular.ttf"}
	if len(names) != len(want) {
		t.Fatalf("fonts = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("fonts = %v, want %v", names, want)
		}
	}

	// The uploaded copy wins over the system one.
	for _, f := range fonts {
		if f.Name != "Roboto-Regular.ttf" {
			continue
		}
		p, ok := x.Lookup(f.ID)
		if !ok {
			t.Fatal("lookup failed")
		}
		data, _ := os.ReadFile(p)
		if string(data) != "uploaded" {
			t.Fatalf("got %q, want the uploaded font", data)
		}
	}
	if _, ok := x.Lookup("deadbeef"); ok {
		t.Fatal("unknown id must not resolve")
	}
}

func TestParseSpec(t *testing.T) {
	good := [][3]string{{"preview", "board-n-pieces", "0.9.0"}, {"local", "chessbook", "0.1.0"}}
	for _, g := range good {
		if _, err := ParseSpec(g[0], g[1], g[2]); err != nil {
			t.Errorf("%v: %v", g, err)
		}
	}
	bad := [][3]string{
		{"..", "x", "0.1.0"}, {"preview", "../etc", "0.1.0"}, {"preview", "x", "0.1"},
		{"preview", "X", "0.1.0"}, {"preview", "x", "0.1.0/.."}, {"", "x", "0.1.0"},
	}
	for _, b := range bad {
		if _, err := ParseSpec(b[0], b[1], b[2]); !errors.Is(err, ErrBadSpec) {
			t.Errorf("%v: want ErrBadSpec, got %v", b, err)
		}
	}
}

func untar(t *testing.T, data []byte) map[string]string {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	out := map[string]string{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(tr)
		out[h.Name] = string(b)
	}
}

func TestArchiveLocalPackage(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "local", "chessbook", "0.1.0")
	writeFile(t, filepath.Join(dir, "typst.toml"), "[package]\nname = \"chessbook\"\n")
	writeFile(t, filepath.Join(dir, "lib.typ"), "#let x = 1")
	writeFile(t, filepath.Join(dir, "sub", "a.typ"), "a")

	p := &Packages{SearchDirs: func() []string { return []string{root} }}
	spec, _ := ParseSpec("local", "chessbook", "0.1.0")
	a, err := p.Archive(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if a.Immutable {
		t.Fatal("local packages must not be immutable")
	}
	files := untar(t, a.Data)
	if files["lib.typ"] != "#let x = 1" || files["sub/a.typ"] != "a" || files["typst.toml"] == "" {
		t.Fatalf("unexpected archive contents: %v", files)
	}

	again, _ := p.Archive(context.Background(), spec)
	if again.ETag != a.ETag {
		t.Fatal("unchanged package should keep its ETag")
	}
	writeFile(t, filepath.Join(dir, "lib.typ"), "#let x = 22")
	changed, _ := p.Archive(context.Background(), spec)
	if changed.ETag == a.ETag {
		t.Fatal("editing a file must change the ETag")
	}

	missing, _ := ParseSpec("local", "nothere", "0.1.0")
	if _, err := p.Archive(context.Background(), missing); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func tarGz(t *testing.T, entries map[string]string, extra ...*tar.Header) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range entries {
		_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg})
		_, _ = tw.Write([]byte(body))
	}
	for _, h := range extra {
		_ = tw.WriteHeader(h)
	}
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

func TestArchiveDownloadsPreviewIntoCache(t *testing.T) {
	cache := t.TempDir()
	calls := 0
	p := &Packages{
		CacheDir: func() string { return cache },
		Download: func(ctx context.Context, spec Spec) (io.ReadCloser, error) {
			calls++
			return io.NopCloser(bytes.NewReader(tarGz(t, map[string]string{
				"typst.toml": "[package]", "src/lib.typ": "ok",
			}))), nil
		},
	}
	spec, _ := ParseSpec("preview", "demo", "1.2.3")
	a, err := p.Archive(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if !a.Immutable {
		t.Fatal("registry packages are immutable")
	}
	if untar(t, a.Data)["src/lib.typ"] != "ok" {
		t.Fatal("archive missing downloaded file")
	}
	if _, err := os.Stat(filepath.Join(cache, "preview", "demo", "1.2.3", "typst.toml")); err != nil {
		t.Fatalf("package not unpacked into the cache: %v", err)
	}
	if _, err := p.Archive(context.Background(), spec); err != nil || calls != 1 {
		t.Fatalf("second request should hit the cache (calls=%d, err=%v)", calls, err)
	}
}

func TestDownloadRejectsTraversal(t *testing.T) {
	cache := t.TempDir()
	p := &Packages{
		CacheDir: func() string { return cache },
		Download: func(ctx context.Context, spec Spec) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(tarGz(t, map[string]string{
				"typst.toml": "x", "../../escape.txt": "pwned",
			}))), nil
		},
	}
	spec, _ := ParseSpec("preview", "evil", "0.0.1")
	if _, err := p.Archive(context.Background(), spec); err == nil {
		t.Fatal("traversal entry must be rejected")
	}
	if _, err := os.Stat(filepath.Join(cache, "preview", "escape.txt")); err == nil {
		t.Fatal("file escaped the package directory")
	}
	if _, err := os.Stat(filepath.Join(cache, "preview", "evil", "0.0.1")); err == nil {
		t.Fatal("failed download must not leave a package behind")
	}
}

func TestLocalNamespaceIsNeverDownloaded(t *testing.T) {
	p := &Packages{
		CacheDir: func() string { return t.TempDir() },
		Download: func(ctx context.Context, spec Spec) (io.ReadCloser, error) {
			t.Fatal("must not download non-preview packages")
			return nil, nil
		},
	}
	spec, _ := ParseSpec("local", "x", "0.1.0")
	if _, err := p.Archive(context.Background(), spec); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestFailedDownloadIsNotRetriedImmediately(t *testing.T) {
	calls := 0
	p := &Packages{
		CacheDir: func() string { return t.TempDir() },
		Download: func(ctx context.Context, spec Spec) (io.ReadCloser, error) {
			calls++
			return nil, ErrNotFound
		},
	}
	spec, _ := ParseSpec("preview", "nope", "1.0.0")
	for i := 0; i < 3; i++ {
		if _, err := p.Archive(context.Background(), spec); !errors.Is(err, ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	}
	if calls != 1 {
		t.Fatalf("registry asked %d times, want 1", calls)
	}
}
