package server

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLoginLimiterLocksOutAfterRepeatedFailures(t *testing.T) {
	l := newLoginLimiter()
	now := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	l.now = func() time.Time { return now }

	for i := 0; i < maxLoginFailures-1; i++ {
		l.recordFailure("1.2.3.4")
	}
	if ok, _ := l.allowed("1.2.3.4"); !ok {
		t.Fatalf("locked out after %d failures, want %d", maxLoginFailures-1, maxLoginFailures)
	}

	l.recordFailure("1.2.3.4")
	if ok, wait := l.allowed("1.2.3.4"); ok || wait <= 0 {
		t.Fatalf("allowed=%v wait=%v after %d failures, want locked", ok, wait, maxLoginFailures)
	}
	if ok, _ := l.allowed("5.6.7.8"); !ok {
		t.Fatal("lockout leaked to another IP")
	}

	now = now.Add(lockoutDuration + time.Second)
	if ok, _ := l.allowed("1.2.3.4"); !ok {
		t.Fatal("still locked after lockoutDuration")
	}
}

func TestLoginLimiterSuccessResets(t *testing.T) {
	l := newLoginLimiter()
	for i := 0; i < maxLoginFailures-1; i++ {
		l.recordFailure("ip")
	}
	l.recordSuccess("ip")
	l.recordFailure("ip")
	if ok, _ := l.allowed("ip"); !ok {
		t.Fatal("failures before a successful login still counted")
	}
}

func TestClientIPTrustsForwardedOnlyFromPrivatePeer(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.2")

	r.RemoteAddr = "10.0.0.2:5555" // proxy on the private Docker network
	if got := clientIP(r); got != "203.0.113.9" {
		t.Errorf("behind private proxy: clientIP = %q, want 203.0.113.9", got)
	}

	r.RemoteAddr = "198.51.100.7:5555" // public client spoofing the header
	if got := clientIP(r); got != "198.51.100.7" {
		t.Errorf("public peer: clientIP = %q, want the peer address", got)
	}
}

func TestIsHTTPS(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "172.18.0.3:1234"
	r.Header.Set("X-Forwarded-Proto", "https")
	if !isHTTPS(r) {
		t.Error("X-Forwarded-Proto=https from private proxy should count as HTTPS")
	}

	r.RemoteAddr = "198.51.100.7:1234"
	if isHTTPS(r) {
		t.Error("X-Forwarded-Proto from a public peer must be ignored")
	}

	r.TLS = &tls.ConnectionState{}
	if !isHTTPS(r) {
		t.Error("direct TLS should count as HTTPS")
	}
}

func TestSameOrigin(t *testing.T) {
	r := httptest.NewRequest("GET", "http://typst.example.com/", nil)
	r.RemoteAddr = "198.51.100.7:1"
	if !sameOrigin(r) {
		t.Error("no Origin header should pass")
	}
	r.Header.Set("Origin", "https://typst.example.com")
	if !sameOrigin(r) {
		t.Error("matching Origin rejected")
	}
	r.Header.Set("Origin", "https://evil.example.net")
	if sameOrigin(r) {
		t.Error("foreign Origin accepted")
	}
}

func TestPreviewRootWebSocketRequiresAuth(t *testing.T) {
	s := &Server{auth: newAuthManager("secret")}

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Connection", "Upgrade")
	r.Header.Set("Upgrade", "websocket")
	w := httptest.NewRecorder()

	if handled := s.handlePreviewRootWebSocket(w, r); !handled {
		t.Fatal("websocket upgrade at / not handled")
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated preview websocket: status %d, want 401", w.Code)
	}
}

func TestPreviewRootWebSocketRejectsCrossOrigin(t *testing.T) {
	s := &Server{auth: newAuthManager("secret")}
	token, _ := newSessionToken()
	s.auth.sessions[token] = session{expires: time.Now().Add(time.Hour), created: time.Now()}

	r := httptest.NewRequest("GET", "http://typst.example.com/", nil)
	r.Header.Set("Connection", "Upgrade")
	r.Header.Set("Upgrade", "websocket")
	r.Header.Set("Origin", "https://evil.example.net")
	r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	w := httptest.NewRecorder()

	s.handlePreviewRootWebSocket(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-origin preview websocket: status %d, want 403", w.Code)
	}
}

func TestLoginBodyTooLarge(t *testing.T) {
	a := newAuthManager("secret")
	h := withBodyLimit(http.HandlerFunc(a.handleLogin))

	body := `{"password":"` + strings.Repeat("x", defaultBodyLimit) + `"}`
	r := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized login body: status %d, want 413", w.Code)
	}
}

func TestLoginSetsSecureCookieBehindHTTPSProxy(t *testing.T) {
	a := newAuthManager("secret")
	r := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"password":"secret"}`))
	r.RemoteAddr = "172.18.0.3:1234"
	r.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	a.handleLogin(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("login status %d", w.Code)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatalf("session cookie = %+v, want Secure+HttpOnly", cookies)
	}
}

func TestSessionMaxAge(t *testing.T) {
	a := newAuthManager("secret")
	a.sessions["old"] = session{
		expires: time.Now().Add(time.Hour),
		created: time.Now().Add(-sessionMaxAge - time.Hour),
	}
	if a.validSession("old") {
		t.Fatal("session older than sessionMaxAge still valid")
	}
}
