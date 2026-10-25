package server

import (
	"encoding/base32"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Regression tests for the 2026-10-25 security review of multi-user
// sharing and MFA (docs/HISTORY.md).

func TestServerPasswordIsNotAnAdminLoginOnceAccountsExist(t *testing.T) {
	a := newAuthManager("srv-pass", t.TempDir())
	if w := loginReq(a, `{"password":"srv-pass"}`); w.Code != http.StatusOK {
		t.Fatalf("bootstrap login with server password failed: %d", w.Code)
	}
	if !a.isAdmin("admin") {
		t.Fatal("bootstrap admin should be admin while no accounts exist")
	}

	h, _ := newPasswordHash("pw1234")
	a.users["alice"] = User{Username: "alice", PasswordHash: h, Admin: true}
	a.users["bob"] = User{Username: "bob", PasswordHash: h}
	if w := loginReq(a, `{"password":"srv-pass"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("server password still logs in with accounts present: %d", w.Code)
	}
	if a.isAdmin("admin") {
		t.Fatal("legacy admin pseudo-account must not be admin once accounts exist")
	}
}

func TestMFALocksAccountAfterRepeatedWrongCodes(t *testing.T) {
	a := newAuthManager("", t.TempDir())
	h, _ := newPasswordHash("pw1234")
	secret := []byte("12345678901234567890")
	b32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
	a.users["mfa"] = User{Username: "mfa", PasswordHash: h, MFASecret: b32, MFAEnabled: true}

	for range mfaMaxFailures {
		if ok, _ := a.checkSecondFactor("mfa", "000000"); ok {
			t.Fatal("wrong code accepted")
		}
	}
	// Even the correct code is refused during the lockout window, and a
	// login on another account cannot reset it.
	if ok, locked := a.checkSecondFactor("mfa", totpCode(secret, time.Now())); ok || !locked {
		t.Fatalf("locked account accepted code (ok=%v locked=%v)", ok, locked)
	}
}

func TestNonMemberCannotReachProjectThroughParentOrChildFolder(t *testing.T) {
	root, err := os.MkdirTemp("", "typstify-sec-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	s := New(testService(), Options{ProjectRoot: root, StorageDir: tempStorage(t)})
	h := s.Handler()

	owner := &apiClient{t: t, h: h}
	mustStatus(t, owner.do("POST", "/api/auth/register", `{"username":"owner","password":"pw1234"}`), 200, "register owner")
	ownerCookie := owner.cookie
	mustStatus(t, owner.do("POST", "/api/auth/register", `{"username":"mallory","password":"pw1234"}`), 200, "register mallory")
	owner.cookie = ownerCookie

	w := owner.do("POST", "/api/projects", `{"name":"secret"}`)
	mustStatus(t, w, 200, "create project")
	var created projectView
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	p, _ := s.projects.Get(created.ID)
	child := filepath.Join(p.Dir, "chapters")
	_ = os.MkdirAll(child, 0o755)

	m := &apiClient{t: t, h: h}
	mustStatus(t, m.do("POST", "/api/auth/login", `{"username":"mallory","password":"pw1234"}`), 200, "mallory login")

	body := func(dir string) string {
		b, _ := json.Marshal(map[string]string{"path": dir})
		return string(b)
	}
	parent := filepath.Dir(p.Dir)
	mustStatus(t, m.do("POST", "/api/workspace/open", body(parent)), 403, "open parent of project")
	mustStatus(t, m.do("POST", "/api/workspace/open", body(child)), 403, "open child of project")
	mustStatus(t, m.do("POST", "/api/workspace/create", body(filepath.Join(p.Dir, "x"))), 403, "create inside project")

	// Creating on an existing, unregistered folder must not claim it.
	legacy := filepath.Join(root, "legacy")
	_ = os.MkdirAll(legacy, 0o755)
	mustStatus(t, m.do("POST", "/api/workspace/create", body(legacy)), 200, "create on existing folder")
	if _, ok := s.projects.ByDir(legacy); ok {
		t.Fatal("existing folder was claimed by create")
	}

	// Owner reaching a child folder of their own project is fine, and a
	// file there is still governed by the project.
	mustStatus(t, owner.do("POST", "/api/workspace/open", body(child)), 200, "owner opens child")
	mustStatus(t, m.do("GET", "/api/workspace/file?path=a.typ", ""), 403, "mallory reads via owner's open child")
}

func TestMFAEnableRequiresPassword(t *testing.T) {
	a := newAuthManager("", t.TempDir())
	reg := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"username":"user1","password":"pw1234"}`))
	rw := httptest.NewRecorder()
	a.handleRegister(rw, reg)
	cookie := rw.Result().Cookies()[0]

	setup := httptest.NewRequest("POST", "/api/auth/mfa/setup", nil)
	setup.AddCookie(cookie)
	a.handleMFASetup(httptest.NewRecorder(), setup)
	raw, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(a.users["user1"].MFAPendingSecret)

	enable := httptest.NewRequest("POST", "/api/auth/mfa/enable", strings.NewReader(`{"code":"`+totpCode(raw, time.Now())+`"}`))
	enable.AddCookie(cookie)
	ew := httptest.NewRecorder()
	a.handleMFAEnable(ew, enable)
	if ew.Code != http.StatusBadRequest || a.users["user1"].MFAEnabled {
		t.Fatalf("enable without password: %d enabled=%v", ew.Code, a.users["user1"].MFAEnabled)
	}
}
