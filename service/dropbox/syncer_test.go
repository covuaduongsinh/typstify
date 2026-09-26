package dropbox

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"looz.ws/typstify/service/settings"
)

type mockDropboxStorage struct {
	mu    sync.Mutex
	files map[string][]byte
	times map[string]time.Time
}

func newMockDropboxStorage() *mockDropboxStorage {
	return &mockDropboxStorage{
		files: make(map[string][]byte),
		times: make(map[string]time.Time),
	}
}

func TestSyncer_TwoWaySync(t *testing.T) {
	storage := newMockDropboxStorage()

	// Setup remote mock server
	mux := http.NewServeMux()
	mux.HandleFunc("POST /2/files/list_folder", func(w http.ResponseWriter, r *http.Request) {
		storage.mu.Lock()
		defer storage.mu.Unlock()

		var entries []FileMetadata
		for p, data := range storage.files {
			entries = append(entries, FileMetadata{
				Tag:            "file",
				Name:           filepath.Base(p),
				PathDisplay:    p,
				PathLower:      p,
				Size:           int64(len(data)),
				ClientModified: storage.times[p],
				ServerModified: storage.times[p],
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListFolderResult{Entries: entries})
	})

	mux.HandleFunc("POST /2/files/upload", func(w http.ResponseWriter, r *http.Request) {
		arg := r.Header.Get("Dropbox-API-Arg")
		var req map[string]any
		_ = json.Unmarshal([]byte(arg), &req)
		p := req["path"].(string)

		body, _ := io.ReadAll(r.Body)
		storage.mu.Lock()
		storage.files[p] = body
		storage.times[p] = time.Now().UTC()
		storage.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(FileMetadata{
			Tag:         "file",
			Name:        filepath.Base(p),
			PathDisplay: p,
			Size:        int64(len(body)),
		})
	})

	mux.HandleFunc("POST /2/files/download", func(w http.ResponseWriter, r *http.Request) {
		arg := r.Header.Get("Dropbox-API-Arg")
		var req map[string]any
		_ = json.Unmarshal([]byte(arg), &req)
		p := req["path"].(string)

		storage.mu.Lock()
		data, ok := storage.files[p]
		storage.mu.Unlock()

		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = io.Copy(w, bytes.NewReader(data))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create temp local project dir
	tempDir, err := os.MkdirTemp("", "typstify-sync-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a local file
	localFile := filepath.Join(tempDir, "chapter1.typ")
	if err := os.WriteFile(localFile, []byte("= Chapter 1\nLocal Content"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Setup remote initial file
	projName := filepath.Base(tempDir)
	remotePath := "/Typstify/" + projName + "/remote_only.typ"
	storage.files[remotePath] = []byte("= Remote Only")
	storage.times[remotePath] = time.Now().UTC()

	st := &settings.DropboxSettings{
		AccessToken: "test_token",
		SyncFolder:  "/Typstify",
	}
	client := NewClient(st)
	client.SetHosts(srv.URL, srv.URL, srv.URL)
	syncer := NewSyncer(client, st, nil)

	ctx := context.Background()
	res, err := syncer.SyncProject(ctx, tempDir, SyncModeTwoWay)
	if err != nil {
		t.Fatalf("SyncProject failed: %v", err)
	}
	if !res.Success {
		t.Fatalf("Sync was not successful: %+v", res.Errors)
	}

	// 1. local file should be uploaded to Dropbox
	uploadedRemotePath := "/Typstify/" + projName + "/chapter1.typ"
	if _, ok := storage.files[uploadedRemotePath]; !ok {
		t.Errorf("Local file was not uploaded to Dropbox")
	}

	// 2. remote file should be downloaded to local
	downloadedLocalPath := filepath.Join(tempDir, "remote_only.typ")
	if data, err := os.ReadFile(downloadedLocalPath); err != nil || string(data) != "= Remote Only" {
		t.Errorf("Remote file was not downloaded to local")
	}
}
