// Package fonts manages the user's extra font directory (list/upload/delete
// individual .ttf/.otf/.ttc files) shared by both the desktop app and the
// web server. The directory itself is service/settings.TypstSettings's
// ExtraFontPath field -- the same setting typst/export.CompileHelper and
// lsp/client.go already read to find fonts, so anything added here is
// picked up by every compile path (export, preview, tinymist LSP) with no
// further plumbing.
package fonts

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"looz.ws/typstify/service/settings"
)

// MaxUploadSize bounds a single uploaded font file. Generous for a real
// font (usually a few hundred KB to a few MB) while still bounding what
// gets read into a file.
const MaxUploadSize = 30 << 20 // 30MiB

var AllowedExt = map[string]bool{
	".ttf": true,
	".otf": true,
	".ttc": true,
}

var filenamePattern = regexp.MustCompile(`^[\p{L}\p{N}_.\- ]+$`)

// Info describes one font file in the managed directory.
type Info struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// SanitizeFilename validates and normalizes a font filename supplied by a
// client (web upload query param, or a desktop file-picker basename),
// preventing path traversal and restricting to supported font extensions.
func SanitizeFilename(name string) (string, error) {
	name = filepath.Base(strings.TrimSpace(name))
	ext := strings.ToLower(filepath.Ext(name))
	if !AllowedExt[ext] {
		return "", fmt.Errorf("unsupported font file type %q -- only .ttf, .otf, .ttc are supported", ext)
	}
	if name == ext || !filenamePattern.MatchString(name) {
		return "", fmt.Errorf("invalid file name")
	}
	return name, nil
}

// Dir returns the directory typst/tinymist search for extra fonts -- the
// same ExtraFontPath setting typst/export.CompileHelper.fontPaths and
// lsp/client.go already read. If the user hasn't pointed ExtraFontPath
// anywhere (the common case -- it's a free-text path nobody sets by
// default), createIfMissing provisions and persists a default managed
// directory under the app's data dir the first time a font is uploaded.
func Dir(cfg *settings.Settings, createIfMissing bool) (string, error) {
	ts := cfg.Typst()
	if ts.ExtraFontPath != "" {
		if createIfMissing {
			if err := os.MkdirAll(ts.ExtraFontPath, 0o755); err != nil {
				return "", err
			}
		}
		return ts.ExtraFontPath, nil
	}
	if !createIfMissing {
		return "", nil
	}

	dir := filepath.Join(cfg.General().RootDir, "fonts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	ts.ExtraFontPath = dir
	if err := ts.Save(); err != nil {
		return "", err
	}
	return dir, nil
}

// List returns the uploaded font files in the managed directory. Empty
// (not an error) when no font has ever been uploaded and the user hasn't
// set ExtraFontPath themselves.
func List(cfg *settings.Settings) ([]Info, error) {
	dir, err := Dir(cfg, false)
	if err != nil {
		return nil, err
	}
	if dir == "" {
		return []Info{}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Info{}, nil
		}
		return nil, err
	}

	out := make([]Info, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !AllowedExt[strings.ToLower(filepath.Ext(e.Name()))] {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, Info{Name: e.Name(), Size: info.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Upload saves r as a font file named filename (validated via
// SanitizeFilename) in the managed directory, provisioning the directory
// if this is the first font ever uploaded. Callers that stream from an
// HTTP request body should bound r themselves (e.g. http.MaxBytesReader)
// to get a request-appropriate error; Upload additionally bounds the copy
// to MaxUploadSize as a backstop.
func Upload(cfg *settings.Settings, filename string, r io.Reader) error {
	name, err := SanitizeFilename(filename)
	if err != nil {
		return err
	}

	dir, err := Dir(cfg, true)
	if err != nil {
		return err
	}

	dest := filepath.Join(dir, name)
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	// Copy up to MaxUploadSize+1 bytes: if the source has more than that,
	// n exceeds MaxUploadSize and the upload is rejected outright, instead
	// of silently truncating the font file into a corrupt one.
	n, cErr := io.CopyN(f, r, MaxUploadSize+1)
	if cErr != nil && cErr != io.EOF {
		f.Close()
		_ = os.Remove(dest)
		return fmt.Errorf("upload failed: %w", cErr)
	}
	if n > MaxUploadSize {
		f.Close()
		_ = os.Remove(dest)
		return fmt.Errorf("font file exceeds maximum size of %d bytes", MaxUploadSize)
	}
	return nil
}

// Delete removes one uploaded font file.
func Delete(cfg *settings.Settings, filename string) error {
	name, err := SanitizeFilename(filename)
	if err != nil {
		return err
	}

	dir, err := Dir(cfg, false)
	if err != nil {
		return err
	}
	if dir == "" {
		return nil
	}

	if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
