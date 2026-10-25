package projects

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	r, err := Open(filepath.Join(t.TempDir(), "projects.json"))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRoleOrdering(t *testing.T) {
	if !RoleOwner.AtLeast(RoleEditor) || !RoleEditor.AtLeast(RoleCommenter) || !RoleCommenter.AtLeast(RoleViewer) {
		t.Fatal("role ranks out of order")
	}
	if RoleViewer.AtLeast(RoleEditor) {
		t.Fatal("viewer must not satisfy editor")
	}
	if Role("").AtLeast(RoleViewer) {
		t.Fatal("no role must not satisfy viewer")
	}
	if _, err := ParseRole("bogus"); err == nil {
		t.Fatal("ParseRole accepted bogus role")
	}
}

func TestCreateListAndRoles(t *testing.T) {
	r := newTestRegistry(t)
	dir := t.TempDir()
	p, err := r.Create("Sách 1", dir, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if p.ID == "" || p.Owner != "alice" {
		t.Fatalf("bad project %+v", p)
	}
	if _, err := r.Create("dup", dir, "bob"); !errors.Is(err, ErrDirTaken) {
		t.Fatalf("expected ErrDirTaken, got %v", err)
	}

	if got := r.RoleOf(p.ID, "alice", false); got != RoleOwner {
		t.Fatalf("owner role = %q", got)
	}
	if got := r.RoleOf(p.ID, "bob", false); got != "" {
		t.Fatalf("stranger role = %q", got)
	}
	if got := r.RoleOf(p.ID, "root", true); got != RoleOwner {
		t.Fatalf("admin role = %q", got)
	}

	if err := r.SetMember(p.ID, "bob", RoleViewer); err != nil {
		t.Fatal(err)
	}
	if got := r.RoleOf(p.ID, "bob", false); got != RoleViewer {
		t.Fatalf("bob role = %q", got)
	}
	if err := r.SetMember(p.ID, "bob", RoleOwner); err == nil {
		t.Fatal("SetMember must not grant owner")
	}
	if err := r.RemoveMember(p.ID, "alice"); !errors.Is(err, ErrOwner) {
		t.Fatalf("removing owner: %v", err)
	}

	if l := r.ListFor("bob", false); len(l) != 1 || l[0].Role != RoleViewer {
		t.Fatalf("bob list %+v", l)
	}
	if l := r.ListFor("carol", false); len(l) != 0 {
		t.Fatalf("carol sees %+v", l)
	}
	if l := r.ListFor("carol", true); len(l) != 1 {
		t.Fatalf("admin list %+v", l)
	}

	if got, ok := r.ByDir(dir); !ok || got.ID != p.ID {
		t.Fatal("ByDir lookup failed")
	}

	// Persistence.
	r2, err := Open(r.path)
	if err != nil {
		t.Fatal(err)
	}
	if got := r2.RoleOf(p.ID, "bob", false); got != RoleViewer {
		t.Fatalf("reloaded bob role = %q", got)
	}
}

func TestShareLinkJoinExpireRevoke(t *testing.T) {
	r := newTestRegistry(t)
	p, _ := r.Create("P", t.TempDir(), "alice")
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

	token, link, err := r.CreateShare(p.ID, RoleCommenter, now.Add(24*time.Hour), "alice", now)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || link.ID == "" {
		t.Fatal("empty token/link id")
	}
	for _, l := range r.ListShares(p.ID) {
		if l.TokenHash == token {
			t.Fatal("raw token must not be stored")
		}
	}

	got, role, err := r.Join(token, "bob", now)
	if err != nil || got.ID != p.ID || role != RoleCommenter {
		t.Fatalf("join: %v %v %v", got.ID, role, err)
	}
	// Joining never downgrades an existing higher role.
	_ = r.SetMember(p.ID, "dave", RoleEditor)
	if _, role, _ := r.Join(token, "dave", now); role != RoleEditor {
		t.Fatalf("join downgraded dave to %q", role)
	}
	// Owner keeps owner.
	if _, role, _ := r.Join(token, "alice", now); role != RoleOwner {
		t.Fatalf("join changed owner to %q", role)
	}

	if _, _, err := r.Join(token, "eve", now.Add(48*time.Hour)); !errors.Is(err, ErrShareInvalid) {
		t.Fatalf("expired link: %v", err)
	}
	if _, _, err := r.Join("nope", "eve", now); !errors.Is(err, ErrShareInvalid) {
		t.Fatalf("unknown link: %v", err)
	}

	if _, _, err := r.CreateShare(p.ID, RoleOwner, time.Time{}, "alice", now); err == nil {
		t.Fatal("owner share link must be rejected")
	}

	tok2, l2, _ := r.CreateShare(p.ID, RoleViewer, time.Time{}, "alice", now)
	if err := r.RevokeShare(p.ID, l2.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.Join(tok2, "eve", now); !errors.Is(err, ErrShareInvalid) {
		t.Fatalf("revoked link: %v", err)
	}
}

func TestCoveringAndContainsAny(t *testing.T) {
	r := newTestRegistry(t)
	base := t.TempDir()
	book := filepath.Join(base, "book")
	p, _ := r.Create("Book", book, "alice")

	if got, ok := r.Covering(filepath.Join(book, "chapters", "one")); !ok || got.ID != p.ID {
		t.Fatal("child folder not covered by its project")
	}
	if got, ok := r.Covering(book); !ok || got.ID != p.ID {
		t.Fatal("project folder not covered by itself")
	}
	if _, ok := r.Covering(base); ok {
		t.Fatal("parent folder must not be covered")
	}
	if _, ok := r.Covering(book + "-other"); ok {
		t.Fatal("sibling with shared prefix must not be covered")
	}
	if !r.ContainsAny(base) {
		t.Fatal("parent folder should contain the project")
	}
	if r.ContainsAny(book) || r.ContainsAny(filepath.Join(book, "x")) {
		t.Fatal("ContainsAny must only report projects strictly inside")
	}
}

func TestRegisterAndRename(t *testing.T) {
	r := newTestRegistry(t)
	dir := t.TempDir()
	p, err := r.Create("", dir, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != filepath.Base(dir) {
		t.Fatalf("default name %q", p.Name)
	}
	if err := r.Delete(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Get(p.ID); ok {
		t.Fatal("project still present after Delete")
	}
}
