package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

	usersFileName    = "auth_users.json"
	sessionsFileName = "auth_sessions.json"
)

var validUsernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,32}$`)

type User struct {
	Username     string    `json:"username"`
	DisplayName  string    `json:"displayName,omitempty"`
	PasswordHash string    `json:"passwordHash"`
	Salt         string    `json:"salt"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type session struct {
	Username string    `json:"username"`
	Expires  time.Time `json:"expires"`
	Created  time.Time `json:"created"`
}

type authManager struct {
	password   string
	storageDir string

	mu        sync.Mutex
	users     map[string]User
	sessions  map[string]session
	apiTokens map[string]apiToken
	limiter   *loginLimiter
}

func newAuthManager(password string, storageDir string) *authManager {
	a := &authManager{
		password:   password,
		storageDir: storageDir,
		users:      make(map[string]User),
		sessions:   make(map[string]session),
		apiTokens:  make(map[string]apiToken),
		limiter:    newLoginLimiter(),
	}
	a.loadPersistentData()
	return a
}

func (a *authManager) enabled() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.password != "" || len(a.users) > 0
}

func (a *authManager) usersCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.users)
}

func hashPassword(password, salt string) string {
	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte(password))
	h.Write([]byte("typstify-auth-salt-v1"))
	return hex.EncodeToString(h.Sum(nil))
}

func generateSalt() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func verifyPassword(password, salt, expectedHash string) bool {
	computed := hashPassword(password, salt)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(expectedHash)) == 1
}

func (a *authManager) loadPersistentData() {
	if a.storageDir == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	usersPath := filepath.Join(a.storageDir, usersFileName)
	if data, err := os.ReadFile(usersPath); err == nil {
		var usersList []User
		if err := json.Unmarshal(data, &usersList); err == nil {
			for _, u := range usersList {
				a.users[strings.ToLower(u.Username)] = u
			}
		}
	}

	sessionsPath := filepath.Join(a.storageDir, sessionsFileName)
	if data, err := os.ReadFile(sessionsPath); err == nil {
		var sessMap map[string]session
		if err := json.Unmarshal(data, &sessMap); err == nil {
			now := time.Now()
			for token, sess := range sessMap {
				if !now.After(sess.Expires) && now.Sub(sess.Created) <= sessionMaxAge {
					a.sessions[token] = sess
				}
			}
		}
	}

	a.loadTokensLocked()
}

func (a *authManager) saveUsersLocked() error {
	if a.storageDir == "" {
		return nil
	}
	if err := os.MkdirAll(a.storageDir, 0700); err != nil {
		return err
	}

	usersList := make([]User, 0, len(a.users))
	for _, u := range a.users {
		usersList = append(usersList, u)
	}

	data, err := json.MarshalIndent(usersList, "", "  ")
	if err != nil {
		return err
	}

	target := filepath.Join(a.storageDir, usersFileName)
	tmp := filepath.Join(a.storageDir, fmt.Sprintf(".%s.%d.tmp", usersFileName, time.Now().UnixNano()))
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}

func (a *authManager) saveSessionsLocked() error {
	if a.storageDir == "" {
		return nil
	}
	if err := os.MkdirAll(a.storageDir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(a.sessions, "", "  ")
	if err != nil {
		return err
	}

	target := filepath.Join(a.storageDir, sessionsFileName)
	tmp := filepath.Join(a.storageDir, fmt.Sprintf(".%s.%d.tmp", sessionsFileName, time.Now().UnixNano()))
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, target)
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
	if c, err := r.Cookie(sessionCookieName); err == nil && a.validSession(c.Value) {
		return true
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		_, ok := a.validToken(strings.TrimPrefix(h, "Bearer "))
		return ok
	}
	return false
}

func (a *authManager) authenticatedUser(r *http.Request) (string, bool) {
	if sess, ok := a.getSession(r); ok && sess.Username != "" {
		return sess.Username, true
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		token := strings.TrimPrefix(h, "Bearer ")
		return a.validToken(token)
	}
	return "", false
}

func (a *authManager) validSession(token string) bool {
	if token == "" {
		return false
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	sess, ok := a.sessions[token]
	if !ok || now.After(sess.Expires) || now.Sub(sess.Created) > sessionMaxAge {
		if ok {
			delete(a.sessions, token)
			_ = a.saveSessionsLocked()
		}
		return false
	}

	sess.Expires = now.Add(sessionTTL) // sliding expiry
	a.sessions[token] = sess
	_ = a.saveSessionsLocked()
	return true
}

func (a *authManager) getSession(r *http.Request) (session, bool) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return session{}, false
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	sess, ok := a.sessions[c.Value]
	if !ok || now.After(sess.Expires) || now.Sub(sess.Created) > sessionMaxAge {
		return session{}, false
	}
	return sess, true
}

// sweepExpired drops expired sessions. Caller holds a.mu.
func (a *authManager) sweepExpired(now time.Time) {
	modified := false
	for token, sess := range a.sessions {
		if now.After(sess.Expires) || now.Sub(sess.Created) > sessionMaxAge {
			delete(a.sessions, token)
			modified = true
		}
	}
	if modified {
		_ = a.saveSessionsLocked()
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
	Username string `json:"username,omitempty"`
	Password string `json:"password"`
}

type registerRequest struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	DisplayName    string `json:"displayName,omitempty"`
	ServerPassword string `json:"serverPassword,omitempty"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (a *authManager) handleRegister(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if ok, wait := a.limiter.allowed(ip); !ok {
		minutes := int(math.Ceil(wait.Minutes()))
		w.Header().Set("Retry-After", fmt.Sprint(int(math.Ceil(wait.Seconds()))))
		writeError(w, http.StatusTooManyRequests,
			fmt.Sprintf("Quá nhiều yêu cầu. Thử lại sau %d phút.", minutes))
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}

	// Registration Policy:
	// 1. If no users exist yet (Bootstrap mode / Server Owner): allow registration.
	// 2. If users exist: require either an authenticated session/token OR valid server password.
	if a.usersCount() > 0 {
		hasAuth := a.validRequest(r)
		hasServerPass := a.password != "" && req.ServerPassword != "" &&
			subtle.ConstantTimeCompare([]byte(req.ServerPassword), []byte(a.password)) == 1
		if !hasAuth && !hasServerPass {
			a.limiter.recordFailure(ip)
			time.Sleep(failedLoginDelay)
			writeError(w, http.StatusForbidden, "Đăng ký bị khóa. Cần đăng nhập tài khoản quản trị hoặc cung cấp mật khẩu máy chủ.")
			return
		}
	}
	username := strings.ToLower(strings.TrimSpace(req.Username))
	if !validUsernamePattern.MatchString(username) {
		writeError(w, http.StatusBadRequest, "Tên tài khoản không hợp lệ (3-32 ký tự, chỉ gồm chữ, số, gạch ngang, chấm)")
		return
	}

	if len(req.Password) < 4 {
		writeError(w, http.StatusBadRequest, "Mật khẩu phải có ít nhất 4 ký tự")
		return
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = username
	}

	a.mu.Lock()
	if _, exists := a.users[username]; exists {
		a.mu.Unlock()
		writeError(w, http.StatusBadRequest, "Tên tài khoản đã tồn tại, vui lòng chọn tên khác")
		return
	}

	salt, err := generateSalt()
	if err != nil {
		a.mu.Unlock()
		writeError(w, http.StatusInternalServerError, "Lỗi tạo mã bảo mật")
		return
	}

	now := time.Now()
	user := User{
		Username:     username,
		DisplayName:  displayName,
		PasswordHash: hashPassword(req.Password, salt),
		Salt:         salt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	a.users[username] = user
	_ = a.saveUsersLocked()

	// Automatically log in the newly registered user.
	token, err := newSessionToken()
	if err != nil {
		a.mu.Unlock()
		writeError(w, http.StatusInternalServerError, "Lỗi tạo phiên đăng nhập")
		return
	}

	a.sweepExpired(now)
	a.sessions[token] = session{
		Username: username,
		Expires:  now.Add(sessionTTL),
		Created:  now,
	}
	_ = a.saveSessionsLocked()
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

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"username":    username,
		"displayName": displayName,
	})
}

func (a *authManager) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !a.enabled() {
		writeJSON(w, http.StatusOK, map[string]any{"authRequired": false, "ok": true})
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

	username := strings.ToLower(strings.TrimSpace(req.Username))
	var loggedInUser string
	var displayName string
	authenticated := false

	a.mu.Lock()
	if username != "" {
		if u, ok := a.users[username]; ok {
			if verifyPassword(req.Password, u.Salt, u.PasswordHash) {
				authenticated = true
				loggedInUser = u.Username
				displayName = u.DisplayName
			}
		}
	} else {
		// Legacy login or single password check
		if a.password != "" && subtle.ConstantTimeCompare([]byte(req.Password), []byte(a.password)) == 1 {
			authenticated = true
			loggedInUser = "admin"
			displayName = "Quản trị viên"
		} else if len(a.users) == 1 {
			// If only one user exists, allow logging in with just password
			for _, u := range a.users {
				if verifyPassword(req.Password, u.Salt, u.PasswordHash) {
					authenticated = true
					loggedInUser = u.Username
					displayName = u.DisplayName
					break
				}
			}
		}
	}
	a.mu.Unlock()

	if !authenticated {
		a.limiter.recordFailure(ip)
		time.Sleep(failedLoginDelay)
		writeError(w, http.StatusUnauthorized, "Tên đăng nhập hoặc mật khẩu không đúng")
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
	a.sessions[token] = session{
		Username: loggedInUser,
		Expires:  now.Add(sessionTTL),
		Created:  now,
	}
	_ = a.saveSessionsLocked()
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

	writeJSON(w, http.StatusOK, map[string]any{
		"authRequired": true,
		"ok":           true,
		"username":     loggedInUser,
		"displayName":  displayName,
	})
}

func (a *authManager) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Chưa đăng nhập")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}

	if len(req.NewPassword) < 4 {
		writeError(w, http.StatusBadRequest, "Mật khẩu mới phải có ít nhất 4 ký tự")
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	username := sess.Username
	if u, exists := a.users[username]; exists {
		if !verifyPassword(req.CurrentPassword, u.Salt, u.PasswordHash) {
			writeError(w, http.StatusBadRequest, "Mật khẩu hiện tại không đúng")
			return
		}

		newSalt, err := generateSalt()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Lỗi tạo mã bảo mật")
			return
		}
		u.Salt = newSalt
		u.PasswordHash = hashPassword(req.NewPassword, newSalt)
		u.UpdatedAt = time.Now()
		a.users[username] = u
		_ = a.saveUsersLocked()
	} else {
		// Legacy admin / server password user
		if subtle.ConstantTimeCompare([]byte(req.CurrentPassword), []byte(a.password)) != 1 {
			writeError(w, http.StatusBadRequest, "Mật khẩu hiện tại không đúng")
			return
		}
		a.password = req.NewPassword
		// Also create a persistent user entry for this admin so it persists on disk
		newSalt, err := generateSalt()
		if err == nil {
			now := time.Now()
			a.users["admin"] = User{
				Username:     "admin",
				DisplayName:  "Quản trị viên",
				PasswordHash: hashPassword(req.NewPassword, newSalt),
				Salt:         newSalt,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			_ = a.saveUsersLocked()
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "Đổi mật khẩu thành công",
	})
}

func (a *authManager) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		a.mu.Lock()
		delete(a.sessions, c.Value)
		_ = a.saveSessionsLocked()
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
	sess, authenticated := a.getSession(r)
	var displayName string
	if authenticated && sess.Username != "" {
		a.mu.Lock()
		if u, ok := a.users[sess.Username]; ok {
			displayName = u.DisplayName
		}
		a.mu.Unlock()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"authRequired":  a.enabled(),
		"authenticated": !a.enabled() || authenticated,
		"username":      sess.Username,
		"displayName":   displayName,
		"hasUsers":      a.usersCount() > 0,
	})
}
