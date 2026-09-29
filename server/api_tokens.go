package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const apiTokensFileName = "auth_tokens.json"

// apiToken is a long-lived, non-expiring credential for machine-to-machine
// callers (e.g. the desktop app syncing with this self-hosted server) that
// cannot drive the browser cookie/login flow. Only its hash is ever
// persisted -- the plaintext is returned once, at issuance, exactly like a
// GitHub personal access token.
type apiToken struct {
	TokenHash  string    `json:"tokenHash"`
	Username   string    `json:"username"`
	Label      string    `json:"label,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	LastUsedAt time.Time `json:"lastUsedAt,omitempty"`
}

// apiTokenInfo is the safe, listable view of an apiToken -- the hash is
// included because it is one-way (can't be turned back into the plaintext
// token) and doubles as this token's revoke-by id.
type apiTokenInfo struct {
	Hash       string    `json:"hash"`
	Label      string    `json:"label,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	LastUsedAt time.Time `json:"lastUsedAt,omitempty"`
}

func hashToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

func (a *authManager) loadTokensLocked() {
	if a.storageDir == "" {
		return
	}
	path := filepath.Join(a.storageDir, apiTokensFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var tokens map[string]apiToken
	if err := json.Unmarshal(data, &tokens); err == nil {
		a.apiTokens = tokens
	}
}

func (a *authManager) saveTokensLocked() error {
	if a.storageDir == "" {
		return nil
	}
	if err := os.MkdirAll(a.storageDir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(a.apiTokens, "", "  ")
	if err != nil {
		return err
	}

	target := filepath.Join(a.storageDir, apiTokensFileName)
	tmp := filepath.Join(a.storageDir, fmt.Sprintf(".%s.%d.tmp", apiTokensFileName, time.Now().UnixNano()))
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}

// issueToken mints a new API token for username and persists its hash.
// The plaintext is returned once and never stored.
func (a *authManager) issueToken(username, label string) (string, error) {
	plaintext, err := newSessionToken() // 32 random bytes, hex-encoded
	if err != nil {
		return "", err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.apiTokens == nil {
		a.apiTokens = make(map[string]apiToken)
	}
	a.apiTokens[hashToken(plaintext)] = apiToken{
		TokenHash: hashToken(plaintext),
		Username:  username,
		Label:     label,
		CreatedAt: time.Now(),
	}
	if err := a.saveTokensLocked(); err != nil {
		return "", err
	}
	return plaintext, nil
}

// validToken reports whether plaintext matches a live, unrevoked API token,
// and updates its last-used timestamp when it does.
func (a *authManager) validToken(plaintext string) (username string, ok bool) {
	if plaintext == "" {
		return "", false
	}
	hash := hashToken(plaintext)

	a.mu.Lock()
	defer a.mu.Unlock()

	tok, exists := a.apiTokens[hash]
	if !exists {
		return "", false
	}

	tok.LastUsedAt = time.Now()
	a.apiTokens[hash] = tok
	_ = a.saveTokensLocked()
	return tok.Username, true
}

func (a *authManager) revokeToken(hash string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.apiTokens[hash]; !exists {
		return fmt.Errorf("token not found")
	}
	delete(a.apiTokens, hash)
	return a.saveTokensLocked()
}

func (a *authManager) listTokens(username string) []apiTokenInfo {
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make([]apiTokenInfo, 0, len(a.apiTokens))
	for hash, tok := range a.apiTokens {
		if username != "" && tok.Username != username {
			continue
		}
		out = append(out, apiTokenInfo{
			Hash:       hash,
			Label:      tok.Label,
			CreatedAt:  tok.CreatedAt,
			LastUsedAt: tok.LastUsedAt,
		})
	}
	return out
}

type issueTokenRequest struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password"`
	Label    string `json:"label,omitempty"`
}

// handleIssueToken authenticates with the same credentials as handleLogin
// (username+password, or password-only for the legacy/single-user case) and
// returns a long-lived Bearer token instead of setting a cookie -- meant for
// non-browser callers such as the desktop app connecting to this
// self-hosted server over the network.
func (a *authManager) handleIssueToken(w http.ResponseWriter, r *http.Request) {
	if !a.enabled() {
		writeError(w, http.StatusBadRequest, "authentication is not enabled on this server")
		return
	}

	ip := clientIP(r)
	if ok, wait := a.limiter.allowed(ip); !ok {
		writeError(w, http.StatusTooManyRequests, fmt.Sprintf("too many attempts, retry in %s", wait.Round(time.Second)))
		return
	}

	var req issueTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}

	username := strings.ToLower(strings.TrimSpace(req.Username))
	var loggedInUser string
	authenticated := false

	a.mu.Lock()
	if username != "" {
		if u, ok := a.users[username]; ok && verifyPassword(req.Password, u.Salt, u.PasswordHash) {
			authenticated = true
			loggedInUser = u.Username
		}
	} else {
		if a.password != "" && subtle.ConstantTimeCompare([]byte(req.Password), []byte(a.password)) == 1 {
			authenticated = true
			loggedInUser = "admin"
		} else if len(a.users) == 1 {
			for _, u := range a.users {
				if verifyPassword(req.Password, u.Salt, u.PasswordHash) {
					authenticated = true
					loggedInUser = u.Username
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

	token, err := a.issueToken(loggedInUser, req.Label)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"token": token})
}

func (a *authManager) handleListTokens(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.getSession(r)
	username := ""
	if ok {
		username = sess.Username
	}
	writeJSON(w, http.StatusOK, a.listTokens(username))
}

func (a *authManager) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	if hash == "" {
		writeError(w, http.StatusBadRequest, "missing token hash")
		return
	}
	if err := a.revokeToken(hash); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
