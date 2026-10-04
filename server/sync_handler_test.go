package server

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncTargetPathRejectsEscapeAndMetadata(t *testing.T) {
	root := t.TempDir()
	cases := map[string]bool{
		"main.typ":          true,
		"sub/a.typ":         true,
		"../outside.typ":    true, // làm sạch thành /outside.typ, vẫn nằm trong root
		"/../../etc/passwd": true, // tương tự, không thoát khỏi root
		".git/config":       false,
		".typstify/token":   false,
		"":                  false, // gốc project
	}
	for raw, wantOK := range cases {
		rec := httptest.NewRecorder()
		abs, _, ok := syncTargetPath(rec, root, raw)
		if ok != wantOK {
			t.Errorf("syncTargetPath(%q) ok = %v, want %v (status %d)", raw, ok, wantOK, rec.Code)
		}
		if ok && !within(root, abs) {
			t.Errorf("syncTargetPath(%q) = %q, nằm ngoài root %q", raw, abs, root)
		}
	}
}

func TestWriteFileAtomicReplacesContent(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "main.typ")
	if err := os.WriteFile(p, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(p, []byte("new content")); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "new content" {
		t.Fatalf("content = %q", got)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temp file left behind: %d entries", len(entries))
	}
}
