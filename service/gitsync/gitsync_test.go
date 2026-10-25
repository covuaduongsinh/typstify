package gitsync

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// tempDir is removed best-effort: on Windows go-git can still hold pack
// files open when the test ends, failing t.TempDir's strict cleanup.
func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "gitsync-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func put(t *testing.T, dir, rel, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func get(t *testing.T, dir, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func setup(t *testing.T) (*Service, string) {
	t.Helper()
	remote := tempDir(t)
	if _, err := git.PlainInit(remote, true); err != nil {
		t.Fatal(err)
	}
	return &Service{Base: tempDir(t), AllowLocal: true}, remote
}

func TestCommitPushPullAndDivergence(t *testing.T) {
	ctx := context.Background()
	s, remote := setup(t)
	a, b := tempDir(t), tempDir(t)
	for _, k := range []string{"a", "b"} {
		if err := s.SaveConfig(k, Config{Remote: remote, Branch: "main", AuthorName: "Typstify"}); err != nil {
			t.Fatal(err)
		}
	}

	// A creates the repository and publishes it.
	put(t, a, "main.typ", "v1")
	if _, err := s.Commit("a", a, "v1", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Push(ctx, "a", a, false); err != nil {
		t.Fatalf("push: %v", err)
	}
	if _, err := os.Stat(filepath.Join(a, ".git", "info", "exclude")); err != nil {
		t.Fatal("exclude file not written")
	}
	if _, err := s.Commit("a", a, "again", "", ""); !errors.Is(err, ErrNothingToCommit) {
		t.Fatalf("want ErrNothingToCommit, got %v", err)
	}

	// B starts empty and pulls.
	res, err := s.Pull(ctx, "b", b, false)
	if err != nil || !res.Updated || get(t, b, "main.typ") != "v1" {
		t.Fatalf("first pull %+v %v", res, err)
	}

	// Fast-forward.
	put(t, a, "main.typ", "v2")
	if _, err := s.Commit("a", a, "v2", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Push(ctx, "a", a, false); err != nil {
		t.Fatal(err)
	}
	st, err := s.Status(ctx, "b", b, true)
	if err != nil || st.Behind != 1 || st.Ahead != 0 || st.Diverged {
		t.Fatalf("status before pull %+v %v", st, err)
	}
	if _, err := s.Pull(ctx, "b", b, false); err != nil || get(t, b, "main.typ") != "v2" {
		t.Fatalf("fast-forward: %v", err)
	}

	// Uncommitted local changes block a pull.
	put(t, a, "main.typ", "v3")
	_, _ = s.Commit("a", a, "v3", "", "")
	_ = s.Push(ctx, "a", a, false)
	put(t, b, "main.typ", "dirty")
	if _, err := s.Pull(ctx, "b", b, false); !errors.Is(err, ErrDirty) {
		t.Fatalf("want ErrDirty, got %v", err)
	}

	// Both sides committed: no silent merge.
	if _, err := s.Commit("b", b, "b local", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Pull(ctx, "b", b, false); !errors.Is(err, ErrDiverged) {
		t.Fatalf("want ErrDiverged on pull, got %v", err)
	}
	if err := s.Push(ctx, "b", b, false); !errors.Is(err, ErrDiverged) {
		t.Fatalf("want ErrDiverged on push, got %v", err)
	}
	st, _ = s.Status(ctx, "b", b, true)
	if !st.Diverged {
		t.Fatalf("status should report divergence: %+v", st)
	}
	// The user takes the remote side.
	if _, err := s.Pull(ctx, "b", b, true); err != nil || get(t, b, "main.typ") != "v3" {
		t.Fatalf("take remote: %v, content %q", err, get(t, b, "main.typ"))
	}
	st, _ = s.Status(ctx, "b", b, true)
	if st.Ahead != 0 || st.Behind != 0 || len(st.Changes) != 0 {
		t.Fatalf("after taking remote %+v", st)
	}
}

func TestForcePushKeepsLocalSide(t *testing.T) {
	ctx := context.Background()
	s, remote := setup(t)
	a, b := tempDir(t), tempDir(t)
	_ = s.SaveConfig("a", Config{Remote: remote, Branch: "main"})
	_ = s.SaveConfig("b", Config{Remote: remote, Branch: "main"})
	put(t, a, "main.typ", "base")
	_, _ = s.Commit("a", a, "base", "x", "")
	_ = s.Push(ctx, "a", a, false)
	_, _ = s.Pull(ctx, "b", b, false)

	put(t, a, "main.typ", "from a")
	_, _ = s.Commit("a", a, "a", "x", "")
	_ = s.Push(ctx, "a", a, false)
	put(t, b, "main.typ", "from b")
	_, _ = s.Commit("b", b, "b", "x", "")

	if err := s.Push(ctx, "b", b, true); err != nil {
		t.Fatalf("force push: %v", err)
	}
	if _, err := s.Pull(ctx, "a", a, true); err != nil || get(t, a, "main.typ") != "from b" {
		t.Fatalf("a should now have b's version: %v %q", err, get(t, a, "main.typ"))
	}
}

func TestConfigValidationAndToken(t *testing.T) {
	s := &Service{Base: tempDir(t)}
	for _, bad := range []string{"http://example.com/x.git", "https://user:pw@example.com/x.git", "ftp://example.com/x", "/srv/repo", "not a url"} {
		if err := s.SaveConfig("p", Config{Remote: bad}); !errors.Is(err, ErrBadRemote) {
			t.Errorf("%q accepted", bad)
		}
	}
	if err := s.SaveConfig("p", Config{Remote: "https://example.com/x.git", Token: "secret"}); err != nil {
		t.Fatal(err)
	}
	// An empty token keeps the stored one.
	if err := s.SaveConfig("p", Config{Remote: "https://example.com/y.git"}); err != nil {
		t.Fatal(err)
	}
	c, _ := s.LoadConfig("p")
	if c.Token != "secret" || c.Remote != "https://example.com/y.git" {
		t.Fatalf("config %+v", c)
	}
	info, _ := os.Stat(s.configPath("p"))
	if info.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/' {
		t.Fatalf("config readable by others: %v", info.Mode())
	}
}

func TestPrivateAddressesAreBlocked(t *testing.T) {
	s := &Service{Base: tempDir(t)}
	dir := tempDir(t)
	if err := s.SaveConfig("p", Config{Remote: "https://127.0.0.1:9/repo.git", Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	put(t, dir, "main.typ", "x")
	if _, err := s.Commit("p", dir, "x", "x", ""); err != nil {
		t.Fatal(err)
	}
	err := s.Push(context.Background(), "p", dir, false)
	if err == nil || !strings.Contains(err.Error(), "bị chặn") {
		t.Fatalf("push to loopback should be refused at dial time, got %v", err)
	}
}

func TestCleanStatusHasEmptyChangeList(t *testing.T) {
	s, remote := setup(t)
	dir := tempDir(t)
	_ = s.SaveConfig("p", Config{Remote: remote})
	put(t, dir, "main.typ", "x")
	if _, err := s.Commit("p", dir, "x", "x", ""); err != nil {
		t.Fatal(err)
	}
	st, err := s.Status(context.Background(), "p", dir, false)
	if err != nil || st.Changes == nil || len(st.Changes) != 0 {
		t.Fatalf("changes must be an empty list, got %#v (%v)", st.Changes, err)
	}
}

func TestCommitHonoursIgnoreRules(t *testing.T) {
	s, remote := setup(t)
	dir := tempDir(t)
	_ = s.SaveConfig("p", Config{Remote: remote})
	put(t, dir, "main.typ", "x")
	put(t, dir, ".gitignore", "*.pdf\n")
	put(t, dir, "out.pdf", "binary")
	_ = os.MkdirAll(filepath.Join(dir, ".typstify"), 0o755)
	put(t, dir, ".typstify/settings.json", "{}")
	put(t, dir, ".live_preview_1.typ", "scratch")
	if _, err := s.Commit("p", dir, "x", "x", ""); err != nil {
		t.Fatal(err)
	}
	repo, _ := git.PlainOpen(dir)
	head, _ := repo.Head()
	c, _ := repo.CommitObject(head.Hash())
	files, _ := c.Files()
	var names []string
	_ = files.ForEach(func(f *object.File) error { names = append(names, f.Name); return nil })
	if strings.Join(names, ",") != ".gitignore,main.typ" {
		t.Fatalf("committed %v", names)
	}
}

// Review finding: a pull used a hard reset, deleting untracked and
// ignored files that the version history does not keep either.
func TestPullKeepsUntrackedAndIgnoredFiles(t *testing.T) {
	ctx := context.Background()
	s, remote := setup(t)
	a, b := tempDir(t), tempDir(t)
	_ = s.SaveConfig("a", Config{Remote: remote, Branch: "main"})
	_ = s.SaveConfig("b", Config{Remote: remote, Branch: "main"})
	put(t, a, "main.typ", "v1")
	put(t, a, ".gitignore", "*.pdf\n")
	_, _ = s.Commit("a", a, "v1", "x", "")
	_ = s.Push(ctx, "a", a, false)

	// B already has files of its own before its first pull.
	put(t, b, "notes.txt", "mine")
	_ = os.MkdirAll(filepath.Join(b, ".typstify"), 0o755)
	put(t, b, ".typstify/settings.json", "{}")
	if _, err := s.Pull(ctx, "b", b, true); err != nil {
		t.Fatal(err)
	}
	put(t, b, "out.pdf", "build output")

	put(t, a, "main.typ", "v2")
	put(t, a, "chap.typ", "new")
	_, _ = s.Commit("a", a, "v2", "x", "")
	_ = s.Push(ctx, "a", a, false)
	if _, err := s.Pull(ctx, "b", b, false); err != nil {
		t.Fatalf("fast-forward: %v", err)
	}
	if get(t, b, "main.typ") != "v2" || get(t, b, "chap.typ") != "new" {
		t.Fatal("remote changes not applied")
	}
	for _, f := range []string{"notes.txt", ".typstify/settings.json", "out.pdf"} {
		if _, err := os.Stat(filepath.Join(b, filepath.FromSlash(f))); err != nil {
			t.Fatalf("%s was deleted by the pull", f)
		}
	}

	// A file removed upstream is removed here too.
	_ = os.Remove(filepath.Join(a, "chap.typ"))
	_, _ = s.Commit("a", a, "rm", "x", "")
	_ = s.Push(ctx, "a", a, false)
	if _, err := s.Pull(ctx, "b", b, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(b, "chap.typ")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("file deleted upstream should be removed")
	}
	st, _ := s.Status(ctx, "b", b, false)
	for _, c := range st.Changes {
		if c.Path == "main.typ" {
			t.Fatalf("index out of date after pull: %+v", st.Changes)
		}
	}
}

func TestPullSkipsSymlinksFromRemote(t *testing.T) {
	ctx := context.Background()
	s, remote := setup(t)
	a, b := tempDir(t), tempDir(t)
	_ = s.SaveConfig("a", Config{Remote: remote, Branch: "main"})
	_ = s.SaveConfig("b", Config{Remote: remote, Branch: "main"})
	put(t, a, "main.typ", "x")
	if err := os.Symlink("/etc/passwd", filepath.Join(a, "leak")); err != nil {
		t.Skipf("symlinks unavailable here: %v", err)
	}
	_, _ = s.Commit("a", a, "x", "x", "")
	_ = s.Push(ctx, "a", a, false)
	if _, err := s.Pull(ctx, "b", b, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(b, "leak")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a symlink from the remote must not be created")
	}
}

func TestTokenStaysWithItsHost(t *testing.T) {
	s := &Service{Base: tempDir(t)}
	_ = s.SaveConfig("p", Config{Remote: "https://github.com/a/b.git", Token: "pat"})
	_ = s.SaveConfig("p", Config{Remote: "https://github.com/a/c.git"})
	if c, _ := s.LoadConfig("p"); c.Token != "pat" {
		t.Fatal("same host keeps the token")
	}
	_ = s.SaveConfig("p", Config{Remote: "https://evil.example/x.git"})
	if c, _ := s.LoadConfig("p"); c.Token != "" {
		t.Fatal("a token must not follow the remote to another host")
	}
	_ = s.SaveConfig("p", Config{Remote: "https://github.com/a/b.git", Token: "pat2"})
	_ = s.SaveConfig("p", Config{Remote: "https://github.com/a/b.git", ClearToken: true})
	if c, _ := s.LoadConfig("p"); c.Token != "" {
		t.Fatal("clearToken should forget it")
	}
	if err := s.SaveConfig("p", Config{Remote: "https://github.com/a/b.git", Branch: "bad..name"}); !errors.Is(err, ErrBadBranch) {
		t.Fatalf("want ErrBadBranch, got %v", err)
	}
}

func TestBlockedAddresses(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "10.1.2.3", "192.168.1.1", "169.254.169.254", "100.100.100.200", "::1", "64:ff9b::a00:1"} {
		if !blockedIP(net.ParseIP(ip)) {
			t.Errorf("%s should be blocked", ip)
		}
	}
	for _, ip := range []string{"140.82.112.3", "2606:50c0:8000::153"} {
		if blockedIP(net.ParseIP(ip)) {
			t.Errorf("%s is public", ip)
		}
	}
}
