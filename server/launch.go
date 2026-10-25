package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"
)

// desktopUser is the account the desktop app (apps/tauri) signs in as when
// the machine has no account yet.
const desktopUser = "desktop"

// launchTokenTTL bounds how long after start the desktop window may use
// the launch token (once).
const launchTokenTTL = 5 * time.Minute

// prepareLaunch turns on desktop mode (Options.LaunchToken). It runs in
// New, before the server listens, so the account the window signs in as is
// settled before anything else can reach the port: on a machine without
// accounts nobody can register first and become the admin the window then
// uses.
func (a *authManager) prepareLaunch(token string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	user, err := a.localUserLocked(now)
	if err != nil {
		return err
	}
	a.desktopMode = true
	a.launchToken = token
	a.launchUser = user
	a.launchExpires = now.Add(launchTokenTTL)
	return nil
}

// handleLaunch signs the desktop app's window in. The app starts this
// server as a sidecar with a fresh random TYPSTIFY_LAUNCH_TOKEN and opens
// /api/auth/launch?token=…; the token never leaves the two processes and
// works once, within launchTokenTTL.
//
// A wrong token only costs a delay: failures are not counted against the
// shared per-IP limit, or any page reaching 127.0.0.1 could lock the real
// window out (a 256-bit token cannot be guessed anyway).
func (a *authManager) handleLaunch(w http.ResponseWriter, r *http.Request) {
	got := r.URL.Query().Get("token")
	a.mu.Lock()
	valid := a.launchToken != "" && time.Now().Before(a.launchExpires) &&
		subtle.ConstantTimeCompare([]byte(got), []byte(a.launchToken)) == 1
	user := a.launchUser
	if valid {
		a.launchToken = "" // once: a token found later (history, logs) is useless
	}
	a.mu.Unlock()
	if !valid {
		time.Sleep(failedLoginDelay)
		writeError(w, http.StatusForbidden, "Mã khởi động không hợp lệ hoặc đã dùng. Hãy mở lại ứng dụng.")
		return
	}

	token, err := newSessionToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Lỗi tạo phiên đăng nhập")
		return
	}
	now := time.Now()
	a.mu.Lock()
	a.sweepExpired(now)
	a.sessions[token] = session{Username: user, Expires: now.Add(sessionTTL), Created: now}
	_ = a.saveSessionsLocked()
	a.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   isHTTPS(r),
		MaxAge:   int(sessionTTL.Seconds()),
	})
	// The token stays out of the page's address and history.
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// localUserLocked returns the account the desktop window uses: "desktop",
// or the first admin of an existing store, creating "desktop" (an admin:
// this is the machine owner's app) when there is neither. a.mu must be
// held.
func (a *authManager) localUserLocked(now time.Time) (string, error) {
	if _, ok := a.users[desktopUser]; ok {
		return desktopUser, nil
	}
	admins := make([]string, 0, len(a.users))
	for name, u := range a.users {
		if u.Admin {
			admins = append(admins, name)
		}
	}
	if len(admins) > 0 {
		sort.Strings(admins)
		return admins[0], nil
	}
	// A random password nobody knows: the account is used through launch
	// sessions (it can be changed in Settings to sign in from a browser).
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	hash, err := newPasswordHash(hex.EncodeToString(raw))
	if err != nil {
		return "", err
	}
	a.users[desktopUser] = User{
		Username:     desktopUser,
		DisplayName:  "Máy này",
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
		Admin:        true,
	}
	return desktopUser, a.saveUsersLocked()
}

// desktopGuard protects the desktop sidecar, which listens on 127.0.0.1
// where any web page in the user's browser can send requests: the Host
// must be a loopback name (no DNS rebinding) and a cross-site Origin is
// refused on state-changing requests (no CSRF; the session cookie is also
// SameSite=Strict).
func desktopGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		switch strings.Trim(host, "[]") {
		case "127.0.0.1", "localhost", "::1":
		default:
			writeError(w, http.StatusForbidden, "host not allowed")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host {
				writeError(w, http.StatusForbidden, "origin not allowed")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
