package server

import (
	"encoding/base32"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// RFC 6238 appendix B SHA-1 vectors, truncated to 6 digits.
func TestTOTPRFC6238Vectors(t *testing.T) {
	secret := []byte("12345678901234567890")
	cases := map[int64]string{
		59:         "287082",
		1111111109: "081804",
		1111111111: "050471",
		1234567890: "005924",
		2000000000: "279037",
	}
	for unix, want := range cases {
		if got := totpCode(secret, time.Unix(unix, 0)); got != want {
			t.Errorf("t=%d: got %s want %s", unix, got, want)
		}
	}
}

func TestTOTPVerifyAllowsOneStepSkew(t *testing.T) {
	secret := []byte("12345678901234567890")
	b32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
	now := time.Unix(1234567890, 0)
	if !totpVerify(b32, totpCode(secret, now.Add(-30*time.Second)), now) {
		t.Fatal("previous step rejected")
	}
	if !totpVerify(b32, totpCode(secret, now.Add(30*time.Second)), now) {
		t.Fatal("next step rejected")
	}
	if totpVerify(b32, totpCode(secret, now.Add(-90*time.Second)), now) {
		t.Fatal("code three steps old accepted")
	}
	if totpVerify(b32, "abc", now) || totpVerify("!!", "123456", now) {
		t.Fatal("garbage accepted")
	}
}

func loginReq(a *authManager, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	w := httptest.NewRecorder()
	a.handleLogin(w, req)
	return w
}

func TestLoginRequiresMFAWhenEnabled(t *testing.T) {
	a := newAuthManager("", t.TempDir())
	h, _ := newPasswordHash("pw1234")
	secret := []byte("12345678901234567890")
	b32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
	recovery, _ := newPasswordHash("abcde-12345")
	a.users["mfa"] = User{Username: "mfa", PasswordHash: h, MFASecret: b32, MFAEnabled: true,
		RecoveryCodes: []string{recovery}}

	w := loginReq(a, `{"username":"mfa","password":"pw1234"}`)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), `"mfaRequired":true`) {
		t.Fatalf("missing code: %d %s", w.Code, w.Body.String())
	}
	w = loginReq(a, `{"username":"mfa","password":"pw1234","code":"000000"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong code accepted: %d", w.Code)
	}
	code := totpCode(secret, time.Now())
	w = loginReq(a, `{"username":"mfa","password":"pw1234","code":"`+code+`"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("valid code rejected: %d %s", w.Code, w.Body.String())
	}

	// A recovery code works once.
	w = loginReq(a, `{"username":"mfa","password":"pw1234","code":"abcde-12345"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("recovery code rejected: %d %s", w.Code, w.Body.String())
	}
	w = loginReq(a, `{"username":"mfa","password":"pw1234","code":"abcde-12345"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("recovery code reused: %d", w.Code)
	}
}

func TestMFAEnableFlow(t *testing.T) {
	a := newAuthManager("", t.TempDir())
	reg := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"username":"owner","password":"pw1234"}`))
	rw := httptest.NewRecorder()
	a.handleRegister(rw, reg)
	cookie := rw.Result().Cookies()[0]

	setup := httptest.NewRequest("POST", "/api/auth/mfa/setup", nil)
	setup.AddCookie(cookie)
	sw := httptest.NewRecorder()
	a.handleMFASetup(sw, setup)
	if sw.Code != http.StatusOK || !strings.Contains(sw.Body.String(), "otpauth://totp/") {
		t.Fatalf("setup: %d %s", sw.Code, sw.Body.String())
	}

	pending := a.users["owner"].MFAPendingSecret
	raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(pending)
	if err != nil {
		t.Fatal(err)
	}
	enableCode := totpCode(raw, time.Now())
	enable := httptest.NewRequest("POST", "/api/auth/mfa/enable", strings.NewReader(`{"password":"pw1234","code":"`+enableCode+`"}`))
	enable.AddCookie(cookie)
	ew := httptest.NewRecorder()
	a.handleMFAEnable(ew, enable)
	if ew.Code != http.StatusOK || !strings.Contains(ew.Body.String(), "recoveryCodes") {
		t.Fatalf("enable: %d %s", ew.Code, ew.Body.String())
	}
	if u := a.users["owner"]; !u.MFAEnabled || len(u.RecoveryCodes) != mfaRecoveryCodeCount || u.MFAPendingSecret != "" {
		t.Fatalf("user after enable: %+v", u)
	}

	// The code that enabled MFA cannot be replayed. (Recomputing it here
	// made the test flaky: across a 30 s step boundary it is a new code.)
	replay := httptest.NewRequest("POST", "/api/auth/mfa/disable", strings.NewReader(`{"password":"pw1234","code":"`+enableCode+`"}`))
	replay.AddCookie(cookie)
	rpw := httptest.NewRecorder()
	a.handleMFADisable(rpw, replay)
	if rpw.Code != http.StatusBadRequest {
		t.Fatalf("replayed code accepted: %d", rpw.Code)
	}

	disable := httptest.NewRequest("POST", "/api/auth/mfa/disable", strings.NewReader(`{"password":"pw1234","code":"`+totpCode(raw, time.Now().Add(totpStep*time.Second))+`"}`))
	disable.AddCookie(cookie)
	dw := httptest.NewRecorder()
	a.handleMFADisable(dw, disable)
	if dw.Code != http.StatusOK || a.users["owner"].MFAEnabled {
		t.Fatalf("disable: %d %s", dw.Code, dw.Body.String())
	}
}

func TestFirstUserIsAdmin(t *testing.T) {
	a := newAuthManager("", t.TempDir())
	reg := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"username":"first","password":"pw1234"}`))
	rw := httptest.NewRecorder()
	a.handleRegister(rw, reg)
	if !a.users["first"].Admin {
		t.Fatal("first user should be admin")
	}
	reg2 := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"username":"second","password":"pw1234"}`))
	reg2.AddCookie(rw.Result().Cookies()[0])
	rw2 := httptest.NewRecorder()
	a.handleRegister(rw2, reg2)
	if rw2.Code != http.StatusOK || a.users["second"].Admin {
		t.Fatalf("second user admin=%v code=%d", a.users["second"].Admin, rw2.Code)
	}
}
