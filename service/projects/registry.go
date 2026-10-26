// Package projects keeps the server's registry of projects: who owns each
// project directory, which other users may open it and with what role, and
// the share links that let someone join a project. It holds no transport
// code (server/projects_api.go exposes it over HTTP) and does not touch the
// project files themselves.
package projects

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Role is a user's access level on one project. The levels mirror
// typst.app's Write / Review / Read plus the owner.
type Role string

const (
	RoleOwner     Role = "owner"
	RoleEditor    Role = "editor"    // read + write files
	RoleCommenter Role = "commenter" // read + comment (comments arrive with collaboration)
	RoleViewer    Role = "viewer"    // read only
)

func (r Role) rank() int {
	switch r {
	case RoleOwner:
		return 4
	case RoleEditor:
		return 3
	case RoleCommenter:
		return 2
	case RoleViewer:
		return 1
	}
	return 0
}

// AtLeast reports whether r grants everything min grants.
func (r Role) AtLeast(min Role) bool { return r.rank() > 0 && r.rank() >= min.rank() }

// ParseRole validates a role name coming from a request.
func ParseRole(s string) (Role, error) {
	r := Role(strings.ToLower(strings.TrimSpace(s)))
	if r.rank() == 0 {
		return "", fmt.Errorf("vai trò không hợp lệ: %q", s)
	}
	return r, nil
}

var (
	ErrNotFound     = errors.New("không tìm thấy dự án")
	ErrDirTaken     = errors.New("thư mục này đã thuộc một dự án khác")
	ErrOwner        = errors.New("không thể thay đổi hoặc xoá chủ sở hữu dự án")
	ErrShareInvalid = errors.New("liên kết chia sẻ không hợp lệ, đã hết hạn hoặc đã bị thu hồi")
)

// Project is one registered project directory.
type Project struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Dir       string          `json:"dir"`
	Owner     string          `json:"owner"`
	Members   map[string]Role `json:"members,omitempty"` // excludes Owner
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

// ShareLink lets anyone holding its token join a project with Role. Only
// the token's SHA-256 is stored.
type ShareLink struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId"`
	TokenHash string    `json:"tokenHash,omitempty"` // cleared before sending to clients
	Role      Role      `json:"role"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt,omitzero"`
	Uses      int       `json:"uses"`
}

// ProjectWithRole is a project as seen by one user.
type ProjectWithRole struct {
	Project
	Role Role `json:"role"`
}

type fileData struct {
	Projects map[string]*Project   `json:"projects"`
	Shares   map[string]*ShareLink `json:"shares"`
}

// Registry is safe for concurrent use. Every mutation is persisted before
// it returns.
type Registry struct {
	path string

	mu   sync.Mutex
	data fileData
}

// Open loads the registry stored at path; a missing file is an empty
// registry. An empty path keeps the registry in memory only.
func Open(path string) (*Registry, error) {
	r := &Registry{path: path, data: fileData{
		Projects: map[string]*Project{},
		Shares:   map[string]*ShareLink{},
	}}
	if path == "" {
		return r, nil
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &r.data); err != nil {
		return nil, fmt.Errorf("projects registry %s: %w", path, err)
	}
	if r.data.Projects == nil {
		r.data.Projects = map[string]*Project{}
	}
	if r.data.Shares == nil {
		r.data.Shares = map[string]*ShareLink{}
	}
	return r, nil
}

func (r *Registry) saveLocked() error {
	if r.path == "" {
		return nil // in-memory registry (no storage dir configured)
	}
	b, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(r.path), ".projects-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), r.path)
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func cleanDir(dir string) string {
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	return filepath.Clean(dir)
}

func sameDir(a, b string) bool {
	if filepath.Separator == '\\' {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func copyProject(p *Project) Project {
	c := *p
	c.Members = make(map[string]Role, len(p.Members))
	for k, v := range p.Members {
		c.Members[k] = v
	}
	return c
}

// Create registers dir as a new project owned by owner. An empty name
// defaults to the directory's base name.
func (r *Registry) Create(name, dir, owner string) (Project, error) {
	dir = cleanDir(dir)
	if strings.TrimSpace(name) == "" {
		name = filepath.Base(dir)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.data.Projects {
		if sameDir(p.Dir, dir) {
			return Project{}, ErrDirTaken
		}
	}
	id, err := randomHex(12)
	if err != nil {
		return Project{}, err
	}
	now := time.Now().UTC()
	p := &Project{ID: id, Name: strings.TrimSpace(name), Dir: dir, Owner: owner,
		Members: map[string]Role{}, CreatedAt: now, UpdatedAt: now}
	r.data.Projects[id] = p
	if err := r.saveLocked(); err != nil {
		delete(r.data.Projects, id)
		return Project{}, err
	}
	return copyProject(p), nil
}

// Delete unregisters a project and its share links. Files are untouched.
func (r *Registry) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data.Projects[id]; !ok {
		return ErrNotFound
	}
	delete(r.data.Projects, id)
	for sid, s := range r.data.Shares {
		if s.ProjectID == id {
			delete(r.data.Shares, sid)
		}
	}
	return r.saveLocked()
}

func (r *Registry) Get(id string) (Project, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.data.Projects[id]
	if !ok {
		return Project{}, false
	}
	return copyProject(p), true
}

// ByDir finds the project registered for dir, if any.
func (r *Registry) ByDir(dir string) (Project, bool) {
	dir = cleanDir(dir)
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.data.Projects {
		if sameDir(p.Dir, dir) {
			return copyProject(p), true
		}
	}
	return Project{}, false
}

// within reports whether dir is root or underneath it (both clean, absolute).
func within(root, dir string) bool {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return false
	}
	if filepath.Separator == '\\' && !strings.EqualFold(filepath.VolumeName(root), filepath.VolumeName(dir)) {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// Covering returns the registered project whose directory is dir or the
// closest ancestor of it: opening a subfolder of a project is still that
// project for access control.
func (r *Registry) Covering(dir string) (Project, bool) {
	dir = cleanDir(dir)
	r.mu.Lock()
	defer r.mu.Unlock()
	var best *Project
	for _, p := range r.data.Projects {
		if within(p.Dir, dir) && (best == nil || len(p.Dir) > len(best.Dir)) {
			best = p
		}
	}
	if best == nil {
		return Project{}, false
	}
	return copyProject(best), true
}

// ContainsAny reports whether some registered project lives strictly
// inside dir -- opening dir would expose that project's files.
func (r *Registry) ContainsAny(dir string) bool {
	dir = cleanDir(dir)
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.data.Projects {
		if !sameDir(p.Dir, dir) && within(dir, p.Dir) {
			return true
		}
	}
	return false
}

func roleLocked(p *Project, user string, admin bool) Role {
	if admin || p.Owner == user {
		return RoleOwner
	}
	return p.Members[user]
}

// RoleOf returns user's role on project id ("" for none). Admins are
// treated as owners of every project.
func (r *Registry) RoleOf(id, user string, admin bool) Role {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.data.Projects[id]
	if !ok {
		return ""
	}
	return roleLocked(p, user, admin)
}

// ListFor returns every project user can open, most recently updated first.
func (r *Registry) ListFor(user string, admin bool) []ProjectWithRole {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []ProjectWithRole{}
	for _, p := range r.data.Projects {
		if role := roleLocked(p, user, admin); role != "" {
			out = append(out, ProjectWithRole{Project: copyProject(p), Role: role})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out
}

// Touch bumps a project's UpdatedAt (e.g. when it is opened).
func (r *Registry) Touch(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.data.Projects[id]; ok {
		p.UpdatedAt = time.Now().UTC()
		_ = r.saveLocked()
	}
}

// SetMember grants user role on project id. The owner role cannot be
// granted this way.
func (r *Registry) SetMember(id, user string, role Role) error {
	if role.rank() == 0 || role == RoleOwner {
		return fmt.Errorf("vai trò không hợp lệ: %q", role)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.data.Projects[id]
	if !ok {
		return ErrNotFound
	}
	if p.Owner == user {
		return ErrOwner
	}
	if p.Members == nil {
		p.Members = map[string]Role{}
	}
	p.Members[user] = role
	p.UpdatedAt = time.Now().UTC()
	return r.saveLocked()
}

func (r *Registry) RemoveMember(id, user string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.data.Projects[id]
	if !ok {
		return ErrNotFound
	}
	if p.Owner == user {
		return ErrOwner
	}
	delete(p.Members, user)
	return r.saveLocked()
}

// CreateShare makes a share link for project id. The returned token is the
// only copy of it; a zero expires means the link never expires.
func (r *Registry) CreateShare(id string, role Role, expires time.Time, by string, now time.Time) (string, ShareLink, error) {
	if role.rank() == 0 || role == RoleOwner {
		return "", ShareLink{}, fmt.Errorf("vai trò không hợp lệ cho liên kết: %q", role)
	}
	token, err := randomHex(32)
	if err != nil {
		return "", ShareLink{}, err
	}
	sid, err := randomHex(8)
	if err != nil {
		return "", ShareLink{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data.Projects[id]; !ok {
		return "", ShareLink{}, ErrNotFound
	}
	l := &ShareLink{ID: sid, ProjectID: id, TokenHash: hashToken(token), Role: role,
		CreatedBy: by, CreatedAt: now.UTC(), ExpiresAt: expires}
	r.data.Shares[sid] = l
	if err := r.saveLocked(); err != nil {
		delete(r.data.Shares, sid)
		return "", ShareLink{}, err
	}
	return token, *l, nil
}

func (r *Registry) ListShares(id string) []ShareLink {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []ShareLink{}
	for _, s := range r.data.Shares {
		if s.ProjectID == id {
			out = append(out, *s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (r *Registry) RevokeShare(projectID, shareID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.data.Shares[shareID]
	if !ok || s.ProjectID != projectID {
		return ErrShareInvalid
	}
	delete(r.data.Shares, shareID)
	return r.saveLocked()
}

// Join adds user to the project behind token with the link's role, never
// lowering a role the user already has. It returns the user's resulting
// role.
func (r *Registry) Join(token, user string, now time.Time) (Project, Role, error) {
	h := hashToken(token)
	r.mu.Lock()
	defer r.mu.Unlock()
	var link *ShareLink
	for _, s := range r.data.Shares {
		if s.TokenHash == h {
			link = s
			break
		}
	}
	if link == nil || (!link.ExpiresAt.IsZero() && now.After(link.ExpiresAt)) {
		return Project{}, "", ErrShareInvalid
	}
	p, ok := r.data.Projects[link.ProjectID]
	if !ok {
		return Project{}, "", ErrShareInvalid
	}
	current := roleLocked(p, user, false)
	if !current.AtLeast(link.Role) {
		if p.Members == nil {
			p.Members = map[string]Role{}
		}
		p.Members[user] = link.Role
		current = link.Role
	}
	link.Uses++
	if err := r.saveLocked(); err != nil {
		return Project{}, "", err
	}
	return copyProject(p), current, nil
}
