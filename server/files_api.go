package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// checkProjectRoot rejects abs when the server was started with a
// ProjectRoot confinement (see server.Options) and abs falls outside of it.
// A no-op when unconfined.
func (s *Server) checkProjectRoot(abs string) error {
	if s.opts.ProjectRoot == "" || isUnderRoot(s.opts.ProjectRoot, abs) {
		return nil
	}
	return fmt.Errorf(
		"projects must live under %s on this server -- anywhere else (including this path) is wiped on the next deploy, since only %s is backed by persistent storage",
		s.opts.ProjectRoot, s.opts.ProjectRoot,
	)
}

type treeEntry struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"` // slash-separated, relative to the project root
	IsDir   bool      `json:"isDir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

func (s *Server) projectRoot() (string, error) {
	root := s.appSrv.CurrentProjectDir()
	if root == "" {
		return "", errNoProjectOpen
	}
	return root, nil
}

// handleTree lists the entries of a single directory (non-recursive; the
// frontend file tree expands lazily by calling this again per directory).
func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	dir, err := resolveInRoot(root, r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	result := make([]treeEntry, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if name == ".typstify" || name == ".git" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		full := filepath.Join(dir, name)
		rel, err := relPath(root, full)
		if err != nil {
			continue
		}
		result = append(result, treeEntry{
			Name:    name,
			Path:    rel,
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return result[i].Name < result[j].Name
	})

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleFileGet(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	p, err := resolveInRoot(root, r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	data, err := os.ReadFile(p)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(data)
}

// handleFilePut overwrites a file's full content. It is meant for explicit
// "save" actions (e.g. from an export/import flow); the editor's live typing
// is synced through /ws/lsp's didOpen/didChange instead, which additionally
// keeps tinymist's in-memory document cache up to date.
func (s *Server) handleFilePut(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	p, err := resolveInRoot(root, r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	data, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.WriteFile(p, data, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type createFileRequest struct {
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

func (s *Server) handleFileCreate(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	var req createFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	p, err := resolveInRoot(root, req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.IsDir {
		err = os.MkdirAll(p, 0755)
	} else {
		if mkErr := os.MkdirAll(filepath.Dir(p), 0755); mkErr != nil {
			writeError(w, http.StatusInternalServerError, mkErr.Error())
			return
		}
		var f *os.File
		f, err = os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if f != nil {
			f.Close()
		}
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleFileDelete(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	p, err := resolveInRoot(root, r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if p == filepath.Clean(root) {
		writeError(w, http.StatusBadRequest, "cannot delete the project root")
		return
	}

	if err := os.RemoveAll(p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type renameRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (s *Server) handleFileRename(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	var req renameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	from, err := resolveInRoot(root, req.From)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	to, err := resolveInRoot(root, req.To)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := os.MkdirAll(filepath.Dir(to), 0755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.Rename(from, to); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleCurrentProject(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"path": s.appSrv.CurrentProjectDir()})
}

func (s *Server) handleRecentProjects(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.appSrv.Workspace().GetHistory(20))
}

type openProjectRequest struct {
	Path string `json:"path"`
}

func (s *Server) handleOpenProject(w http.ResponseWriter, r *http.Request) {
	var req openProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	abs, err := filepath.Abs(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.checkProjectRoot(abs); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		writeError(w, http.StatusBadRequest, "not a directory")
		return
	}

	s.appSrv.SetProjectDir(abs)
	writeJSON(w, http.StatusOK, map[string]string{"path": abs})
}

type createProjectRequest struct {
	Path string `json:"path"`
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	abs, err := filepath.Abs(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.checkProjectRoot(abs); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	mainFile := filepath.Join(abs, "main.typ")
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		_ = os.WriteFile(mainFile, []byte("= "+filepath.Base(abs)+"\n\n"), 0644)
	}

	s.appSrv.SetProjectDir(abs)
	writeJSON(w, http.StatusOK, map[string]string{"path": abs})
}
