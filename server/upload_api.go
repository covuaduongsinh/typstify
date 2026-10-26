package server

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// uploadBodyLimit bounds one uploaded file (images pasted into the editor,
// files dropped on the file tree).
const uploadBodyLimit = 25 << 20

// uploadName turns a browser-supplied file name into a safe one: base name
// only, letters/digits/._- kept (Vietnamese letters included), spaces to
// dashes, no leading dot, and an extension required. Names are NFC first:
// macOS and Safari send decomposed Vietnamese, whose combining marks are
// not letters and would be dropped.
func uploadName(raw string) (string, error) {
	base := filepath.Base(strings.ReplaceAll(norm.NFC.String(strings.TrimSpace(raw)), "\\", "/"))
	var b strings.Builder
	for _, r := range base {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '.', r == '_', r == '-':
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteRune('-')
		}
	}
	name := strings.TrimLeft(b.String(), ".-")
	ext := filepath.Ext(name)
	if name == "" || ext == "" || ext == name || len(name) > 120 {
		return "", errors.New("tên tệp không hợp lệ")
	}
	return name, nil
}

// maxNameTries bounds the numbered names createFree tries.
const maxNameTries = 1000

// createFree writes data to name in dir, or to name-1, name-2… when it is
// taken. The file is created exclusively, so two uploads of the same name
// at once never overwrite each other.
func createFree(dir, name string, data []byte) (string, error) {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 0; i < maxNameTries; i++ {
		candidate := name
		if i > 0 {
			candidate = stem + "-" + strconv.Itoa(i) + ext
		}
		target := filepath.Join(dir, candidate)
		f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, err = f.Write(data)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			_ = os.Remove(target)
			return "", err
		}
		return target, nil
	}
	return "", errors.New("quá nhiều tệp trùng tên")
}

// handleUpload saves the raw body as ?name= in ?dir= (created if needed)
// and returns the project-relative path. Existing files are never
// overwritten: a numbered name is chosen instead.
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot(r)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	name, err := uploadName(r.URL.Query().Get("name"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	dir, err := resolveInRoot(root, r.URL.Query().Get("dir"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if dir != filepath.Clean(root) && isProtectedPath(root, dir) {
		writeError(w, http.StatusBadRequest, "thư mục được bảo vệ")
		return
	}
	// withBodyLimit caps this route at uploadBodyLimit.
	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeBodyError(w, err)
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	target, err := createFree(dir, name, data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	rel, _ := filepath.Rel(root, target)
	writeJSON(w, http.StatusOK, map[string]string{"path": filepath.ToSlash(rel)})
}
