package server

import (
	"errors"
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

	if joinedAbs != rootAbs && !strings.HasPrefix(joinedAbs, rootAbs+string(filepath.Separator)) {
		return "", errPathEscapesRoot
	}

	return joinedAbs, nil
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
	return abs == rootAbs || strings.HasPrefix(abs, rootAbs+string(filepath.Separator))
}
