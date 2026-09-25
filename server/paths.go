package server

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var errPathEscapesRoot = errors.New("path escapes project root")

// resolveInRoot cleans rel and joins it onto root, rejecting any path that
// would climb outside of root (e.g. via "../../etc/passwd"). rel is treated
// as slash-separated and relative to root regardless of leading slashes.
func resolveInRoot(root, rel string) (string, error) {
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "/")
	// Cleaning "/"+rel first collapses any ".." components against a fake
	// root, so they can't climb past the real root once joined below.
	cleaned := filepath.Clean("/" + rel)

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	joinedAbs := filepath.Join(rootAbs, cleaned)

	if !within(rootAbs, joinedAbs) {
		return "", errPathEscapesRoot
	}

	// The lexical check above can't see symlinks: a link inside the project
	// (which the AI agent's shell can create) pointing at /etc or another
	// project would let reads, writes and deletes escape. Compare the
	// resolved locations too.
	if !within(realPath(rootAbs), realPath(joinedAbs)) {
		return "", errPathEscapesRoot
	}

	return joinedAbs, nil
}

// within reports whether p is root or underneath it (both absolute, clean).
func within(root, p string) bool {
	return p == root || strings.HasPrefix(p, root+string(filepath.Separator))
}

// realPath resolves symlinks in p. For a path that doesn't exist yet (a
// file about to be created) it resolves the longest existing ancestor and
// re-appends the rest, so a symlinked parent directory is still caught.
func realPath(p string) string {
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	var rest []string
	cur := p
	for {
		parent := filepath.Dir(cur)
		rest = append([]string{filepath.Base(cur)}, rest...)
		if parent == cur {
			return p
		}
		if real, err := filepath.EvalSymlinks(parent); err == nil {
			return filepath.Join(append([]string{real}, rest...)...)
		} else if !os.IsNotExist(err) {
			return p
		}
		cur = parent
	}
}

// relPath returns p relative to root using forward slashes, suitable for
// JSON responses sent to the browser.
func relPath(root, p string) (string, error) {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(r), nil
}

// isUnderRoot reports whether the already-absolute path abs is root itself or
// somewhere underneath it. Unlike resolveInRoot, abs is not joined onto root
// -- it's an independent absolute path being checked for containment (see
// handleOpenProject/handleCreateProject's ProjectRoot guard).
func isUnderRoot(root, abs string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	return within(rootAbs, abs) && within(realPath(rootAbs), realPath(abs))
}
