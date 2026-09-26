package dropbox

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"looz.ws/typstify/service/settings"
)

func TestDropboxClient_AccountAndTokens(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /2/users/get_current_account", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer valid_token" {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"account_id": "dbid:12345",
			"name": map[string]string{
				"display_name": "Test User",
			},
			"email": "user@example.com",
		})
	})

	mux.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("grant_type") == "refresh_token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "new_refreshed_token",
				"expires_in":   3600,
				"token_type":   "bearer",
			})
			return
		}
		if r.Form.Get("grant_type") == "authorization_code" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "valid_token",
				"refresh_token": "valid_refresh",
				"expires_in":    3600,
				"account_id":    "dbid:12345",
			})
			return
		}
		http.Error(w, "bad grant", http.StatusBadRequest)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	st := &settings.DropboxSettings{
		AccessToken:  "valid_token",
		RefreshToken: "valid_refresh",
		TokenExpiry:  time.Now().Add(time.Hour),
	}
	client := NewClient(st)
	client.SetHosts(srv.URL, srv.URL, srv.URL)

	ctx := context.Background()
	acc, err := client.GetCurrentAccount(ctx)
	if err != nil {
		t.Fatalf("GetCurrentAccount failed: %v", err)
	}
	if acc.DisplayName != "Test User" || acc.Email != "user@example.com" {
		t.Errorf("Unexpected account: %+v", acc)
	}

	// Test Token Refresh
	st.AccessToken = ""
	st.TokenExpiry = time.Now().Add(-time.Hour)
	err = client.refreshTokenLocked(ctx)
	if err != nil {
		t.Fatalf("refreshTokenLocked failed: %v", err)
	}
	if st.AccessToken != "new_refreshed_token" {
		t.Errorf("Expected new_refreshed_token, got %s", st.AccessToken)
	}
}

func TestDropboxClient_Files(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /2/files/list_folder", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListFolderResult{
			Entries: []FileMetadata{
				{
					Tag:            "file",
					Name:           "main.typ",
					PathDisplay:    "/Typstify/demo/main.typ",
					PathLower:      "/typstify/demo/main.typ",
					Size:           120,
					ServerModified: time.Now().UTC(),
				},
			},
			HasMore: false,
		})
	})

	mux.HandleFunc("POST /2/files/upload", func(w http.ResponseWriter, r *http.Request) {
		arg := r.Header.Get("Dropbox-API-Arg")
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			http.Error(w, "empty body", http.StatusBadRequest)
			return
		}
		var req map[string]any
		_ = json.Unmarshal([]byte(arg), &req)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(FileMetadata{
			Tag:         "file",
			Name:        "main.typ",
			PathDisplay: req["path"].(string),
			Size:        int64(len(body)),
		})
	})

	mux.HandleFunc("POST /2/files/download", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("= Hello Typstify\n"))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	st := &settings.DropboxSettings{
		AccessToken: "valid_token",
	}
	client := NewClient(st)
	client.SetHosts(srv.URL, srv.URL, srv.URL)

	ctx := context.Background()

	// List
	entries, err := client.ListFolder(ctx, "/Typstify/demo", true)
	if err != nil {
		t.Fatalf("ListFolder failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "main.typ" {
		t.Fatalf("Unexpected entries: %+v", entries)
	}

	// Upload
	meta, err := client.UploadFile(ctx, "/Typstify/demo/main.typ", strings.NewReader("= Test"), time.Now())
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}
	if meta.Size != 6 {
		t.Errorf("Unexpected uploaded size: %d", meta.Size)
	}

	// Download
	rc, _, err := client.DownloadFile(ctx, "/Typstify/demo/main.typ")
	if err != nil {
		t.Fatalf("DownloadFile failed: %v", err)
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	if string(data) != "= Hello Typstify\n" {
		t.Errorf("Unexpected content: %s", string(data))
	}
}
