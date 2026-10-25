package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"looz.ws/typstify/service/projects"
	"looz.ws/typstify/service/universe"
	"looz.ws/typstify/service/wasmassets"
)

const projectsFileName = "projects.json"

func openProjectRegistry(storageDir string) *projects.Registry {
	path := ""
	if storageDir != "" {
		path = filepath.Join(storageDir, projectsFileName)
	}
	reg, err := projects.Open(path)
	if err != nil {
		// A corrupt registry must not take the whole server down; start
		// empty in memory so it is not overwritten, and say so loudly.
		log.Printf("projects: %v -- using an empty in-memory registry", err)
		reg, _ = projects.Open("")
	}
	return reg
}

// requestUser returns the caller's username and whether they are an
// admin. With auth disabled everyone is an anonymous admin.
func (s *Server) requestUser(r *http.Request) (string, bool) {
	if !s.auth.enabled() {
		return "", true
	}
	user, _ := s.auth.authenticatedUser(r)
	return user, s.auth.isAdmin(user)
}

// currentProjectRole reports the caller's role on the currently open
// project. registered is false when the open directory isn't in the
// registry; such legacy projects keep the old "any logged-in user"
// behavior.
func (s *Server) currentProjectRole(r *http.Request) (p projects.Project, role projects.Role, registered bool) {
	dir := s.appSrv.CurrentProjectDir()
	if dir == "" {
		return projects.Project{}, "", false
	}
	// A subfolder of a registered project is still that project.
	p, ok := s.projects.Covering(dir)
	if !ok {
		return projects.Project{}, "", false
	}
	user, admin := s.requestUser(r)
	return p, s.projects.RoleOf(p.ID, user, admin), true
}

// checkDirAccess decides whether the caller may open (min viewer) or
// create in (min editor) dir by path: inside a registered project the
// caller needs min on it, and a folder that contains other users'
// registered projects is admin-only, since opening it would expose them.
func (s *Server) checkDirAccess(r *http.Request, dir string, min projects.Role) (projects.Project, bool, error) {
	user, admin := s.requestUser(r)
	if p, ok := s.projects.Covering(dir); ok {
		if !s.projects.RoleOf(p.ID, user, admin).AtLeast(min) {
			return p, true, errors.New(forbiddenMessage(min))
		}
		return p, true, nil
	}
	if !admin && s.projects.ContainsAny(dir) {
		return projects.Project{}, false, errors.New("Thư mục này chứa dự án của người khác; chỉ quản trị viên được mở")
	}
	return projects.Project{}, false, nil
}

// handleRole registers an authenticated route that additionally needs at
// least min on the open project (when that project is registered).
func (s *Server) handleRole(pattern string, min projects.Role, fn http.HandlerFunc) {
	s.handle(pattern, s.roleGate(min, fn))
}

// roleGate wraps fn with the open-project role check, for routes mounted
// outside handleRole (the tinymist preview proxy).
func (s *Server) roleGate(min projects.Role, fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.hasProjectRole(r, min) {
			writeError(w, http.StatusForbidden, forbiddenMessage(min))
			return
		}
		fn(w, r)
	}
}

// hasProjectRole reports whether the caller has at least min on the open
// project; unregistered projects keep the legacy open behavior.
func (s *Server) hasProjectRole(r *http.Request, min projects.Role) bool {
	_, role, registered := s.currentProjectRole(r)
	return !registered || role.AtLeast(min)
}

// handleAdmin registers a route only admins may use: server-wide settings
// (which include executable paths), fonts, integrations and agent control.
func (s *Server) handleAdmin(pattern string, fn http.HandlerFunc) {
	s.handle(pattern, func(w http.ResponseWriter, r *http.Request) {
		if _, admin := s.requestUser(r); !admin {
			writeError(w, http.StatusForbidden, "Chỉ quản trị viên được thực hiện thao tác này")
			return
		}
		fn(w, r)
	})
}

func forbiddenMessage(min projects.Role) string {
	switch min {
	case projects.RoleEditor:
		return "Bạn chỉ có quyền xem dự án này, không thể chỉnh sửa"
	case projects.RoleOwner:
		return "Chỉ chủ sở hữu dự án được thực hiện thao tác này"
	}
	return "Bạn không có quyền truy cập dự án này"
}

// projectSlug turns a project name into a directory name: letters and
// digits are kept (any script), everything else collapses to "-".
func projectSlug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "du-an"
	}
	return slug
}

// projectsBaseDir is where POST /api/projects creates new projects.
func (s *Server) projectsBaseDir() string {
	if s.opts.ProjectRoot != "" {
		return s.opts.ProjectRoot
	}
	if s.storageDir != "" {
		return filepath.Join(s.storageDir, "projects")
	}
	return filepath.Join(os.TempDir(), "typstify-projects")
}

// projectView is what non-owner endpoints return about a project: no
// server path and no member list (those stay with the owner-only
// /members endpoint).
type projectView struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Owner     string        `json:"owner"`
	Role      projects.Role `json:"role"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

func viewOf(p projects.Project, role projects.Role) projectView {
	return projectView{ID: p.ID, Name: p.Name, Owner: p.Owner, Role: role, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
}

func (s *Server) handleProjectsList(w http.ResponseWriter, r *http.Request) {
	user, admin := s.requestUser(r)
	list := s.projects.ListFor(user, admin)
	out := make([]projectView, 0, len(list))
	for _, p := range list {
		out = append(out, viewOf(p.Project, p.Role))
	}
	writeJSON(w, http.StatusOK, out)
}

type createNamedProjectRequest struct {
	Name string `json:"name"`
	// Template starts the project from a Typst Universe template
	// (@preview/name:version), like `typst init`.
	Template *struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"template,omitempty"`
	// MainContent, when no template is given, is the text of main.typ
	// (the built-in chess templates).
	MainContent string `json:"mainContent,omitempty"`
}

// maxMainContent bounds MainContent.
const maxMainContent = 1 << 20

// createdProject is POST /api/projects' reply: the project and the file
// to open first.
type createdProject struct {
	projectView
	Main string `json:"main"`
}

func (s *Server) handleProjectsCreate(w http.ResponseWriter, r *http.Request) {
	var req createNamedProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "Tên dự án không được để trống")
		return
	}
	if len([]rune(name)) > 120 {
		writeError(w, http.StatusBadRequest, "Tên dự án quá dài")
		return
	}
	if len(req.MainContent) > maxMainContent {
		writeError(w, http.StatusBadRequest, "Nội dung mẫu quá lớn")
		return
	}
	var spec wasmassets.Spec
	if req.Template != nil {
		var err error
		if spec, err = wasmassets.ParseSpec("preview", req.Template.Name, req.Template.Version); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	base := s.projectsBaseDir()
	slug := projectSlug(name)
	if err := s.checkProjectRoot(filepath.Join(base, slug)); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// The folder is claimed with an exclusive Mkdir: two requests for the
	// same name never share one, so the cleanup below only ever removes a
	// folder this request created.
	dir := ""
	for i := 1; i <= 1000 && dir == ""; i++ {
		candidate := filepath.Join(base, slug)
		if i > 1 {
			candidate = filepath.Join(base, slug+"-"+strconv.Itoa(i))
		}
		err := os.Mkdir(candidate, 0o755)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		dir = candidate
	}
	if dir == "" {
		writeError(w, http.StatusConflict, "Quá nhiều dự án trùng tên")
		return
	}
	main, status, err := s.seedProject(r, dir, name, req.MainContent, spec)
	if err != nil {
		// The folder was created by this request: leave nothing half made.
		_ = os.RemoveAll(dir)
		writeError(w, status, err.Error())
		return
	}

	user, _ := s.requestUser(r)
	p, err := s.projects.Create(name, dir, user)
	if err != nil {
		_ = os.RemoveAll(dir)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, createdProject{projectView: viewOf(p, projects.RoleOwner), Main: main})
}

// seedProject writes the first files of a new project in dir: a Universe
// template (spec set), the given main.typ text, or a heading. It returns
// the file to open and, on failure, the HTTP status to report.
func (s *Server) seedProject(r *http.Request, dir, name, mainContent string, spec wasmassets.Spec) (string, int, error) {
	if spec.Name != "" {
		pkgDir, err := s.wasmPackages.Dir(r.Context(), spec)
		if errors.Is(err, wasmassets.ErrNotFound) {
			return "", http.StatusNotFound, err
		}
		if err != nil {
			return "", http.StatusBadGateway, err
		}
		main, err := universe.CopyTemplate(pkgDir, dir)
		if errors.Is(err, universe.ErrNoTemplate) {
			return "", http.StatusBadRequest, err
		}
		if err != nil {
			return "", http.StatusInternalServerError, err
		}
		return main, 0, nil
	}
	content := mainContent
	if content == "" {
		content = "= " + name + "\n\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "main.typ"), []byte(content), 0o644); err != nil {
		return "", http.StatusInternalServerError, err
	}
	return "main.typ", 0, nil
}

// projectForRequest loads {id} and the caller's role, writing 404 when the
// project doesn't exist or the caller has no access to it.
func (s *Server) projectForRequest(w http.ResponseWriter, r *http.Request, min projects.Role) (projects.Project, projects.Role, bool) {
	p, ok := s.projects.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, projects.ErrNotFound.Error())
		return projects.Project{}, "", false
	}
	user, admin := s.requestUser(r)
	role := s.projects.RoleOf(p.ID, user, admin)
	if role == "" {
		writeError(w, http.StatusNotFound, projects.ErrNotFound.Error())
		return projects.Project{}, "", false
	}
	if !role.AtLeast(min) {
		writeError(w, http.StatusForbidden, forbiddenMessage(min))
		return projects.Project{}, "", false
	}
	return p, role, true
}

func (s *Server) handleProjectOpen(w http.ResponseWriter, r *http.Request) {
	p, role, ok := s.projectForRequest(w, r, projects.RoleViewer)
	if !ok {
		return
	}
	if info, err := os.Stat(p.Dir); err != nil || !info.IsDir() {
		writeError(w, http.StatusConflict, "Thư mục dự án không còn tồn tại: "+p.Dir)
		return
	}
	s.appSrv.SetProjectDir(p.Dir)
	s.projects.Touch(p.ID)
	writeJSON(w, http.StatusOK, map[string]any{"path": p.Dir, "project": viewOf(p, role)})
}

func (s *Server) handleProjectDelete(w http.ResponseWriter, r *http.Request) {
	p, _, ok := s.projectForRequest(w, r, projects.RoleOwner)
	if !ok {
		return
	}
	// Only unregisters: files stay on disk so nothing is lost by accident.
	if err := s.projects.Delete(p.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type memberView struct {
	Username string        `json:"username"`
	Role     projects.Role `json:"role"`
}

func (s *Server) handleProjectMembers(w http.ResponseWriter, r *http.Request) {
	p, _, ok := s.projectForRequest(w, r, projects.RoleOwner)
	if !ok {
		return
	}
	out := []memberView{{Username: p.Owner, Role: projects.RoleOwner}}
	for u, role := range p.Members {
		out = append(out, memberView{Username: u, Role: role})
	}
	writeJSON(w, http.StatusOK, out)
}

type setMemberRequest struct {
	Role string `json:"role"`
}

func (s *Server) handleProjectSetMember(w http.ResponseWriter, r *http.Request) {
	p, _, ok := s.projectForRequest(w, r, projects.RoleOwner)
	if !ok {
		return
	}
	var req setMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}
	role, err := projects.ParseRole(req.Role)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	user := strings.ToLower(strings.TrimSpace(r.PathValue("user")))
	if !s.auth.userExists(user) {
		writeError(w, http.StatusNotFound, "Không có tài khoản "+user)
		return
	}
	if err := s.projects.SetMember(p.ID, user, role); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleProjectRemoveMember(w http.ResponseWriter, r *http.Request) {
	p, _, ok := s.projectForRequest(w, r, projects.RoleOwner)
	if !ok {
		return
	}
	if err := s.projects.RemoveMember(p.ID, strings.ToLower(r.PathValue("user"))); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleProjectShares(w http.ResponseWriter, r *http.Request) {
	p, _, ok := s.projectForRequest(w, r, projects.RoleOwner)
	if !ok {
		return
	}
	links := s.projects.ListShares(p.ID)
	for i := range links {
		links[i].TokenHash = "" // never needed by the client
	}
	writeJSON(w, http.StatusOK, links)
}

type createShareRequest struct {
	Role          string `json:"role"`
	ExpiresInDays int    `json:"expiresInDays"`
}

func (s *Server) handleProjectCreateShare(w http.ResponseWriter, r *http.Request) {
	p, _, ok := s.projectForRequest(w, r, projects.RoleOwner)
	if !ok {
		return
	}
	var req createShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}
	role, err := projects.ParseRole(req.Role)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ExpiresInDays < 0 || req.ExpiresInDays > 3650 {
		writeError(w, http.StatusBadRequest, "Thời hạn không hợp lệ")
		return
	}
	now := time.Now()
	var expires time.Time
	if req.ExpiresInDays > 0 {
		expires = now.Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour).UTC()
	}
	user, _ := s.requestUser(r)
	token, link, err := s.projects.CreateShare(p.ID, role, expires, user, now)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	link.TokenHash = ""
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "link": link})
}

func (s *Server) handleProjectRevokeShare(w http.ResponseWriter, r *http.Request) {
	p, _, ok := s.projectForRequest(w, r, projects.RoleOwner)
	if !ok {
		return
	}
	if err := s.projects.RevokeShare(p.ID, r.PathValue("shareId")); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type joinProjectRequest struct {
	Token string `json:"token"`
}

func (s *Server) handleProjectJoin(w http.ResponseWriter, r *http.Request) {
	var req joinProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}
	user, _ := s.requestUser(r)
	if user == "" {
		writeError(w, http.StatusBadRequest, "Cần đăng nhập bằng tài khoản người dùng để tham gia dự án")
		return
	}
	p, role, err := s.projects.Join(strings.TrimSpace(req.Token), user, time.Now())
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, viewOf(p, role))
}

// handleWorkspaceRole tells the editor what the caller may do on the open
// project, so the UI can switch to read-only mode.
func (s *Server) handleWorkspaceRole(w http.ResponseWriter, r *http.Request) {
	p, role, registered := s.currentProjectRole(r)
	if !registered {
		writeJSON(w, http.StatusOK, map[string]any{"registered": false, "role": projects.RoleEditor})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"registered": true, "role": role, "project": viewOf(p, role)})
}
