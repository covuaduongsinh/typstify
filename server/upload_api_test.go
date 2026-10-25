package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploadName(t *testing.T) {
	ok := map[string]string{
		"Ảnh chụp màn hình.png": "Ảnh-chụp-màn-hình.png",
		"../../etc/passwd.txt":  "passwd.txt",
		`C:\x\y\bàn cờ.svg`:     "bàn-cờ.svg",
		".hidden.png":           "hidden.png",
		"Tiếng.png":           "Tiếng.png", // decomposed (macOS)
	}
	for in, want := range ok {
		if got, err := uploadName(in); err != nil || got != want {
			t.Errorf("%q -> %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "noext", ".", "..", "$$$"} {
		if _, err := uploadName(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestUploadAPI(t *testing.T) {
	root, err := os.MkdirTemp("", "typstify-upload-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	s := New(testService(), Options{ProjectRoot: root, StorageDir: tempStorage(t)})
	h := s.Handler()
	owner := &apiClient{t: t, h: h}
	mustStatus(t, owner.do("POST", "/api/auth/register", `{"username":"owner","password":"pw1234"}`), 200, "register")
	w := owner.do("POST", "/api/projects", `{"name":"Tải lên"}`)
	var created projectView
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	mustStatus(t, owner.do("POST", "/api/projects/"+created.ID+"/open", ""), 200, "open")
	proj, _ := s.projects.Get(created.ID)

	post := func(q, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/workspace/upload?"+q, strings.NewReader(body))
		req.AddCookie(owner.cookie)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	var res struct{ Path string }
	w = post("dir=assets&name=logo.png", "PNG1")
	mustStatus(t, w, 200, "upload")
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Path != "assets/logo.png" {
		t.Fatalf("path %q", res.Path)
	}
	w = post("dir=assets&name=logo.png", "PNG2")
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Path != "assets/logo-1.png" {
		t.Fatalf("existing files must not be overwritten, got %q", res.Path)
	}
	if b, _ := os.ReadFile(filepath.Join(proj.Dir, "assets", "logo.png")); string(b) != "PNG1" {
		t.Fatal("first upload changed")
	}
	mustStatus(t, post("dir=.git&name=config.txt", "x"), 400, "protected dir")
	mustStatus(t, post("dir=assets&name=noext", "x"), 400, "bad name")
	mustStatus(t, post("dir=assets&name=big.bin", strings.Repeat("x", uploadBodyLimit+1)), 413, "too large")
}

// A font larger than the 2 MiB default body limit reaches the handler.
func TestFontUploadBodyLimit(t *testing.T) {
	var got int64
	h := withBodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, err := io.Copy(io.Discard, r.Body)
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		got = n
	}))
	body := strings.Repeat("x", 5<<20)
	req := httptest.NewRequest("POST", "/api/settings/fonts?filename=Big.ttf", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 || got != int64(len(body)) {
		t.Fatalf("5 MiB font: status %d, read %d", rec.Code, got)
	}
}
