package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TOTP per RFC 6238 with the defaults every authenticator app supports:
// HMAC-SHA1, 30-second steps, 6 digits. One step of clock skew is accepted
// either way, and a step that already logged in cannot be replayed.
const (
	totpStep             = 30
	totpDigits           = 6
	totpSkewSteps        = 1
	totpSecretBytes      = 20
	mfaRecoveryCodeCount = 10
	mfaIssuer            = "Typstify"
)

var totpB32 = base32.StdEncoding.WithPadding(base32.NoPadding)

func totpCodeAtStep(secret []byte, step int64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(step))
	mac := hmac.New(sha1.New, secret)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	bin := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", totpDigits, bin%1_000_000)
}

func totpCode(secret []byte, t time.Time) string {
	return totpCodeAtStep(secret, t.Unix()/totpStep)
}

// totpMatch returns the time step code was generated for, within the
// allowed skew of now.
func totpMatch(b32Secret, code string, now time.Time) (int64, bool) {
	code = strings.TrimSpace(code)
	if len(code) != totpDigits {
		return 0, false
	}
	secret, err := totpB32.DecodeString(strings.ToUpper(b32Secret))
	if err != nil || len(secret) == 0 {
		return 0, false
	}
	cur := now.Unix() / totpStep
	for d := int64(-totpSkewSteps); d <= totpSkewSteps; d++ {
		if hmac.Equal([]byte(totpCodeAtStep(secret, cur+d)), []byte(code)) {
			return cur + d, true
		}
	}
	return 0, false
}

func totpVerify(b32Secret, code string, now time.Time) bool {
	_, ok := totpMatch(b32Secret, code, now)
	return ok
}

func writeMFARequired(w http.ResponseWriter, codeGiven, locked bool) {
	msg := "Cần mã xác thực hai bước"
	switch {
	case locked:
		msg = fmt.Sprintf("Nhập sai mã xác thực quá nhiều lần. Thử lại sau %d phút.", int(mfaLockout.Minutes()))
	case codeGiven:
		msg = "Mã xác thực hai bước không đúng"
	}
	writeJSON(w, http.StatusUnauthorized, map[string]any{"error": msg, "mfaRequired": true})
}

func sessionToken(r *http.Request) string {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		return c.Value
	}
	return ""
}

// checkAccountPassword verifies username's own password outside a.mu.
func (a *authManager) checkAccountPassword(username, password string) bool {
	a.mu.Lock()
	u, ok := a.users[username]
	a.mu.Unlock()
	return ok && verifyPassword(password, u.Salt, u.PasswordHash)
}

// sessionUser returns the username of the logged-in account behind r, or
// writes 401 and returns false. Only real accounts can manage MFA.
func (a *authManager) sessionUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	sess, ok := a.getSession(r)
	if ok {
		a.mu.Lock()
		_, exists := a.users[sess.Username]
		a.mu.Unlock()
		if exists {
			return sess.Username, true
		}
	}
	writeError(w, http.StatusUnauthorized, "Chưa đăng nhập bằng tài khoản người dùng")
	return "", false
}

// handleMFASetup creates a pending secret and returns it for the user to
// add to an authenticator app. MFA is only switched on by handleMFAEnable.
func (a *authManager) handleMFASetup(w http.ResponseWriter, r *http.Request) {
	username, ok := a.sessionUser(w, r)
	if !ok {
		return
	}
	raw := make([]byte, totpSecretBytes)
	if _, err := rand.Read(raw); err != nil {
		writeError(w, http.StatusInternalServerError, "Lỗi tạo mã bảo mật")
		return
	}
	secret := totpB32.EncodeToString(raw)

	a.mu.Lock()
	u := a.users[username]
	if u.MFAEnabled {
		a.mu.Unlock()
		writeError(w, http.StatusConflict, "Xác thực hai bước đã được bật")
		return
	}
	u.MFAPendingSecret = secret
	a.users[username] = u
	_ = a.saveUsersLocked()
	a.mu.Unlock()

	label := url.PathEscape(mfaIssuer + ":" + username)
	uri := fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		label, secret, url.QueryEscape(mfaIssuer), totpDigits, totpStep)
	writeJSON(w, http.StatusOK, map[string]string{"secret": secret, "otpauthUri": uri})
}

type mfaCodeRequest struct {
	Code     string `json:"code"`
	Password string `json:"password,omitempty"`
}

// handleMFAEnable confirms the pending secret with a code from the app and
// the account password (so a stolen session alone can't bind the
// attacker's authenticator), turns MFA on, ends the account's other
// sessions and returns one-time recovery codes (shown only once).
func (a *authManager) handleMFAEnable(w http.ResponseWriter, r *http.Request) {
	username, ok := a.sessionUser(w, r)
	if !ok {
		return
	}
	ip := clientIP(r)
	if ok, _ := a.limiter.allowed(ip); !ok {
		writeError(w, http.StatusTooManyRequests, "Quá nhiều lần thử. Vui lòng thử lại sau.")
		return
	}
	var req mfaCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}
	if !a.checkAccountPassword(username, req.Password) {
		a.limiter.recordFailure(ip)
		time.Sleep(failedLoginDelay)
		writeError(w, http.StatusBadRequest, "Mật khẩu không đúng")
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	u := a.users[username]
	if u.MFAPendingSecret == "" {
		writeError(w, http.StatusBadRequest, "Chưa khởi tạo xác thực hai bước")
		return
	}
	step, ok := totpMatch(u.MFAPendingSecret, req.Code, time.Now())
	if !ok {
		writeError(w, http.StatusBadRequest, "Mã xác thực không đúng")
		return
	}

	codes := make([]string, 0, mfaRecoveryCodeCount)
	hashes := make([]string, 0, mfaRecoveryCodeCount)
	for range mfaRecoveryCodeCount {
		buf := make([]byte, 5)
		if _, err := rand.Read(buf); err != nil {
			writeError(w, http.StatusInternalServerError, "Lỗi tạo mã khôi phục")
			return
		}
		c := hex.EncodeToString(buf)
		c = c[:5] + "-" + c[5:]
		h, err := newPasswordHash(c)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Lỗi tạo mã khôi phục")
			return
		}
		codes = append(codes, c)
		hashes = append(hashes, h)
	}

	u.MFASecret = u.MFAPendingSecret
	u.MFAPendingSecret = ""
	u.MFAEnabled = true
	u.MFALastStep = step
	u.RecoveryCodes = hashes
	u.MFAFailures = 0
	u.UpdatedAt = time.Now()
	a.users[username] = u
	_ = a.saveUsersLocked()
	a.revokeOtherSessionsLocked(username, sessionToken(r))

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "recoveryCodes": codes})
}

// handleMFADisable turns MFA off; it needs both the password and a second
// factor so a hijacked session alone cannot remove it.
func (a *authManager) handleMFADisable(w http.ResponseWriter, r *http.Request) {
	username, ok := a.sessionUser(w, r)
	if !ok {
		return
	}
	ip := clientIP(r)
	if ok, _ := a.limiter.allowed(ip); !ok {
		writeError(w, http.StatusTooManyRequests, "Quá nhiều lần thử. Vui lòng thử lại sau.")
		return
	}
	var req mfaCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}

	a.mu.Lock()
	enabled := a.users[username].MFAEnabled
	a.mu.Unlock()
	if !enabled {
		writeError(w, http.StatusBadRequest, "Xác thực hai bước chưa được bật")
		return
	}
	// Both checks run outside a.mu; the second-factor check also counts
	// toward the account's MFA lockout.
	if !a.checkAccountPassword(username, req.Password) {
		a.limiter.recordFailure(ip)
		time.Sleep(failedLoginDelay)
		writeError(w, http.StatusBadRequest, "Mật khẩu hoặc mã xác thực không đúng")
		return
	}
	if ok, locked := a.checkSecondFactor(username, req.Code); !ok {
		a.limiter.recordFailure(ip)
		time.Sleep(failedLoginDelay)
		msg := "Mật khẩu hoặc mã xác thực không đúng"
		if locked {
			msg = "Nhập sai mã xác thực quá nhiều lần. Thử lại sau."
		}
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	u := a.users[username]
	u.MFAEnabled = false
	u.MFASecret = ""
	u.MFALastStep = 0
	u.RecoveryCodes = nil
	u.MFAFailures = 0
	u.MFALockedUntil = time.Time{}
	u.UpdatedAt = time.Now()
	a.users[username] = u
	_ = a.saveUsersLocked()
	a.revokeOtherSessionsLocked(username, sessionToken(r))

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
