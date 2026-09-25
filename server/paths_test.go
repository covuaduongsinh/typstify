package server

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveInRootRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"../etc/passwd", "a/../../x", "/../../x"} {
		p, err := resolveInRoot(root, rel)
		if err == nil && !within(root, p) {
			t.Errorf("%q resolved outside root to %s", rel, p)
		}
	}
	p, err := resolveInRoot(root, "chapters/ch1.typ")
	if err != nil || p != filepath.Join(root, "chapters", "ch1.typ") {
		t.Errorf("normal path: %q, %v", p, err)
	}
}

func TestResolveInRootRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skip("symlinks unsupported:", err)
	}

	for _, rel := range []string{"link", "link/secret.txt", "link/new-file.typ", "link/newdir/x.typ"} {
		if _, err := resolveInRoot(root, rel); !errors.Is(err, errPathEscapesRoot) {
			t.Errorf("%q through a symlink out of the root: err = %v, want errPathEscapesRoot", rel, err)
		}
	}
}

func TestResolveInRootAllowsSymlinkedRoot(t *testing.T) {
	real := t.TempDir()
	parent := t.TempDir()
	root := filepath.Join(parent, "project")
	if err := os.Symlink(real, root); err != nil {
		t.Skip("symlinks unsupported:", err)
	}
	if _, err := resolveInRoot(root, "main.typ"); err != nil {
		t.Errorf("file in a project whose root is a symlink rejected: %v", err)
	}
}

func TestIsProtectedPath(t *testing.T) {
	root := t.TempDir()
	cases := map[string]bool{
		".git":          true,
		".git/config":   true,
		".typstify":     true,
		"chapters/.git": false,
		"main.typ":      false,
		".gitignore":    false,
	}
	for rel, want := range cases {
		if got := isProtectedPath(root, filepath.Join(root, rel)); got != want {
			t.Errorf("isProtectedPath(%q) = %v, want %v", rel, got, want)
		}
	}
}
