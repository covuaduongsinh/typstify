package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"
)

const (
	sessionCookieName = "typstify_session"
	// sessionTTL is the sliding idle expiry; sessionMaxAge caps a session's
	// total lifetime so a stolen cookie can't be kept alive forever by use.
	sessionTTL    = 30 * 24 * time.Hour
	sessionMaxAge = 90 * 24 * time.Hour
	// failedLoginDelay slows every wrong-password response a little on top
	// of the per-IP lockout in loginLimiter.
	failedLoginDelay = 500 * time.Millisecond
)

type session struct {
	expires time.Time
	created time.Time
}

// authManager gates the server with a single shared password, matching the
// self-hosted single-user deployment model in docs/plans/plan_web_version.md.
// When Password is empty, auth is disabled entirely -- only safe for
// loopback-only/dev use, never when exposed on a network.
type authManager struct {
	password string

	mu       sync.Mutex
	sessions map[string]session
	limiter  *loginLimiter
}

func newAuthManager(password string) *authManager {
	return &authManager{
		password: password,
		sessions: make(map[string]session),
		limiter:  newLoginLimiter(),
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

	now := time.Now()
	sess, ok := a.sessions[token]
	if !ok || now.After(sess.expires) || now.Sub(sess.created) > sessionMaxAge {
		delete(a.sessions, token)
		return false
	}

	sess.expires = now.Add(sessionTTL) // sliding expiry
	a.sessions[token] = sess
	return true
}

// sweepExpired drops expired sessions. Caller holds a.mu.
func (a *authManager) sweepExpired(now time.Time) {
	for token, sess := range a.sessions {
		if now.After(sess.expires) || now.Sub(sess.created) > sessionMaxAge {
			delete(a.sessions, token)
		}
	}
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

	ip := clientIP(r)
	if ok, wait := a.limiter.allowed(ip); !ok {
		minutes := int(math.Ceil(wait.Minutes()))
		w.Header().Set("Retry-After", fmt.Sprint(int(math.Ceil(wait.Seconds()))))
		writeError(w, http.StatusTooManyRequests,
			fmt.Sprintf("Sai mật khẩu quá nhiều lần. Thử lại sau %d phút.", minutes))
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}

	if subtle.ConstantTimeCompare([]byte(req.Password), []byte(a.password)) != 1 {
		a.limiter.recordFailure(ip)
		time.Sleep(failedLoginDelay)
		writeError(w, http.StatusUnauthorized, "Mật khẩu không đúng")
		return
	}
	a.limiter.recordSuccess(ip)

	token, err := newSessionToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	now := time.Now()
	a.mu.Lock()
	a.sweepExpired(now)
	a.sessions[token] = session{expires: now.Add(sessionTTL), created: now}
	a.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS(r),
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
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS(r),
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
