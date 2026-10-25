package server

import (
	"crypto/subtle"
	"slices"
	"strings"
	"sync"
	"time"
)

// Per-account second-factor lockout. It is separate from the per-IP
// loginLimiter: a success on another account (or from another IP) must not
// reset the count of wrong codes guessed for this one.
const (
	mfaMaxFailures = 5
	mfaLockout     = 15 * time.Minute
)

var (
	dummyHashOnce sync.Once
	dummyHash     string
)

// equalizeUnknownUser spends the same argon2id work as a real password
// check, so response time doesn't reveal whether a username exists.
func equalizeUnknownUser(password string) {
	dummyHashOnce.Do(func() { dummyHash, _ = newPasswordHash("typstify-unknown-user") })
	_ = verifyArgon2id(password, dummyHash)
}

// authenticatePassword checks a login's first factor. username may be
// empty for the password-only forms: the server password while no accounts
// exist yet (bootstrap admin), or the single account's password. The
// expensive hash runs without holding a.mu, so a burst of logins cannot
// stall every other request; a legacy hash is upgraded afterwards.
func (a *authManager) authenticatePassword(username, password string) (user, displayName string, ok bool) {
	username = strings.ToLower(strings.TrimSpace(username))

	a.mu.Lock()
	var cand User
	found, bootstrap := false, false
	switch {
	case username != "":
		cand, found = a.users[username]
	case a.password != "" && len(a.users) == 0:
		bootstrap = true
	case len(a.users) == 1:
		for _, u := range a.users {
			cand, found = u, true
		}
	}
	serverPassword := a.password
	a.mu.Unlock()

	if bootstrap {
		if subtle.ConstantTimeCompare([]byte(password), []byte(serverPassword)) == 1 {
			return "admin", "Quản trị viên", true
		}
		return "", "", false
	}
	if !found {
		equalizeUnknownUser(password)
		return "", "", false
	}
	if !verifyPassword(password, cand.Salt, cand.PasswordHash) {
		return "", "", false
	}

	if passwordNeedsRehash(cand.PasswordHash) {
		if h, err := newPasswordHash(password); err == nil {
			a.mu.Lock()
			// Commit only if nobody changed the password meanwhile.
			if u, ok := a.users[cand.Username]; ok && u.PasswordHash == cand.PasswordHash {
				u.PasswordHash = h
				u.Salt = ""
				u.UpdatedAt = time.Now()
				a.users[cand.Username] = u
				_ = a.saveUsersLocked()
			}
			a.mu.Unlock()
		}
	}
	return cand.Username, cand.DisplayName, true
}

// isRecoveryCodeFormat matches the "xxxxx-xxxxx" codes from handleMFAEnable,
// so a wrong 6-digit guess never pays for ten argon2id checks.
func isRecoveryCodeFormat(code string) bool {
	return len(code) == 11 && code[5] == '-'
}

// checkSecondFactor verifies the TOTP or recovery code for an account
// whose password already checked out. Accounts without MFA pass. locked
// reports that the account is in its MFA lockout window.
func (a *authManager) checkSecondFactor(username, code string) (ok, locked bool) {
	code = strings.TrimSpace(code)
	now := time.Now()

	a.mu.Lock()
	u, exists := a.users[username]
	if !exists || !u.MFAEnabled {
		a.mu.Unlock()
		return true, false
	}
	if code == "" {
		a.mu.Unlock()
		return false, false
	}
	if now.Before(u.MFALockedUntil) {
		a.mu.Unlock()
		return false, true
	}
	if step, match := totpMatch(u.MFASecret, code, now); match && step > u.MFALastStep {
		u.MFALastStep = step
		u.MFAFailures = 0
		a.users[username] = u
		_ = a.saveUsersLocked()
		a.mu.Unlock()
		return true, false
	}
	var hashes []string
	if isRecoveryCodeFormat(code) {
		hashes = slices.Clone(u.RecoveryCodes)
	}
	a.mu.Unlock()

	matched := ""
	for _, h := range hashes {
		if verifyArgon2id(code, h) {
			matched = h
			break
		}
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	u = a.users[username]
	if matched != "" {
		if i := slices.Index(u.RecoveryCodes, matched); i >= 0 {
			u.RecoveryCodes = slices.Delete(slices.Clone(u.RecoveryCodes), i, i+1)
			u.MFAFailures = 0
			a.users[username] = u
			_ = a.saveUsersLocked()
			return true, false
		}
	}
	u.MFAFailures++
	if u.MFAFailures >= mfaMaxFailures {
		u.MFALockedUntil = now.Add(mfaLockout)
		u.MFAFailures = 0
		locked = true
	}
	a.users[username] = u
	_ = a.saveUsersLocked()
	return false, locked
}

// revokeOtherSessionsLocked ends every session of username except keep,
// after a security change (MFA turned on/off) so a stolen cookie stops
// working. Caller holds a.mu.
func (a *authManager) revokeOtherSessionsLocked(username, keep string) {
	changed := false
	for tok, s := range a.sessions {
		if s.Username == username && tok != keep {
			delete(a.sessions, tok)
			changed = true
		}
	}
	if changed {
		_ = a.saveSessionsLocked()
	}
}
