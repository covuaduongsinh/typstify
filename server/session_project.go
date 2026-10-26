package server

import (
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"looz.ws/typstify/lsp"
	"looz.ws/typstify/service"
)

// projectHeader names the project the web UI has open in that tab
// (URL-encoded directory). Two tabs of one browser share the session, so
// when one opens another project the other is told (409) instead of
// silently working on it (docs/plans/plan_phase3b_per_user_projects.md).
const projectHeader = "X-Typstify-Project"

var errProjectChanged = errors.New("Dự án đã được đổi ở một tab khác; hãy tải lại trang")

// projectDir is the project a request works on: its session's project,
// else the server's default one (the -project flag, or the project opened
// without a session: auth disabled, or an API token).
func (s *Server) projectDir(r *http.Request) string {
	if dir, ok := s.auth.sessionProject(r); ok && dir != "" {
		return dir
	}
	return s.appSrv.CurrentProjectDir()
}

// projectRoot is projectDir for routes that need a project open; it also
// refuses a tab that still shows another project.
func (s *Server) projectRoot(r *http.Request) (string, error) {
	root := s.projectDir(r)
	if root == "" {
		return "", errNoProjectOpen
	}
	if want := r.Header.Get(projectHeader); want != "" {
		if dir, err := url.PathUnescape(want); err == nil && !sameDir(dir, root) {
			return "", errProjectChanged
		}
	}
	s.markActive(root)
	return root, nil
}

// openProjectFor makes dir the request's project: its session's, or the
// server default when there is no session.
func (s *Server) openProjectFor(r *http.Request, dir string) {
	if !s.auth.setSessionProject(r, dir) {
		s.appSrv.SetDefaultProjectDir(dir)
	} else {
		s.appSrv.Workspace().TouchRecent(dir)
	}
	s.appSrv.Runtimes().Touch(dir)
	s.markActive(dir)
}

// activeProjects remembers which projects requests touched, for the
// periodic history snapshots (startHistorySnapshots).
type activeProjects struct {
	mu   sync.Mutex
	dirs map[string]time.Time
}

func (s *Server) markActive(dir string) {
	s.active.mu.Lock()
	if s.active.dirs == nil {
		s.active.dirs = map[string]time.Time{}
	}
	s.active.dirs[dir] = time.Now()
	s.active.mu.Unlock()
}

// activeSince lists the projects touched after t.
func (s *Server) activeSince(t time.Time) []string {
	s.active.mu.Lock()
	defer s.active.mu.Unlock()
	var out []string
	for dir, at := range s.active.dirs {
		if at.After(t) {
			out = append(out, dir)
		} else if time.Since(at) > 24*time.Hour {
			delete(s.active.dirs, dir)
		}
	}
	return out
}

// tooManyProjectsMessage explains a failed Runtimes.Acquire to the user.
func tooManyProjectsMessage(err error) string {
	if errors.Is(err, service.ErrTooManyProjects) {
		return "Máy chủ đang mở tối đa số dự án cùng lúc; hãy thử lại khi có người đóng dự án"
	}
	return err.Error()
}

// projectPreview is the tinymist preview of the request's project, nil
// until it was started (POST /api/preview/restart) or once its runtime was
// closed for being idle.
func (s *Server) projectPreview(r *http.Request) *lsp.PreviewService {
	dir := s.projectDir(r)
	if dir == "" {
		return nil
	}
	rt := s.appSrv.Runtimes().Get(dir)
	if rt == nil {
		return nil
	}
	s.appSrv.Runtimes().Touch(dir)
	return rt.Preview()
}
