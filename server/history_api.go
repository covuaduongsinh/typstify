package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"looz.ws/typstify/service/history"
)

// Transport for the version history (service/history).

const historyAutoInterval = 5 * time.Minute

// envMegabytes reads a size in MiB from the environment (0 when unset or
// invalid, meaning the default).
func envMegabytes(name string) int64 {
	n, err := strconv.ParseInt(os.Getenv(name), 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n << 20
}

// historyStore is the history of the open folder: keyed by its project's
// registry id when registered (survives moving the folder), plus the
// subfolder when one was opened (versions hold paths relative to the
// folder they were taken in); else by its path.
func (s *Server) historyStore(root string) *history.Store {
	if p, ok := s.projects.Covering(root); ok {
		key := "project:" + p.ID
		if rel, err := filepath.Rel(p.Dir, root); err == nil && rel != "." {
			key += ":" + strings.ToLower(filepath.ToSlash(rel))
		}
		return s.history.For(key)
	}
	key := filepath.Clean(root)
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	return s.history.For("dir:" + key)
}

// runHistoryAuto takes an automatic version of the open project every few
// minutes when something changed, whoever changed it.
func (s *Server) runHistoryAuto(ctx context.Context) {
	t := time.NewTicker(historyAutoInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if s.appSrv == nil {
				continue
			}
			root := s.appSrv.CurrentProjectDir()
			if root == "" {
				continue
			}
			if _, _, err := s.historyStore(root).Take(root, "Tự động", "", history.KindAuto, false); err != nil {
				log.Printf("history: %v", err)
			}
		}
	}
}

func (s *Server) handleHistoryList(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	list, err := s.historyStore(root).List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []history.Snapshot{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) requestDisplayName(r *http.Request) string {
	user, _ := s.requestUser(r)
	return s.auth.displayNameOf(user)
}

func (s *Server) handleHistoryCreate(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	var req struct {
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	label := strings.TrimSpace(req.Label)
	if label == "" || len([]rune(label)) > 120 {
		writeError(w, http.StatusBadRequest, "Tên phiên bản cần từ 1 đến 120 ký tự")
		return
	}
	snap, _, err := s.historyStore(root).Take(root, s.requestDisplayName(r), label, history.KindManual, true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func historyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, history.ErrNotFound), errors.Is(err, history.ErrNoFile):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func (s *Server) handleHistoryGet(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	st := s.historyStore(root)
	snap, err := st.Get(r.PathValue("id"))
	if err != nil {
		historyError(w, err)
		return
	}
	changes, err := st.Compare(snap.ID, root)
	if err != nil {
		historyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"snapshot": snap, "changes": changes})
}

func (s *Server) handleHistoryFile(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	data, err := s.historyStore(root).Read(r.PathValue("id"), r.URL.Query().Get("path"))
	if err != nil {
		historyError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	_, _ = w.Write(data)
}

// handleHistoryRestore writes a version's files back. Files open in a
// co-editing room are written to disk like any outside edit; the room
// folds the change in (or asks the people editing).
func (s *Server) handleHistoryRestore(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	var req struct {
		Paths       []string `json:"paths"`
		DeleteNewer bool     `json:"deleteNewer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	target := func(rel string) (string, error) {
		// Folder names compare case-insensitively on Windows and macOS.
		for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
			if strings.EqualFold(part, ".git") || strings.EqualFold(part, ".typstify") {
				return "", errors.New("đường dẫn được bảo vệ: " + rel)
			}
		}
		p, err := resolveInRoot(root, rel)
		if err != nil {
			return "", err
		}
		if isProtectedPath(root, p) {
			return "", errors.New("đường dẫn được bảo vệ: " + rel)
		}
		return p, nil
	}
	writer := history.Writer{
		Write: func(rel string, data []byte) error {
			p, err := target(rel)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return err
			}
			return writeFileAtomic(p, data)
		},
		Remove: func(rel string) error {
			p, err := target(rel)
			if err != nil {
				return err
			}
			return os.Remove(p)
		},
	}
	safety, err := s.historyStore(root).Restore(r.PathValue("id"), root, s.requestDisplayName(r), req.Paths, req.DeleteNewer, writer)
	if err != nil {
		historyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "backup": safety})
}
