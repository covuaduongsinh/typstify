// Package gitsync connects a project folder to a Git remote (GitHub,
// GitLab, Gitea…) with go-git, so no git binary is needed on the server.
// It commits, pushes and pulls; merges are not attempted: when both sides
// have new commits the user chooses which side wins, and the server's
// version history (service/history) keeps the other one recoverable.
package gitsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/client"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

var (
	ErrNoRemote        = errors.New("chưa cấu hình kho Git")
	ErrBadRemote       = errors.New("địa chỉ kho phải là https://… và không chứa mật khẩu")
	ErrNothingToCommit = errors.New("không có thay đổi để commit")
	ErrDirty           = errors.New("còn thay đổi chưa commit: hãy commit trước khi kéo về")
	ErrDiverged        = errors.New("cả kho và máy chủ đều có commit mới")
	ErrNotRepo         = errors.New("dự án chưa là kho Git")
	ErrBadBranch       = errors.New("tên nhánh không hợp lệ")
)

// excluded is written to .git/info/exclude (never committed): editor
// state and preview scratch files.
const excluded = ".typstify/\n.live_preview_*\n"

// Config is a project's remote settings. Token never leaves the server.
type Config struct {
	Remote      string `json:"remote"`
	Branch      string `json:"branch"`
	Username    string `json:"username"`
	Token       string `json:"token,omitempty"`
	AuthorName  string `json:"authorName"`
	AuthorEmail string `json:"authorEmail"`
	// ClearToken (request only) forgets the stored token.
	ClearToken bool `json:"clearToken,omitempty"`
}

type FileChange struct {
	Path   string `json:"path"`
	Status string `json:"status"` // added, modified, deleted, untracked, renamed
}

type CommitInfo struct {
	Hash    string    `json:"hash"`
	Message string    `json:"message"`
	Author  string    `json:"author"`
	When    time.Time `json:"when"`
}

type Status struct {
	Initialized bool         `json:"initialized"`
	Branch      string       `json:"branch"`
	Remote      string       `json:"remote"`
	HasToken    bool         `json:"hasToken"`
	Changes     []FileChange `json:"changes"`
	Head        *CommitInfo  `json:"head,omitempty"`
	RemoteHead  *CommitInfo  `json:"remoteHead,omitempty"`
	Ahead       int          `json:"ahead"`
	Behind      int          `json:"behind"`
	Diverged    bool         `json:"diverged"`
	// FetchError is set when the remote could not be reached; the rest of
	// the status still describes the local repository.
	FetchError string `json:"fetchError,omitempty"`
}

// Service runs Git operations; one at a time per server, since they move
// files in the project.
type Service struct {
	Base string
	// AllowPrivate lets remotes resolve to private or loopback addresses
	// (a self-hosted Gitea on the LAN). Off by default: otherwise any
	// project owner could make the server probe internal services.
	AllowPrivate bool
	// AllowLocal accepts file paths as remotes (tests only).
	AllowLocal bool

	mu       sync.Mutex             // guards locks
	locks    map[string]*sync.Mutex // one per project: operations move files
	initOnce sync.Once
}

// lock serializes operations on one project; other projects proceed.
func (s *Service) lock(key string) func() {
	s.mu.Lock()
	if s.locks == nil {
		s.locks = map[string]*sync.Mutex{}
	}
	m, ok := s.locks[key]
	if !ok {
		m = &sync.Mutex{}
		s.locks[key] = m
	}
	s.mu.Unlock()
	m.Lock()
	return m.Unlock
}

var errBlockedAddr = errors.New("địa chỉ nội bộ bị chặn")

// sharedNets are not private in the RFC 1918 sense but still reach
// infrastructure: carrier-grade NAT (Tailscale, some cloud metadata) and
// NAT64.
var sharedNets = func() []*net.IPNet {
	var out []*net.IPNet
	for _, cidr := range []string{"100.64.0.0/10", "64:ff9b::/96", "64:ff9b:1::/48"} {
		_, n, _ := net.ParseCIDR(cidr)
		out = append(out, n)
	}
	return out
}()

func blockedIP(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	for _, n := range sharedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// guardedTransport refuses connections to loopback, private, link-local
// and unspecified addresses at dial time, so DNS tricks cannot bypass it.
func guardedTransport(allowPrivate bool) *http.Transport {
	d := &net.Dialer{
		Timeout: 30 * time.Second,
		Control: func(network, address string, c syscall.RawConn) error {
			if allowPrivate {
				return nil
			}
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			if blockedIP(net.ParseIP(host)) {
				return errBlockedAddr
			}
			return nil
		},
	}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = nil
	t.DialContext = d.DialContext
	return t
}

func (s *Service) setup() {
	s.initOnce.Do(func() {
		hc := &http.Client{Transport: guardedTransport(s.AllowPrivate), Timeout: 5 * time.Minute}
		client.InstallProtocol("https", githttp.NewClient(hc))
		client.InstallProtocol("http", githttp.NewClient(hc))
	})
}

// ValidateRemote accepts https URLs without embedded credentials (http
// too when private remotes are allowed).
func (s *Service) ValidateRemote(raw string) error {
	if s.AllowLocal && (filepath.IsAbs(raw) || strings.HasPrefix(raw, "file://")) {
		return nil
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || u.User != nil {
		return ErrBadRemote
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && s.AllowPrivate) {
		return ErrBadRemote
	}
	return nil
}

func (s *Service) configPath(key string) string {
	return filepath.Join(s.Base, safeKey(key)+".json")
}

func safeKey(key string) string {
	var b strings.Builder
	for _, r := range key {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

func (s *Service) LoadConfig(key string) (Config, error) {
	var c Config
	data, err := os.ReadFile(s.configPath(key))
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	return c, json.Unmarshal(data, &c)
}

func hostOf(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return strings.ToLower(u.Scheme + "://" + u.Host)
	}
	return raw
}

// SaveConfig stores the settings. An empty Token keeps the stored one,
// but only for the same host: a token is never sent to another server.
func (s *Service) SaveConfig(key string, c Config) error {
	if err := s.ValidateRemote(c.Remote); err != nil {
		return err
	}
	if c.Branch != "" && plumbing.NewBranchReferenceName(c.Branch).Validate() != nil {
		return ErrBadBranch
	}
	if c.Token == "" && !c.ClearToken {
		old, _ := s.LoadConfig(key)
		if hostOf(old.Remote) == hostOf(c.Remote) {
			c.Token = old.Token
		}
	}
	c.ClearToken = false
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.Base, 0o700); err != nil {
		return err
	}
	tmp := s.configPath(key) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.configPath(key))
}

func (c Config) auth() transport.AuthMethod {
	if c.Token == "" {
		return nil
	}
	user := c.Username
	if user == "" {
		user = "git"
	}
	return &githttp.BasicAuth{Username: user, Password: c.Token}
}

func (c Config) branch(repo *git.Repository) string {
	if c.Branch != "" {
		return c.Branch
	}
	if head, err := repo.Head(); err == nil && head.Name().IsBranch() {
		return head.Name().Short()
	}
	return "main"
}

// open returns the project's repository, creating it when create is set.
func (s *Service) open(dir string, c Config, create bool) (*git.Repository, error) {
	repo, err := git.PlainOpen(dir)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		if !create {
			return nil, ErrNotRepo
		}
		repo, err = git.PlainInit(dir, false)
		if err != nil {
			return nil, err
		}
		branch := c.Branch
		if branch == "" {
			branch = "main"
		}
		head := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName(branch))
		if err := repo.Storer.SetReference(head); err != nil {
			return nil, err
		}
		_ = os.MkdirAll(filepath.Join(dir, ".git", "info"), 0o755)
		_ = os.WriteFile(filepath.Join(dir, ".git", "info", "exclude"), []byte(excluded), 0o644)
	}
	if err != nil {
		return nil, err
	}
	if c.Remote != "" {
		rc, err := repo.Remote("origin")
		if err != nil || len(rc.Config().URLs) == 0 || rc.Config().URLs[0] != c.Remote {
			_ = repo.DeleteRemote("origin")
			if _, err := repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{c.Remote}}); err != nil {
				return nil, err
			}
		}
	}
	return repo, nil
}

func info(c *object.Commit) *CommitInfo {
	msg := strings.TrimSpace(c.Message)
	if i := strings.IndexByte(msg, '\n'); i >= 0 {
		msg = msg[:i]
	}
	return &CommitInfo{Hash: c.Hash.String(), Message: msg, Author: c.Author.Name, When: c.Author.When}
}

// worktree returns the repository's worktree with its ignore rules
// loaded: go-git's Add(All) consults only Worktree.Excludes, not the
// project's .gitignore files nor .git/info/exclude.
func worktree(repo *git.Repository) (*git.Worktree, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	patterns, _ := gitignore.ReadPatterns(wt.Filesystem, nil)
	for _, line := range strings.Fields(excluded) {
		patterns = append(patterns, gitignore.ParsePattern(line, nil))
	}
	wt.Excludes = patterns
	return wt, nil
}

func changes(repo *git.Repository) ([]FileChange, error) {
	wt, err := worktree(repo)
	if err != nil {
		return nil, err
	}
	st, err := wt.Status()
	if err != nil {
		return nil, err
	}
	out := []FileChange{} // never null in JSON
	for p, fs := range st {
		code := fs.Worktree
		if code == git.Unmodified {
			code = fs.Staging
		}
		var status string
		switch code {
		case git.Untracked:
			status = "untracked"
		case git.Added:
			status = "added"
		case git.Modified:
			status = "modified"
		case git.Deleted:
			status = "deleted"
		case git.Renamed, git.Copied:
			status = "renamed"
		default:
			continue
		}
		out = append(out, FileChange{Path: p, Status: status})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// ancestors collects up to limit commits reachable from h.
func ancestors(repo *git.Repository, h plumbing.Hash, limit int) map[plumbing.Hash]bool {
	seen := map[plumbing.Hash]bool{}
	queue := []plumbing.Hash{h}
	for len(queue) > 0 && len(seen) < limit {
		cur := queue[0]
		queue = queue[1:]
		if seen[cur] {
			continue
		}
		seen[cur] = true
		c, err := repo.CommitObject(cur)
		if err != nil {
			continue
		}
		queue = append(queue, c.ParentHashes...)
	}
	return seen
}

const graphLimit = 20000

// compare counts commits only on each side.
func compare(repo *git.Repository, local, remote plumbing.Hash) (ahead, behind int) {
	if local == remote {
		return 0, 0
	}
	l := ancestors(repo, local, graphLimit)
	r := ancestors(repo, remote, graphLimit)
	for h := range l {
		if !r[h] {
			ahead++
		}
	}
	for h := range r {
		if !l[h] {
			behind++
		}
	}
	return ahead, behind
}

func (s *Service) fetch(ctx context.Context, repo *git.Repository, c Config, branch string) error {
	spec := config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", branch, branch))
	err := repo.FetchContext(ctx, &git.FetchOptions{RemoteName: "origin", Auth: c.auth(), RefSpecs: []config.RefSpec{spec}})
	switch {
	case err == nil, errors.Is(err, git.NoErrAlreadyUpToDate), errors.Is(err, transport.ErrEmptyRemoteRepository):
		return nil
	default:
		return err
	}
}

func refHash(repo *git.Repository, name plumbing.ReferenceName) (plumbing.Hash, bool) {
	ref, err := repo.Reference(name, true)
	if err != nil {
		return plumbing.ZeroHash, false
	}
	return ref.Hash(), true
}

// Status describes the repository; fetch first contacts the remote so
// ahead/behind are current.
func (s *Service) Status(ctx context.Context, key, dir string, fetch bool) (Status, error) {
	s.setup()
	defer s.lock(key)()
	c, err := s.LoadConfig(key)
	if err != nil {
		return Status{}, err
	}
	st := Status{Remote: c.Remote, HasToken: c.Token != "", Changes: []FileChange{}}
	repo, err := s.open(dir, c, false)
	if errors.Is(err, ErrNotRepo) {
		st.Branch = c.Branch
		return st, nil
	}
	if err != nil {
		return st, err
	}
	st.Initialized = true
	st.Branch = c.branch(repo)
	if st.Changes, err = changes(repo); err != nil {
		return st, err
	}
	if fetch && c.Remote != "" {
		if err := s.fetch(ctx, repo, c, st.Branch); err != nil {
			st.FetchError = err.Error()
		}
	}
	local, hasLocal := refHash(repo, plumbing.HEAD)
	if hasLocal {
		if cm, err := repo.CommitObject(local); err == nil {
			st.Head = info(cm)
		}
	}
	remote, hasRemote := refHash(repo, plumbing.NewRemoteReferenceName("origin", st.Branch))
	if hasRemote {
		if cm, err := repo.CommitObject(remote); err == nil {
			st.RemoteHead = info(cm)
		}
	}
	if hasLocal && hasRemote {
		st.Ahead, st.Behind = compare(repo, local, remote)
		st.Diverged = st.Ahead > 0 && st.Behind > 0
	} else if hasLocal {
		st.Ahead = len(ancestors(repo, local, graphLimit))
	}
	return st, nil
}

// Commit records every change in the project (creating the repository
// on first use).
func (s *Service) Commit(key, dir, message, authorName, authorEmail string) (*CommitInfo, error) {
	s.setup()
	defer s.lock(key)()
	c, err := s.LoadConfig(key)
	if err != nil {
		return nil, err
	}
	repo, err := s.open(dir, c, true)
	if err != nil {
		return nil, err
	}
	pending, err := changes(repo)
	if err != nil {
		return nil, err
	}
	if len(pending) == 0 {
		return nil, ErrNothingToCommit
	}
	wt, err := worktree(repo)
	if err != nil {
		return nil, err
	}
	if err := wt.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return nil, err
	}
	if c.AuthorName != "" {
		authorName, authorEmail = c.AuthorName, c.AuthorEmail
	}
	if authorEmail == "" {
		authorEmail = "typstify@localhost"
	}
	h, err := wt.Commit(strings.TrimSpace(message), &git.CommitOptions{
		Author: &object.Signature{Name: authorName, Email: authorEmail, When: time.Now()},
	})
	if err != nil {
		return nil, err
	}
	cm, err := repo.CommitObject(h)
	if err != nil {
		return nil, err
	}
	return info(cm), nil
}

// Push sends the branch to the remote; force overwrites the remote branch
// (when both sides changed and the user keeps theirs).
func (s *Service) Push(ctx context.Context, key, dir string, force bool) error {
	s.setup()
	defer s.lock(key)()
	c, err := s.LoadConfig(key)
	if err != nil {
		return err
	}
	if c.Remote == "" {
		return ErrNoRemote
	}
	repo, err := s.open(dir, c, false)
	if err != nil {
		return err
	}
	branch := c.branch(repo)
	spec := config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", branch, branch))
	if force {
		spec = "+" + spec
	}
	err = repo.PushContext(ctx, &git.PushOptions{RemoteName: "origin", Auth: c.auth(), RefSpecs: []config.RefSpec{spec}})
	if errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil
	}
	// go-git reports a rejected push with an unwrapped
	// fmt.Errorf("non-fast-forward update: <ref>") (remote.go), so the
	// message is the only signal.
	if err != nil && strings.Contains(err.Error(), "non-fast-forward") {
		return ErrDiverged
	}
	return err
}

// PullResult says what a pull did.
type PullResult struct {
	Updated bool   `json:"updated"`
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
}

// Pull brings the remote branch in. A fast-forward needs a clean working
// tree; when both sides have commits, takeRemote resets the project to
// the remote branch (the caller saves a history version first), otherwise
// ErrDiverged is returned.
func (s *Service) Pull(ctx context.Context, key, dir string, takeRemote bool) (PullResult, error) {
	s.setup()
	defer s.lock(key)()
	c, err := s.LoadConfig(key)
	if err != nil {
		return PullResult{}, err
	}
	if c.Remote == "" {
		return PullResult{}, ErrNoRemote
	}
	repo, err := s.open(dir, c, true)
	if err != nil {
		return PullResult{}, err
	}
	branch := c.branch(repo)
	if err := s.fetch(ctx, repo, c, branch); err != nil {
		return PullResult{}, err
	}
	remote, ok := refHash(repo, plumbing.NewRemoteReferenceName("origin", branch))
	if !ok {
		return PullResult{}, nil // empty remote: nothing to bring in
	}
	local, hasLocal := refHash(repo, plumbing.HEAD)
	if hasLocal && local == remote {
		return PullResult{}, nil
	}
	pending, err := changes(repo)
	if err != nil {
		return PullResult{}, err
	}
	if hasLocal {
		ahead, behind := compare(repo, local, remote)
		if behind == 0 {
			return PullResult{}, nil // only local commits
		}
		if ahead > 0 && !takeRemote {
			return PullResult{}, ErrDiverged
		}
	}
	if !takeRemote && blocksPull(pending, remoteTreeOf(repo, remote)) {
		return PullResult{}, ErrDirty
	}
	// Only the files that differ between the two commits are written or
	// removed. A hard reset would also delete every untracked file,
	// including ignored ones (.typstify/, build output) that no history
	// version keeps.
	var fromTree *object.Tree
	if hasLocal {
		if c, err := repo.CommitObject(local); err == nil {
			fromTree, _ = c.Tree()
		}
	}
	toCommit, err := repo.CommitObject(remote)
	if err != nil {
		return PullResult{}, err
	}
	toTree, err := toCommit.Tree()
	if err != nil {
		return PullResult{}, err
	}
	if err := applyTree(dir, fromTree, toTree); err != nil {
		return PullResult{}, err
	}
	branchRef := plumbing.NewBranchReferenceName(branch)
	if err := repo.Storer.SetReference(plumbing.NewHashReference(branchRef, remote)); err != nil {
		return PullResult{}, err
	}
	if err := repo.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, branchRef)); err != nil {
		return PullResult{}, err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return PullResult{}, err
	}
	// The index follows the new commit; the working tree was updated above.
	if err := wt.Reset(&git.ResetOptions{Commit: remote, Mode: git.MixedReset}); err != nil {
		return PullResult{}, err
	}
	res := PullResult{Updated: true, To: remote.String()}
	if hasLocal {
		res.From = local.String()
	}
	return res, nil
}

// safeTreePath accepts a path from a commit only if it stays inside the
// project and out of .git (any case: Windows and macOS folders are not
// case-sensitive).
func safeTreePath(name string) bool {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, ":") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") {
			return false
		}
	}
	return true
}

// applyTree turns the files of tree `from` into those of `to` in dir.
// Only regular files are written: symlinks and submodules from a remote
// are skipped, so a pull can never plant a link to files outside the
// project (typst would follow it).
func applyTree(dir string, from, to *object.Tree) error {
	if from == nil {
		from = &object.Tree{}
	}
	changes, err := object.DiffTree(from, to)
	if err != nil {
		return err
	}
	for _, ch := range changes {
		action, err := ch.Action()
		if err != nil {
			return err
		}
		if action == merkletrie.Delete {
			name := ch.From.Name
			if safeTreePath(name) {
				if err := os.Remove(filepath.Join(dir, filepath.FromSlash(name))); err != nil && !errors.Is(err, os.ErrNotExist) {
					return err
				}
			}
			continue
		}
		name := ch.To.Name
		if !safeTreePath(name) {
			continue
		}
		f, err := to.File(name)
		if err != nil {
			return err
		}
		if f.Mode != filemode.Regular && f.Mode != filemode.Executable && f.Mode != filemode.Deprecated {
			continue
		}
		content, err := f.Contents()
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		// Never write through an existing symlink.
		if info, err := os.Lstat(target); err == nil && info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(target); err != nil {
				return err
			}
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func remoteTreeOf(repo *git.Repository, h plumbing.Hash) *object.Tree {
	c, err := repo.CommitObject(h)
	if err != nil {
		return nil
	}
	t, _ := c.Tree()
	return t
}

// blocksPull reports local changes a pull would overwrite: any change to
// a tracked file, or an untracked file the incoming commit also has.
// Other untracked files are left alone, as git does.
func blocksPull(pending []FileChange, incoming *object.Tree) bool {
	for _, c := range pending {
		if c.Status != "untracked" {
			return true
		}
		if incoming != nil {
			if _, err := incoming.File(c.Path); err == nil {
				return true
			}
		}
	}
	return false
}
