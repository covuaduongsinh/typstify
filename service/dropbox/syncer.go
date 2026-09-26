package dropbox

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"looz.ws/typstify/service/bus"
	"looz.ws/typstify/service/settings"
)

type SyncMode string

const (
	SyncModeTwoWay SyncMode = "two-way"
	SyncModePush   SyncMode = "push"
	SyncModePull   SyncMode = "pull"
)

type SyncResult struct {
	Mode       SyncMode      `json:"mode"`
	Uploaded   []string      `json:"uploaded"`
	Downloaded []string      `json:"downloaded"`
	Conflicts  []string      `json:"conflicts"`
	Errors     []string      `json:"errors"`
	DurationMs int64         `json:"duration_ms"`
	Timestamp  time.Time     `json:"timestamp"`
	Success    bool          `json:"success"`
}

type Syncer struct {
	client   *Client
	settings *settings.DropboxSettings
	eventbus *bus.EventBus

	mu         sync.Mutex
	syncing    bool
	lastResult *SyncResult
	stopAutoCh chan struct{}
}

func NewSyncer(client *Client, st *settings.DropboxSettings, eventbus *bus.EventBus) *Syncer {
	s := &Syncer{
		client:   client,
		settings: st,
		eventbus: eventbus,
	}
	s.restartAutoSync()
	return s
}

func (s *Syncer) IsSyncing() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.syncing
}

func (s *Syncer) LastResult() *SyncResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastResult
}

func (s *Syncer) restartAutoSync() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopAutoCh != nil {
		close(s.stopAutoCh)
		s.stopAutoCh = nil
	}

	if !s.settings.AutoSync || s.settings.AutoSyncInterval <= 0 || !s.settings.IsConnected() {
		return
	}

	s.stopAutoCh = make(chan struct{})
	stopCh := s.stopAutoCh
	interval := time.Duration(s.settings.AutoSyncInterval) * time.Minute

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				// Auto sync only if connected
				if s.settings.IsConnected() {
					// We will sync when project is set
				}
			}
		}
	}()
}

// isIgnored reports whether relative path should be excluded from sync.
func isIgnored(rel string) bool {
	slashRel := filepath.ToSlash(rel)
	parts := strings.Split(slashRel, "/")
	for _, p := range parts {
		if p == ".git" || p == ".typstify" || p == ".DS_Store" || p == "Thumbs.db" {
			return true
		}
		if strings.HasPrefix(p, ".") && p != "." {
			return true
		}
	}
	// Ignore lock files, temp files
	if strings.HasSuffix(slashRel, ".tmp") || strings.HasSuffix(slashRel, ".swp") {
		return true
	}
	return false
}

// SyncProject synchronizes the given local project directory with Dropbox folder.
func (s *Syncer) SyncProject(ctx context.Context, localProjectDir string, mode SyncMode) (*SyncResult, error) {
	s.mu.Lock()
	if s.syncing {
		s.mu.Unlock()
		return nil, fmt.Errorf("tiến trình đồng bộ đang chạy, vui lòng chờ trong giây lát")
	}
	s.syncing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.syncing = false
		s.mu.Unlock()
	}()

	start := time.Now()
	result := &SyncResult{
		Mode:       mode,
		Uploaded:   make([]string, 0),
		Downloaded: make([]string, 0),
		Conflicts:  make([]string, 0),
		Errors:     make([]string, 0),
		Timestamp:  start,
		Success:    true,
	}

	projectName := filepath.Base(localProjectDir)
	remoteBase := strings.TrimRight(s.settings.SyncFolder, "/")
	if remoteBase == "" {
		remoteBase = "/Typstify"
	}
	remoteProjectDir := remoteBase + "/" + projectName

	// 1. Scan local files
	localFiles := make(map[string]os.FileInfo)
	err := filepath.Walk(localProjectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(localProjectDir, path)
		if err != nil || rel == "." {
			return nil
		}
		if isIgnored(rel) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			localFiles[filepath.ToSlash(rel)] = info
		}
		return nil
	})
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("quét tệp cục bộ thất bại: %v", err))
		s.recordResult(result, start)
		return result, err
	}

	// 2. Fetch remote files
	remoteEntries, err := s.client.ListFolder(ctx, remoteProjectDir, true)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("lấy danh sách tệp trên Dropbox thất bại: %v", err))
		s.recordResult(result, start)
		return result, err
	}

	remoteFiles := make(map[string]FileMetadata)
	remotePrefix := strings.ToLower(remoteProjectDir + "/")
	for _, entry := range remoteEntries {
		if entry.Tag == "file" {
			// Extract relative path
			pLower := strings.ToLower(entry.PathDisplay)
			if strings.HasPrefix(pLower, remotePrefix) {
				rel := entry.PathDisplay[len(remoteProjectDir)+1:]
				remoteFiles[filepath.ToSlash(rel)] = entry
			} else {
				// Fallback if path casing differs
				rel := strings.TrimPrefix(entry.PathLower, strings.ToLower(remoteProjectDir)+"/")
				remoteFiles[filepath.ToSlash(rel)] = entry
			}
		}
	}

	// 3. Process Local Files -> Upload / Compare
	for relPath, localInfo := range localFiles {
		localAbs := filepath.Join(localProjectDir, filepath.FromSlash(relPath))
		remoteMeta, existsRemote := remoteFiles[relPath]
		remoteFile := remoteProjectDir + "/" + relPath

		if !existsRemote {
			if mode == SyncModeTwoWay || mode == SyncModePush {
				if err := s.uploadLocalFile(ctx, localAbs, remoteFile, localInfo.ModTime()); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("tải lên %s thất bại: %v", relPath, err))
				} else {
					result.Uploaded = append(result.Uploaded, relPath)
				}
			}
		} else {
			// File exists both locally and remotely
			localMod := localInfo.ModTime().UTC()
			remoteMod := remoteMeta.ClientModified.UTC()
			if remoteMod.IsZero() {
				remoteMod = remoteMeta.ServerModified.UTC()
			}

			// Time difference threshold: 2 seconds to avoid filesystem time drift
			timeDiff := localMod.Sub(remoteMod)

			if mode == SyncModePush {
				// Force upload
				if err := s.uploadLocalFile(ctx, localAbs, remoteFile, localInfo.ModTime()); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("tải lên %s: %v", relPath, err))
				} else {
					result.Uploaded = append(result.Uploaded, relPath)
				}
			} else if mode == SyncModePull {
				// Force download
				if err := s.downloadRemoteFile(ctx, remoteFile, localAbs); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("tải về %s: %v", relPath, err))
				} else {
					result.Downloaded = append(result.Downloaded, relPath)
				}
			} else {
				// Two-Way Sync
				if timeDiff > 2*time.Second {
					// Local is newer
					if err := s.uploadLocalFile(ctx, localAbs, remoteFile, localInfo.ModTime()); err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("tải lên %s: %v", relPath, err))
					} else {
						result.Uploaded = append(result.Uploaded, relPath)
					}
				} else if timeDiff < -2*time.Second {
					// Remote is newer
					if err := s.downloadRemoteFile(ctx, remoteFile, localAbs); err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("tải về %s: %v", relPath, err))
					} else {
						result.Downloaded = append(result.Downloaded, relPath)
					}
				}
			}
		}
	}

	// 4. Process Remote Files not present locally
	for relPath := range remoteFiles {
		if _, existsLocal := localFiles[relPath]; !existsLocal {
			if mode == SyncModeTwoWay || mode == SyncModePull {
				remoteFile := remoteProjectDir + "/" + relPath
				localAbs := filepath.Join(localProjectDir, filepath.FromSlash(relPath))
				if err := s.downloadRemoteFile(ctx, remoteFile, localAbs); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("tải về %s: %v", relPath, err))
				} else {
					result.Downloaded = append(result.Downloaded, relPath)
				}
			}
		}
	}

	if len(result.Errors) > 0 {
		result.Success = false
	} else {
		s.settings.LastSyncTime = time.Now()
		_ = s.settings.Save()
	}

	s.recordResult(result, start)
	return result, nil
}

func (s *Syncer) uploadLocalFile(ctx context.Context, localAbs, remoteFile string, modTime time.Time) error {
	f, err := os.Open(localAbs)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = s.client.UploadFile(ctx, remoteFile, f, modTime)
	return err
}

func (s *Syncer) downloadRemoteFile(ctx context.Context, remoteFile, localAbs string) error {
	rc, _, err := s.client.DownloadFile(ctx, remoteFile)
	if err != nil {
		return err
	}
	defer rc.Close()

	if err := os.MkdirAll(filepath.Dir(localAbs), 0755); err != nil {
		return err
	}

	tmpFile := localAbs + ".synctmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		os.Remove(tmpFile)
		return err
	}
	out.Close()

	return os.Rename(tmpFile, localAbs)
}

func (s *Syncer) recordResult(r *SyncResult, start time.Time) {
	r.DurationMs = time.Since(start).Milliseconds()
	s.mu.Lock()
	s.lastResult = r
	s.mu.Unlock()
	log.Printf("[Dropbox Sync] Completed in %dms, uploaded: %d, downloaded: %d, errors: %d",
		r.DurationMs, len(r.Uploaded), len(r.Downloaded), len(r.Errors))
}
