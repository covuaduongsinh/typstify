package server

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// Each browser session works on its own project
// (docs/plans/plan_phase3b_per_user_projects.md).

type sessionProjectsFixture struct {
	s          *Server
	owner, ed  *apiClient
	idX, idY   string
	dirX, dirY string
}

func newSessionProjectsFixture(t *testing.T) *sessionProjectsFixture {
	t.Helper()
	root, err := os.MkdirTemp("", "typstify-sessions-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	s := New(testService(), Options{ProjectRoot: root, StorageDir: tempStorage(t)})
	h := s.Handler()

	owner := &apiClient{t: t, h: h}
	mustStatus(t, owner.do("POST", "/api/auth/register", `{"username":"owner","password":"pw1234"}`), 200, "register owner")
	ownerCookie := owner.cookie
	mustStatus(t, owner.do("POST", "/api/auth/register", `{"username":"editor1","password":"pw1234"}`), 200, "register ed")
	owner.cookie = ownerCookie

	create := func(name string) string {
		w := owner.do("POST", "/api/projects", `{"name":"`+name+`"}`)
		mustStatus(t, w, 200, "create "+name)
		var v projectView
		_ = json.Unmarshal(w.Body.Bytes(), &v)
		return v.ID
	}
	f := &sessionProjectsFixture{s: s, owner: owner, idX: create("Sách X"), idY: create("Sách Y")}
	px, _ := s.projects.Get(f.idX)
	py, _ := s.projects.Get(f.idY)
	f.dirX, f.dirY = px.Dir, py.Dir

	// ed edits X but may only read Y.
	mustStatus(t, owner.do("PUT", "/api/projects/"+f.idX+"/members/editor1", `{"role":"editor"}`), 200, "ed edits X")
	mustStatus(t, owner.do("PUT", "/api/projects/"+f.idY+"/members/editor1", `{"role":"viewer"}`), 200, "ed views Y")

	f.ed = &apiClient{t: t, h: h}
	mustStatus(t, f.ed.do("POST", "/api/auth/login", `{"username":"editor1","password":"pw1234"}`), 200, "ed login")
	return f
}

func currentPath(c *apiClient) string {
	var cur struct{ Path string }
	_ = json.Unmarshal(c.do("GET", "/api/workspace/current", "").Body.Bytes(), &cur)
	return cur.Path
}

func TestSessionsKeepTheirOwnProject(t *testing.T) {
	f := newSessionProjectsFixture(t)
	mustStatus(t, f.ed.do("POST", "/api/projects/"+f.idX+"/open", ""), 200, "ed opens X")
	mustStatus(t, f.owner.do("POST", "/api/projects/"+f.idY+"/open", ""), 200, "owner opens Y")

	if got := currentPath(f.ed); !sameDir(got, f.dirX) {
		t.Fatalf("ed's project = %q after the owner opened Y, want X %q", got, f.dirX)
	}
	if got := currentPath(f.owner); !sameDir(got, f.dirY) {
		t.Fatalf("owner's project = %q, want Y %q", got, f.dirY)
	}

	// ed is an editor on X: the write goes to X, judged by ed's role on X
	// (before, the role was taken on whatever project was opened last).
	mustStatus(t, f.ed.do("PUT", "/api/workspace/file?path=main.typ", "= viết vào X"), 200, "ed writes in X")
	mustStatus(t, f.owner.do("PUT", "/api/workspace/file?path=main.typ", "= viết vào Y"), 200, "owner writes in Y")

	x, _ := os.ReadFile(f.dirX + "/main.typ")
	y, _ := os.ReadFile(f.dirY + "/main.typ")
	if string(x) != "= viết vào X" || string(y) != "= viết vào Y" {
		t.Fatalf("X = %q, Y = %q", x, y)
	}
	if w := f.ed.do("GET", "/api/workspace/file?path=main.typ", ""); !strings.Contains(w.Body.String(), "viết vào X") {
		t.Fatalf("ed reads %s", w.Body.String())
	}
}

func TestRoleIsCheckedOnTheSessionsProject(t *testing.T) {
	f := newSessionProjectsFixture(t)
	mustStatus(t, f.ed.do("POST", "/api/projects/"+f.idY+"/open", ""), 200, "ed opens Y")
	mustStatus(t, f.owner.do("POST", "/api/projects/"+f.idX+"/open", ""), 200, "owner opens X")
	// ed may only read Y. Before, opening X switched the whole server, so
	// ed's write was judged on X (editor) and landed in X.
	mustStatus(t, f.ed.do("PUT", "/api/workspace/file?path=main.typ", "x"), 403, "ed writes in Y")
	if b, _ := os.ReadFile(f.dirX + "/main.typ"); string(b) == "x" {
		t.Fatal("ed's write landed in X")
	}
}

func TestStaleTabGetsConflict(t *testing.T) {
	f := newSessionProjectsFixture(t)
	mustStatus(t, f.ed.do("POST", "/api/projects/"+f.idX+"/open", ""), 200, "tab 1 opens X")
	mustStatus(t, f.ed.do("POST", "/api/projects/"+f.idY+"/open", ""), 200, "tab 2 opens Y")

	read := func(project string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/api/workspace/tree", nil)
		req.AddCookie(f.ed.cookie)
		req.Header.Set(projectHeader, url.PathEscape(project))
		w := httptest.NewRecorder()
		f.s.Handler().ServeHTTP(w, req)
		return w
	}
	if w := read(f.dirX); w.Code != 409 || !strings.Contains(w.Body.String(), "tab khác") {
		t.Fatalf("tab still on X: %d %s, want 409", w.Code, w.Body.String())
	}
	mustStatus(t, read(f.dirY), 200, "tab on Y")
}

func TestWithoutSessionsOpenSetsTheDefaultProject(t *testing.T) {
	root := t.TempDir()
	s := New(testService(), Options{ProjectRoot: root, StorageDir: tempStorage(t)})
	c := &apiClient{t: t, h: s.Handler()}
	dir := root + "/sach"
	mustStatus(t, c.do("POST", "/api/workspace/create", `{"path":`+jsonString(dir)+`}`), 200, "create without auth")
	if got := currentPath(c); !sameDir(got, dir) {
		t.Fatalf("project = %q, want %q", got, dir)
	}
	if got := s.appSrv.CurrentProjectDir(); !sameDir(got, dir) {
		t.Fatalf("server default = %q, want %q", got, dir)
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestActiveDocumentIsPerProject(t *testing.T) {
	s := New(testService(), Options{ProjectRoot: t.TempDir(), StorageDir: tempStorage(t)})
	x, y := t.TempDir(), t.TempDir()
	s.SetActiveFile(x+"/chuong1.typ", 0)
	s.SetActiveFile(y+"/bai-tap.typ", 3)
	if got := s.activeDocumentFor(x).File; got != x+"/chuong1.typ" {
		t.Fatalf("X's document = %q", got)
	}
	if got := s.activeDocumentFor(y); got.File != y+"/bai-tap.typ" || got.CursorPos != 3 {
		t.Fatalf("Y's document = %+v", got)
	}
}
