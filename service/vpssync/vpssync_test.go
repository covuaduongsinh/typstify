package vpssync

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"looz.ws/typstify/service/settings"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestScanManifestSkipsIgnoredDirs(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"main.typ":             "= hi",
		"sub/a.typ":            "a",
		".git/HEAD":            "ref",
		"node_modules/x.js":    "x",
		"dist/out.pdf":         "pdf",
		".typstify/token.json": "secret",
	})

	items, err := ScanManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, it := range items {
		got = append(got, it.Path)
	}
	sort.Strings(got)
	want := []string{"main.typ", "sub/a.typ"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("ScanManifest paths = %v, want %v", got, want)
	}
}

func TestIsIgnoredPatterns(t *testing.T) {
	patterns := []string{"*.bak", "drafts/*"}
	cases := map[string]bool{
		"main.typ":        false,
		"notes.bak":       true,
		"sub/old.bak":     true,
		"drafts/wip.typ":  true,
		".git/config":     true,
		"sub/.typstify/x": true,
	}
	for rel, want := range cases {
		if got := IsIgnored(rel, patterns); got != want {
			t.Errorf("IsIgnored(%q) = %v, want %v", rel, got, want)
		}
	}
}

// fakeVPS mô phỏng các endpoint /api/sync/* ở mức đủ để kiểm tra delta.
type fakeVPS struct {
	files  map[string]FileItem
	pushed []string
	token  string
}

func (f *fakeVPS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+f.token {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "authentication required"})
		return
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/sync/manifest":
		items := []FileItem{}
		for _, it := range f.files {
			items = append(items, it)
		}
		_ = json.NewEncoder(w).Encode(items)
	case r.Method == http.MethodPost && r.URL.Path == "/api/sync/push":
		data, _ := io.ReadAll(r.Body)
		rel := r.URL.Query().Get("path")
		f.pushed = append(f.pushed, rel)
		f.files[rel] = FileItem{Path: rel, Hash: hashBytes(data)}
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func hashBytes(data []byte) string {
	p := filepath.Join(os.TempDir(), "vpssync-hash-tmp")
	_ = os.WriteFile(p, data, 0600)
	h, _ := hashFile(p)
	_ = os.Remove(p)
	return h
}

func newTestEngine(t *testing.T, root, url, token string, patterns []string) *SyncEngine {
	t.Helper()
	cfg := &settings.VPSSyncSettings{ServerURL: url, Token: token, IgnorePatterns: patterns}
	return NewSyncEngine(func() *settings.VPSSyncSettings { return cfg }, func() string { return root })
}

func TestPerformSyncUploadsOnlyChangedFiles(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"main.typ":  "= main",
		"same.typ":  "unchanged",
		"skip.bak":  "ignored",
		".git/HEAD": "ref",
	})

	vps := &fakeVPS{files: map[string]FileItem{}, token: "tok"}
	// same.typ đã có trên VPS với đúng nội dung.
	vps.files["same.typ"] = FileItem{Path: "same.typ", Hash: hashBytes([]byte("unchanged"))}
	srv := httptest.NewServer(vps)
	defer srv.Close()

	e := newTestEngine(t, root, srv.URL, "tok", []string{"*.bak"})
	res, err := e.PerformSync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Uploaded) != 1 || res.Uploaded[0] != "main.typ" {
		t.Fatalf("Uploaded = %v, want [main.typ]", res.Uploaded)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	if st := e.Status(); st.Syncing || st.LastError != "" || st.LastSyncTime.IsZero() {
		t.Fatalf("status after success = %+v", st)
	}
}

func TestTestConnectionRejectsBadToken(t *testing.T) {
	vps := &fakeVPS{files: map[string]FileItem{}, token: "tok"}
	srv := httptest.NewServer(vps)
	defer srv.Close()

	e := newTestEngine(t, t.TempDir(), srv.URL, "wrong", nil)
	if err := e.TestConnection(context.Background()); err == nil {
		t.Fatal("TestConnection with wrong token returned nil error")
	}
	e = newTestEngine(t, t.TempDir(), srv.URL, "tok", nil)
	if err := e.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection with right token: %v", err)
	}
}
