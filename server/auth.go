package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

const (
	sessionCookieName = "typstify_session"
	sessionTTL        = 30 * 24 * time.Hour
)

// authManager gates the server with a single shared password, matching the
// self-hosted single-user deployment model in docs/plans/plan_web_version.md.
// When Password is empty, auth is disabled entirely -- only safe for
// loopback-only/dev use, never when exposed on a network.
type authManager struct {
	password string

	mu       sync.Mutex
	sessions map[string]time.Time
}

func newAuthManager(password string) *authManager {
	return &authManager{
		password: password,
		sessions: make(map[string]time.Time),
	}
}

func (a *authManager) enabled() bool {
	return a.password != ""
}

// require wraps an http.HandlerFunc so it only runs for authenticated
// requests (or always, if auth is disabled).
func (a *authManager) require(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.enabled() && !a.validRequest(r) {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next(w, r)
	}
}

func (a *authManager) validRequest(r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	return a.validSession(c.Value)
}

func (a *authManager) validSession(token string) bool {
	if token == "" {
		return false
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	exp, ok := a.sessions[token]
	if !ok || time.Now().After(exp) {
		delete(a.sessions, token)
		return false
	}

	a.sessions[token] = time.Now().Add(sessionTTL) // sliding expiry
	return true
}

func newSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

type loginRequest struct {
	Password string `json:"password"`
}

func (a *authManager) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !a.enabled() {
		writeJSON(w, http.StatusOK, map[string]bool{"authRequired": false, "ok": true})
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if subtle.ConstantTimeCompare([]byte(req.Password), []byte(a.password)) != 1 {
		writeError(w, http.StatusUnauthorized, "incorrect password")
		return
	}

	token, err := newSessionToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	a.mu.Lock()
	a.sessions[token] = time.Now().Add(sessionTTL)
	a.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   int(sessionTTL.Seconds()),
	})

	writeJSON(w, http.StatusOK, map[string]bool{"authRequired": true, "ok": true})
}

func (a *authManager) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		a.mu.Lock()
		delete(a.sessions, c.Value)
		a.mu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *authManager) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{
		"authRequired":  a.enabled(),
		"authenticated": !a.enabled() || a.validRequest(r),
	})
}
