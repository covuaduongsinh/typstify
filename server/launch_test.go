package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// desktopReq is a request as the desktop window (or a page on the machine)
// sends it to the sidecar on 127.0.0.1.
func desktopReq(c *apiClient, method, path, body string, header map[string]string) *httptest.ResponseRecorder {
	c.t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	r.Host = "127.0.0.1:62187"
	for k, v := range header {
		if k == "Host" {
			r.Host = v
		} else {
			r.Header.Set(k, v)
		}
	}
	if c.cookie != nil {
		r.AddCookie(c.cookie)
	}
	w := httptest.NewRecorder()
	c.h.ServeHTTP(w, r)
	for _, ck := range w.Result().Cookies() {
		if ck.Name == sessionCookieName && ck.Value != "" {
			c.cookie = ck
		}
	}
	return w
}

// withCookie is a request carrying the client's session cookie.
func withCookie(c *apiClient) *http.Request {
	r := httptest.NewRequest("GET", "/", nil)
	if c.cookie != nil {
		r.AddCookie(c.cookie)
	}
	return r
}

func TestLaunchTokenSignsDesktopInOnce(t *testing.T) {
	s := New(testService(), Options{StorageDir: tempStorage(t), LaunchToken: "launch-secret"})
	c := &apiClient{t: t, h: s.Handler()}

	// The desktop account exists before anything can reach the port, and
	// nobody can register without a session (no bootstrap race).
	if u, ok := s.auth.users[desktopUser]; !ok || !u.Admin {
		t.Fatalf("desktop account: %+v %v", u, ok)
	}
	if w := desktopReq(c, "POST", "/api/auth/register", `{"username":"intruder","password":"pw1234"}`, nil); w.Code != 403 {
		t.Fatalf("bootstrap register in desktop mode: %d", w.Code)
	}
	if w := desktopReq(c, "GET", "/api/workspace/tree", "", nil); w.Code != 401 {
		t.Fatalf("no session: %d", w.Code)
	}
	if w := desktopReq(c, "GET", "/api/auth/launch?token=wrong", "", nil); w.Code != 403 {
		t.Fatalf("wrong token: %d", w.Code)
	}
	// Wrong tokens do not lock the real window out.
	for range 6 {
		desktopReq(&apiClient{t: t, h: c.h}, "GET", "/api/auth/launch?token=x", "", nil)
	}

	w := desktopReq(c, "GET", "/api/auth/launch?token=launch-secret", "", nil)
	if w.Code != 303 || w.Header().Get("Location") != "/" || c.cookie == nil {
		t.Fatalf("launch: %d %q cookie=%v", w.Code, w.Header().Get("Location"), c.cookie)
	}
	if user, ok := s.auth.authenticatedUser(withCookie(c)); !ok || user != desktopUser {
		t.Fatalf("session user = %q %v", user, ok)
	}
	// Once: a token found later in history is useless.
	other := &apiClient{t: t, h: c.h}
	if w := desktopReq(other, "GET", "/api/auth/launch?token=launch-secret", "", nil); w.Code != 403 || other.cookie != nil {
		t.Fatalf("second use: %d", w.Code)
	}
	if w := desktopReq(c, "GET", "/api/auth/status", "", nil); !strings.Contains(w.Body.String(), `"desktop":true`) {
		t.Fatalf("status: %s", w.Body.String())
	}
}

func TestDesktopGuardRefusesForeignHostsAndOrigins(t *testing.T) {
	s := New(testService(), Options{StorageDir: tempStorage(t), LaunchToken: "tok"})
	c := &apiClient{t: t, h: s.Handler()}
	desktopReq(c, "GET", "/api/auth/launch?token=tok", "", nil)

	// DNS rebinding: a foreign Host is refused.
	if w := desktopReq(c, "GET", "/api/workspace/tree", "", map[string]string{"Host": "evil.example:62187"}); w.Code != 403 {
		t.Fatalf("foreign host: %d", w.Code)
	}
	// CSRF: a write from another origin is refused, the app's own passes.
	if w := desktopReq(c, "POST", "/api/projects", `{"name":"x"}`, map[string]string{"Origin": "https://evil.example"}); w.Code != 403 {
		t.Fatalf("foreign origin: %d", w.Code)
	}
	if w := desktopReq(c, "POST", "/api/projects", `{"name":"Dự án"}`, map[string]string{"Origin": "http://127.0.0.1:62187"}); w.Code != 200 {
		t.Fatalf("own origin: %d %s", w.Code, w.Body.String())
	}
}

func TestLaunchUsesExistingAdmin(t *testing.T) {
	storage := tempStorage(t)
	// An account store from before (e.g. the server used from a browser).
	first := New(testService(), Options{StorageDir: storage})
	fc := &apiClient{t: t, h: first.Handler()}
	mustStatus(t, fc.do("POST", "/api/auth/register", `{"username":"owner","password":"pw1234"}`), 200, "register")

	s := New(testService(), Options{StorageDir: storage, LaunchToken: "tok"})
	c := &apiClient{t: t, h: s.Handler()}
	if w := desktopReq(c, "GET", "/api/auth/launch?token=tok", "", nil); w.Code != 303 {
		t.Fatalf("launch: %d", w.Code)
	}
	if user, _ := s.auth.authenticatedUser(withCookie(c)); user != "owner" {
		t.Fatalf("launch signed in as %q, want the existing admin", user)
	}
	if _, ok := s.auth.users[desktopUser]; ok {
		t.Fatal("a desktop account was created next to an existing admin")
	}
}

func TestLaunchRefusedWithoutToken(t *testing.T) {
	s := New(testService(), Options{StorageDir: tempStorage(t), Password: "pw"})
	c := &apiClient{t: t, h: s.Handler()}
	mustStatus(t, c.do("GET", "/api/auth/launch?token=", ""), 403, "unconfigured")
}
