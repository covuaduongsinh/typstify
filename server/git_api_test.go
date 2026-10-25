package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"

	"looz.ws/typstify/service/gitsync"
)

func TestGitAPI(t *testing.T) {
	root, err := os.MkdirTemp("", "typstify-git-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	bare := t.TempDir()
	if _, err := git.PlainInit(bare, true); err != nil {
		t.Fatal(err)
	}
	s := New(testService(), Options{ProjectRoot: root, StorageDir: tempStorage(t)})
	s.git = &gitsync.Service{Base: t.TempDir(), AllowLocal: true}
	h := s.Handler()

	owner := &apiClient{t: t, h: h}
	mustStatus(t, owner.do("POST", "/api/auth/register", `{"username":"owner","password":"pw1234"}`), 200, "register owner")
	ownerCookie := owner.cookie
	mustStatus(t, owner.do("POST", "/api/auth/register", `{"username":"editor1","password":"pw1234"}`), 200, "register editor")
	owner.cookie = ownerCookie

	w := owner.do("POST", "/api/projects", `{"name":"Git"}`)
	var created projectView
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	mustStatus(t, owner.do("PUT", "/api/projects/"+created.ID+"/members/editor1", `{"role":"editor"}`), 200, "add editor")
	mustStatus(t, owner.do("POST", "/api/projects/"+created.ID+"/open", ""), 200, "open")
	mustStatus(t, owner.do("PUT", "/api/workspace/file?path=main.typ", "= Một"), 200, "write")

	ed := &apiClient{t: t, h: h}
	mustStatus(t, ed.do("POST", "/api/auth/login", `{"username":"editor1","password":"pw1234"}`), 200, "editor login")

	body, _ := json.Marshal(map[string]string{"remote": bare, "branch": "main", "token": "s3cret"})
	mustStatus(t, ed.do("PUT", "/api/git/config", string(body)), 403, "editor configures")
	mustStatus(t, owner.do("PUT", "/api/git/config", string(body)), 200, "owner configures")
	if w := ed.do("GET", "/api/git", ""); strings.Contains(w.Body.String(), "s3cret") || !strings.Contains(w.Body.String(), `"hasToken":true`) {
		t.Fatalf("status must not leak the token: %s", w.Body.String())
	}

	mustStatus(t, ed.do("POST", "/api/git/commit", `{"message":""}`), 400, "empty message")
	mustStatus(t, ed.do("POST", "/api/git/commit", `{"message":"Bản đầu"}`), 200, "commit")
	mustStatus(t, ed.do("POST", "/api/git/commit", `{"message":"lại"}`), 400, "nothing to commit")
	mustStatus(t, ed.do("POST", "/api/git/push", `{}`), 200, "push")
	mustStatus(t, ed.do("POST", "/api/git/push", `{"force":true}`), 403, "editor force-pushes")

	w = ed.do("GET", "/api/git?fetch=1", "")
	var st struct {
		Status struct {
			Initialized   bool
			Ahead, Behind int
			Head          struct{ Message string }
		}
	}
	_ = json.Unmarshal(w.Body.Bytes(), &st)
	if !st.Status.Initialized || st.Status.Ahead != 0 || st.Status.Behind != 0 || st.Status.Head.Message != "Bản đầu" {
		t.Fatalf("status %s", w.Body.String())
	}

	mustStatus(t, ed.do("POST", "/api/git/pull", `{}`), 200, "pull (up to date)")

	w = owner.do("GET", "/api/history", "")
	if !strings.Contains(w.Body.String(), "Trước khi kéo từ Git") {
		t.Fatalf("pull should save a history version first: %s", w.Body.String())
	}

	// Review finding: a subfolder of the project must not reuse the
	// project's Git settings or history (paths are relative to the root).
	proj, _ := s.projects.Get(created.ID)
	sub := filepath.Join(proj.Dir, "chuong")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	openBody, _ := json.Marshal(map[string]string{"path": sub})
	mustStatus(t, owner.do("POST", "/api/workspace/open", string(openBody)), 200, "open subfolder")
	mustStatus(t, ed.do("POST", "/api/git/commit", `{"message":"x"}`), 409, "git in a subfolder")
	if w := owner.do("GET", "/api/history", ""); strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("subfolder must have its own history, got %s", w.Body.String())
	}
}
