package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"looz.ws/typstify/service"
)

func TestDropboxAPI_StatusAndDisconnect(t *testing.T) {
	appSrv := service.NewService(context.Background())
	s := New(appSrv, Options{})

	// 1. Get initial status (unconnected)
	req := httptest.NewRequest("GET", "/api/dropbox/status", nil)
	w := httptest.NewRecorder()
	s.handleDropboxStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("handleDropboxStatus returned %d, want 200", w.Code)
	}

	var statusResp dropboxStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&statusResp); err != nil {
		t.Fatalf("decode status failed: %v", err)
	}
	if statusResp.Connected {
		t.Errorf("expected disconnected initially")
	}

	// 2. Test Auth URL generation
	urlReqBody := `{"appKey": "my_app_key", "redirectUri": "http://localhost:5173/auth/callback"}`
	req = httptest.NewRequest("POST", "/api/dropbox/auth/url", strings.NewReader(urlReqBody))
	w = httptest.NewRecorder()
	s.handleDropboxAuthURL(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("handleDropboxAuthURL returned %d", w.Code)
	}
	var urlResp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&urlResp)
	if !strings.Contains(urlResp["url"], "client_id=my_app_key") {
		t.Errorf("url does not contain client_id: %s", urlResp["url"])
	}

	// 3. Test Disconnect
	req = httptest.NewRequest("POST", "/api/dropbox/auth/disconnect", nil)
	w = httptest.NewRecorder()
	s.handleDropboxDisconnect(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("handleDropboxDisconnect returned %d", w.Code)
	}
}
