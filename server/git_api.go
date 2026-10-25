package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"looz.ws/typstify/service/gitsync"
	"looz.ws/typstify/service/history"
	"looz.ws/typstify/service/projects"
)

// Transport for Git sync (service/gitsync). Settings belong to the
// project owner (admins for unregistered projects); editors commit, push
// and pull; viewers see the status.

func (s *Server) newGitSync() *gitsync.Service {
	return &gitsync.Service{
		Base:         filepath.Join(s.storageDir, "git"),
		AllowPrivate: os.Getenv("TYPSTIFY_GIT_ALLOW_PRIVATE") == "1",
	}
}

// gitKey identifies the open project's Git settings, like its history.
func (s *Server) gitKey(root string) string {
	if p, ok := s.projects.Covering(root); ok {
		return "project-" + p.ID
	}
	key := filepath.Clean(root)
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	sum := sha256.Sum256([]byte(key))
	return "dir-" + hex.EncodeToString(sum[:8])
}

// gitRoot is the open folder for Git operations. A subfolder of a
// registered project is refused: its settings belong to the whole
// project, and a repository there would nest inside the project's.
func (s *Server) gitRoot(w http.ResponseWriter) (string, bool) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return "", false
	}
	if p, ok := s.projects.Covering(root); ok && !sameDir(p.Dir, root) {
		writeError(w, http.StatusConflict, "Hãy mở thư mục gốc của dự án để dùng Git")
		return "", false
	}
	return root, true
}

// canConfigureGit: the owner of a registered project, else an admin.
func (s *Server) canConfigureGit(r *http.Request) bool {
	_, role, registered := s.currentProjectRole(r)
	if registered {
		return role.AtLeast(projects.RoleOwner)
	}
	_, admin := s.requestUser(r)
	return admin
}

func gitError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, gitsync.ErrNoRemote), errors.Is(err, gitsync.ErrNotRepo):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, gitsync.ErrDiverged), errors.Is(err, gitsync.ErrDirty):
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "code": gitCode(err)})
	case errors.Is(err, gitsync.ErrBadRemote), errors.Is(err, gitsync.ErrNothingToCommit):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusBadGateway, err.Error())
	}
}

func gitCode(err error) string {
	if errors.Is(err, gitsync.ErrDiverged) {
		return "diverged"
	}
	return "dirty"
}

func (s *Server) handleGitStatus(w http.ResponseWriter, r *http.Request) {
	root, ok := s.gitRoot(w)
	if !ok {
		return
	}
	// Contacting the remote is the editors' call: it costs a network
	// round trip and holds the project's Git lock meanwhile.
	fetch := r.URL.Query().Get("fetch") == "1" && s.hasProjectRole(r, projects.RoleEditor)
	st, err := s.git.Status(r.Context(), s.gitKey(root), root, fetch)
	if err != nil {
		gitError(w, err)
		return
	}
	cfg, _ := s.git.LoadConfig(s.gitKey(root))
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       st,
		"canConfigure": s.canConfigureGit(r),
		"config": map[string]string{
			"remote": cfg.Remote, "branch": cfg.Branch, "username": cfg.Username,
			"authorName": cfg.AuthorName, "authorEmail": cfg.AuthorEmail,
		},
	})
}

func (s *Server) handleGitConfig(w http.ResponseWriter, r *http.Request) {
	root, ok := s.gitRoot(w)
	if !ok {
		return
	}
	if !s.canConfigureGit(r) {
		writeError(w, http.StatusForbidden, "Chỉ chủ dự án được cấu hình Git")
		return
	}
	var cfg gitsync.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	cfg.Remote = strings.TrimSpace(cfg.Remote)
	cfg.Branch = strings.TrimSpace(cfg.Branch)
	if err := s.git.SaveConfig(s.gitKey(root), cfg); err != nil {
		gitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleGitCommit(w http.ResponseWriter, r *http.Request) {
	root, ok := s.gitRoot(w)
	if !ok {
		return
	}
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		writeError(w, http.StatusBadRequest, "Cần nội dung commit")
		return
	}
	user, _ := s.requestUser(r)
	c, err := s.git.Commit(s.gitKey(root), root, req.Message, s.auth.displayNameOf(user), "")
	if err != nil {
		gitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleGitPush(w http.ResponseWriter, r *http.Request) {
	root, ok := s.gitRoot(w)
	if !ok {
		return
	}
	var req struct {
		Force bool `json:"force"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	// Overwriting the shared remote branch is the owner's call.
	if req.Force && !s.canConfigureGit(r) {
		writeError(w, http.StatusForbidden, "Chỉ chủ dự án được ghi đè nhánh trên kho")
		return
	}
	if err := s.git.Push(r.Context(), s.gitKey(root), root, req.Force); err != nil {
		gitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleGitPull brings the remote branch in. The current files are saved
// as a history version first, so even "take the remote side" can be
// undone from the History panel.
func (s *Server) handleGitPull(w http.ResponseWriter, r *http.Request) {
	root, ok := s.gitRoot(w)
	if !ok {
		return
	}
	var req struct {
		TakeRemote bool `json:"takeRemote"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if _, _, err := s.historyStore(root).Take(root, s.requestDisplayName(r), "Trước khi kéo từ Git", history.KindRestore, false); err != nil {
		writeError(w, http.StatusInternalServerError, "không lưu được phiên bản an toàn: "+err.Error())
		return
	}
	res, err := s.git.Pull(r.Context(), s.gitKey(root), root, req.TakeRemote)
	if err != nil {
		gitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
