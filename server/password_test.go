package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewPasswordHashIsArgon2idAndVerifies(t *testing.T) {
	h, err := newPasswordHash("s3cret-pass")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$") {
		t.Fatalf("hash %q is not a PHC argon2id string", h)
	}
	if !verifyPassword("s3cret-pass", "", h) {
		t.Fatal("correct password rejected")
	}
	if verifyPassword("wrong", "", h) {
		t.Fatal("wrong password accepted")
	}
	h2, _ := newPasswordHash("s3cret-pass")
	if h == h2 {
		t.Fatal("two hashes of the same password must differ (random salt)")
	}
	if passwordNeedsRehash(h) {
		t.Fatal("fresh argon2id hash should not need rehash")
	}
}

func TestLegacySHA256HashStillVerifiesAndNeedsRehash(t *testing.T) {
	salt, _ := generateSalt()
	legacy := hashPassword("oldpass", salt)
	if !verifyPassword("oldpass", salt, legacy) {
		t.Fatal("legacy hash no longer verifies")
	}
	if verifyPassword("nope", salt, legacy) {
		t.Fatal("legacy hash accepted wrong password")
	}
	if !passwordNeedsRehash(legacy) {
		t.Fatal("legacy hash should need rehash")
	}
}

func TestMalformedArgon2HashRejected(t *testing.T) {
	for _, h := range []string{"$argon2id$", "$argon2id$v=19$m=x$a$b", "$argon2id$v=19$m=65536,t=1,p=1$!!$!!"} {
		if verifyPassword("x", "", h) {
			t.Fatalf("malformed hash %q accepted", h)
		}
	}
}

func TestLoginUpgradesLegacyHash(t *testing.T) {
	a := newAuthManager("", t.TempDir())
	salt, _ := generateSalt()
	a.users["legacy"] = User{Username: "legacy", PasswordHash: hashPassword("oldpass", salt), Salt: salt}

	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"legacy","password":"oldpass"}`))
	w := httptest.NewRecorder()
	a.handleLogin(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login status %d: %s", w.Code, w.Body.String())
	}

	u := a.users["legacy"]
	if !strings.HasPrefix(u.PasswordHash, "$argon2id$") {
		t.Fatalf("hash not upgraded: %q", u.PasswordHash)
	}
	if u.Salt != "" {
		t.Fatal("legacy salt should be cleared after upgrade")
	}

	// Reload from disk: upgraded hash must have been persisted.
	b := newAuthManager("", a.storageDir)
	if !verifyPassword("oldpass", b.users["legacy"].Salt, b.users["legacy"].PasswordHash) {
		t.Fatal("upgraded hash not persisted")
	}
}
