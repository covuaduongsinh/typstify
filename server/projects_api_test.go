package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"looz.ws/typstify/service"
)

var (
	testServiceOnce sync.Once
	testServiceInst *service.ServiceFacade
)

// testService returns one ServiceFacade shared by all tests in this
// package: each NewService opens the same on-disk bbolt workspace DB, and
// a second open in the same process times out and log.Fatal()s.
func testService() *service.ServiceFacade {
	testServiceOnce.Do(func() { testServiceInst = service.NewService(context.Background()) })
	return testServiceInst
}

// tempStorage is a server storage dir removed best-effort: background
// writers (session/project files written via temp file + rename) can still
// be finishing when the test returns, which makes t.TempDir's strict
// cleanup fail on Windows.
func tempStorage(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "typstify-storage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

type apiClient struct {
	t      *testing.T
	h      http.Handler
	cookie *http.Cookie
}

func (c *apiClient) do(method, path, body string) *httptest.ResponseRecorder {
	c.t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	w := httptest.NewRecorder()
	c.h.ServeHTTP(w, req)
	for _, ck := range w.Result().Cookies() {
		if ck.Name == sessionCookieName && ck.Value != "" {
			c.cookie = ck
		}
	}
	return w
}

func mustStatus(t *testing.T, w *httptest.ResponseRecorder, want int, what string) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("%s: status %d, want %d: %s", what, w.Code, want, w.Body.String())
	}
}

func TestProjectSharingAndRoleEnforcement(t *testing.T) {
	// Opening a project starts tinymist and a file watcher on it, which on
	// Windows keep the directory busy past the test, so t.TempDir's strict
	// cleanup would fail; remove it best-effort instead.
	root, err := os.MkdirTemp("", "typstify-projects-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	s := New(testService(), Options{ProjectRoot: root, StorageDir: tempStorage(t)})
	h := s.Handler()

	owner := &apiClient{t: t, h: h}
	mustStatus(t, owner.do("POST", "/api/auth/register", `{"username":"owner","password":"pw1234"}`), 200, "register owner")
	// The owner (admin) registers a second account; that response logs the
	// browser in as the new user, so keep the owner's cookie aside.
	ownerCookie := owner.cookie
	mustStatus(t, owner.do("POST", "/api/auth/register", `{"username":"guest","password":"pw1234"}`), 200, "register guest")
	owner.cookie = ownerCookie

	w := owner.do("POST", "/api/projects", `{"name":"Sách Cờ Vua"}`)
	mustStatus(t, w, 200, "create project")
	var created projectView
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	if p, _ := s.projects.Get(created.ID); created.Role != "owner" || !strings.HasPrefix(p.Dir, root) || strings.Contains(w.Body.String(), `"dir"`) {
		t.Fatalf("created %+v", created)
	}

	w = owner.do("POST", "/api/projects/"+created.ID+"/shares", `{"role":"viewer","expiresInDays":7}`)
	mustStatus(t, w, 200, "create share")
	var share struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &share)
	if share.Token == "" || strings.Contains(w.Body.String(), "tokenHash\":\"") {
		t.Fatalf("share response %s", w.Body.String())
	}

	guest := &apiClient{t: t, h: h}
	mustStatus(t, guest.do("POST", "/api/auth/login", `{"username":"guest","password":"pw1234"}`), 200, "guest login")

	// Before joining, the guest cannot see or open the project.
	if w := guest.do("GET", "/api/projects", ""); strings.Contains(w.Body.String(), created.ID) {
		t.Fatal("guest sees project before joining")
	}
	mustStatus(t, guest.do("POST", "/api/projects/"+created.ID+"/open", ""), 404, "open before join")

	mustStatus(t, guest.do("POST", "/api/projects/join", `{"token":"`+share.Token+`"}`), 200, "join")
	mustStatus(t, guest.do("POST", "/api/projects/"+created.ID+"/open", ""), 200, "open after join")

	mustStatus(t, guest.do("GET", "/api/workspace/file?path=main.typ", ""), 200, "viewer reads")
	mustStatus(t, guest.do("PUT", "/api/workspace/file?path=main.typ", "hacked"), 403, "viewer writes")
	mustStatus(t, guest.do("POST", "/api/workspace/rename", `{"from":"main.typ","to":"x.typ"}`), 403, "viewer renames")
	mustStatus(t, guest.do("POST", "/api/projects/"+created.ID+"/shares", `{"role":"editor"}`), 403, "viewer shares")
	mustStatus(t, guest.do("PUT", "/api/settings/typst", `{}`), 403, "non-admin settings")

	if w := guest.do("GET", "/api/workspace/role", ""); !strings.Contains(w.Body.String(), `"role":"viewer"`) {
		t.Fatalf("workspace role %s", w.Body.String())
	}

	// Owner promotes guest to editor; writes now succeed.
	mustStatus(t, owner.do("PUT", "/api/projects/"+created.ID+"/members/guest", `{"role":"editor"}`), 200, "promote")
	mustStatus(t, guest.do("PUT", "/api/workspace/file?path=main.typ", "= ok"), 200, "editor writes")

	mustStatus(t, owner.do("DELETE", "/api/projects/"+created.ID+"/members/owner", ""), 400, "remove owner")
	mustStatus(t, owner.do("PUT", "/api/projects/"+created.ID+"/members/nobody", `{"role":"viewer"}`), 404, "unknown user")

	// Revoked access is enforced on the already-open project.
	mustStatus(t, owner.do("DELETE", "/api/projects/"+created.ID+"/members/guest", ""), 200, "remove guest")
	mustStatus(t, guest.do("GET", "/api/workspace/file?path=main.typ", ""), 403, "removed member reads")
}

func TestProjectSlug(t *testing.T) {
	cases := map[string]string{
		"Sách Cờ Vua":      "sách-cờ-vua",
		"  ../../etc  ":    "etc",
		"***":              "du-an",
		"Bài 1: Khai cuộc": "bài-1-khai-cuộc",
	}
	for in, want := range cases {
		if got := projectSlug(in); got != want {
			t.Errorf("projectSlug(%q) = %q, want %q", in, got, want)
		}
	}
}
