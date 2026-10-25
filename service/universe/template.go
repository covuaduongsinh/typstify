package universe

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	// Limits for one template copied into a project.
	maxTemplateFiles = 2000 // files and folders
	maxTemplateBytes = 32 << 20
	// maxReadmeBytes bounds a package README served to the browser.
	maxReadmeBytes = 1 << 20
)

var ErrNoTemplate = errors.New("gói này không có template")

// manifest is the part of typst.toml a template needs.
type manifest struct {
	Package struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
	} `toml:"package"`
	Template *Template `toml:"template"`
}

// ReadTemplate returns the [template] section of the package in pkgDir.
func ReadTemplate(pkgDir string) (Template, error) {
	raw, err := os.ReadFile(filepath.Join(pkgDir, "typst.toml"))
	if err != nil {
		return Template{}, err
	}
	var m manifest
	if err := toml.Unmarshal(raw, &m); err != nil {
		return Template{}, fmt.Errorf("typst.toml: %w", err)
	}
	if m.Template == nil || m.Template.Path == "" || m.Template.Entrypoint == "" {
		return Template{}, ErrNoTemplate
	}
	return *m.Template, nil
}

// inside resolves rel under base, refusing anything that leaves it.
func inside(base, rel string) (string, error) {
	rel = filepath.FromSlash(rel)
	if rel == "" || filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" {
		return "", fmt.Errorf("đường dẫn không hợp lệ: %q", rel)
	}
	p := filepath.Join(base, rel)
	if r, err := filepath.Rel(base, p); err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("đường dẫn không hợp lệ: %q", rel)
	}
	return p, nil
}

// CopyTemplate copies the template directory of the package in pkgDir into
// dst (which must exist), like `typst init`, and returns the entrypoint
// relative to dst (slash-separated). Symlinks and special files are
// skipped; existing files in dst are never overwritten.
func CopyTemplate(pkgDir, dst string) (string, error) {
	t, err := ReadTemplate(pkgDir)
	if err != nil {
		return "", err
	}
	// Resolve symlinks first: the containment check must hold for the real
	// paths (a package dir from the typst CLI's cache may contain links).
	realPkg, err := filepath.EvalSymlinks(pkgDir)
	if err != nil {
		return "", err
	}
	src, err := inside(realPkg, t.Path)
	if err != nil {
		return "", err
	}
	if src, err = filepath.EvalSymlinks(src); err != nil {
		return "", fmt.Errorf("thư mục template %q không tồn tại", t.Path)
	}
	if _, err := inside(realPkg, mustRel(realPkg, src)); err != nil {
		return "", err
	}
	if info, err := os.Lstat(src); err != nil || !info.IsDir() {
		return "", fmt.Errorf("thư mục template %q không tồn tại", t.Path)
	}
	entry, err := inside(src, t.Entrypoint)
	if err != nil {
		return "", err
	}
	if info, err := os.Lstat(entry); err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("tệp chính %q của template không tồn tại", t.Entrypoint)
	}

	files, total := 0, int64(0)
	err = filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil || rel == "." {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() && strings.EqualFold(d.Name(), ".git") {
			return fs.SkipDir // a template's own repository is not the project's
		}
		if !d.IsDir() && !d.Type().IsRegular() {
			return nil // symlinks and devices are not copied
		}
		if files++; files > maxTemplateFiles {
			return errors.New("template quá lớn")
		}
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if total += info.Size(); total > maxTemplateBytes {
			return errors.New("template quá lớn")
		}
		return copyNew(p, target)
	})
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(t.Entrypoint))), nil
}

// mustRel is filepath.Rel, or a path inside() refuses when there is none.
func mustRel(base, p string) string {
	r, err := filepath.Rel(base, p)
	if err != nil {
		return ".."
	}
	return r
}

func copyNew(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	return err
}

// ReadReadme returns the README of the package in pkgDir, or "" when it
// has none.
func ReadReadme(pkgDir string) (string, error) {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if !e.Type().IsRegular() || !strings.EqualFold(e.Name(), "README.md") {
			continue
		}
		f, err := os.Open(filepath.Join(pkgDir, e.Name()))
		if err != nil {
			return "", err
		}
		defer f.Close()
		raw, err := io.ReadAll(io.LimitReader(f, maxReadmeBytes))
		return string(raw), err
	}
	return "", nil
}
