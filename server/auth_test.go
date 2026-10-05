package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthRegisterAndLoginFlow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "typstify-auth-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	auth := newAuthManager("", tempDir)

	// 1. Register a new user
	regBody := `{"username":"duongsinh","password":"mypassword123","displayName":"Duong Sinh"}`
	req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(regBody))
	w := httptest.NewRecorder()
	auth.handleRegister(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("register failed with status %d: %s", w.Code, w.Body.String())
	}

	var regResp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&regResp); err != nil {
		t.Fatal(err)
	}
	if regResp["username"] != "duongsinh" {
		t.Fatalf("expected username duongsinh, got %v", regResp["username"])
	}

	cookies := w.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != sessionCookieName {
		t.Fatal("session cookie not set after registration")
	}
	sessionToken := cookies[0].Value

	// 2. Unauthenticated registration when users exist should be rejected with 403
	reqUnauth := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"username":"attacker","password":"password123"}`))
	wUnauth := httptest.NewRecorder()
	auth.handleRegister(wUnauth, reqUnauth)
	if wUnauth.Code != http.StatusForbidden {
		t.Fatalf("unauthenticated register status %d, want 403 Forbidden", wUnauth.Code)
	}

	// 3. Duplicate registration by authenticated user should be rejected with 400
	reqDup := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(regBody))
	reqDup.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionToken})
	wDup := httptest.NewRecorder()
	auth.handleRegister(wDup, reqDup)
	if wDup.Code != http.StatusBadRequest {
		t.Fatalf("duplicate register status %d, want 400", wDup.Code)
	}

	// 4. Authenticated registration for a new user should succeed
	reqNewUser := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"username":"user2","password":"password123"}`))
	reqNewUser.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionToken})
	wNewUser := httptest.NewRecorder()
	auth.handleRegister(wNewUser, reqNewUser)
	if wNewUser.Code != http.StatusOK {
		t.Fatalf("authenticated register status %d, want 200: %s", wNewUser.Code, wNewUser.Body.String())
	}

	// 5. Registration using server password should succeed
	authWithServerPass := newAuthManager("supersecret", tempDir)
	// Already has users in tempDir
	reqWithPass := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"username":"user3","password":"password123","serverPassword":"supersecret"}`))
	wWithPass := httptest.NewRecorder()
	authWithServerPass.handleRegister(wWithPass, reqWithPass)
	if wWithPass.Code != http.StatusOK {
		t.Fatalf("register with server password status %d, want 200: %s", wWithPass.Code, wWithPass.Body.String())
	}
	// 3. Login with wrong password
	wrongLogin := `{"username":"duongsinh","password":"wrongpassword"}`
	reqWrong := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(wrongLogin))
	wWrong := httptest.NewRecorder()
	auth.handleLogin(wWrong, reqWrong)
	if wWrong.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password login status %d, want 401", wWrong.Code)
	}

	// 4. Login with correct credentials
	correctLogin := `{"username":"duongsinh","password":"mypassword123"}`
	reqLogin := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(correctLogin))
	wLogin := httptest.NewRecorder()
	auth.handleLogin(wLogin, reqLogin)
	if wLogin.Code != http.StatusOK {
		t.Fatalf("login failed: %s", wLogin.Body.String())
	}

	// 5. Change password
	changeBody := `{"currentPassword":"mypassword123","newPassword":"newsecretpassword456"}`
	reqChange := httptest.NewRequest("POST", "/api/auth/change-password", strings.NewReader(changeBody))
	reqChange.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionToken})
	wChange := httptest.NewRecorder()
	auth.handleChangePassword(wChange, reqChange)
	if wChange.Code != http.StatusOK {
		t.Fatalf("change password failed: %s", wChange.Body.String())
	}

	// 6. Old password should now fail
	reqOld := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(correctLogin))
	wOld := httptest.NewRecorder()
	auth.handleLogin(wOld, reqOld)
	if wOld.Code != http.StatusUnauthorized {
		t.Fatalf("login with old password should fail, got %d", wOld.Code)
	}

	// 7. New password should succeed
	newLogin := `{"username":"duongsinh","password":"newsecretpassword456"}`
	reqNew := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(newLogin))
	wNew := httptest.NewRecorder()
	auth.handleLogin(wNew, reqNew)
	if wNew.Code != http.StatusOK {
		t.Fatalf("login with new password failed: %s", wNew.Body.String())
	}
}

func TestSessionPersistenceAcrossServerRestarts(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "typstify-session-persist-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Start server 1
	auth1 := newAuthManager("", tempDir)
	regBody := `{"username":"player1","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(regBody))
	w := httptest.NewRecorder()
	auth1.handleRegister(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("register failed: %s", w.Body.String())
	}

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no cookie")
	}
	token := cookies[0].Value

	// Verify users and sessions files were written to disk
	usersFile := filepath.Join(tempDir, usersFileName)
	if _, err := os.Stat(usersFile); err != nil {
		t.Fatalf("users file not created: %v", err)
	}
	sessionsFile := filepath.Join(tempDir, sessionsFileName)
	if _, err := os.Stat(sessionsFile); err != nil {
		t.Fatalf("sessions file not created: %v", err)
	}

	// Simulate server reboot by creating a fresh authManager instance reading from the same storageDir
	auth2 := newAuthManager("", tempDir)

	// Status check with existing cookie should still be authenticated!
	reqStatus := httptest.NewRequest("GET", "/api/auth/status", nil)
	reqStatus.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	wStatus := httptest.NewRecorder()
	auth2.handleStatus(wStatus, reqStatus)

	var statusResp map[string]any
	if err := json.NewDecoder(wStatus.Body).Decode(&statusResp); err != nil {
		t.Fatal(err)
	}
	if statusResp["authenticated"] != true {
		t.Fatalf("session should persist across restart, got %v", statusResp)
	}
	if statusResp["username"] != "player1" {
		t.Fatalf("expected username player1, got %v", statusResp["username"])
	}
}
