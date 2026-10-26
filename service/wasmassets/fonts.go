// Package wasmassets serves what the in-browser Typst compiler (typst.ts,
// web/src/lib/wasm) cannot read by itself: font files and package
// archives. The native typst CLI finds both on the local disk; the WASM
// build only sees what the server hands it over HTTP.
package wasmassets

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FontExt lists the font formats typst can load.
var FontExt = map[string]bool{".ttf": true, ".otf": true, ".ttc": true, ".otc": true}

// DefaultSystemFontPatterns selects the system fonts handed to the browser
// compiler: the families chessbook/lib/theme.typ asks for (Roboto, Noto
// Serif), Noto Sans and Noto Sans Symbols 2 for the chess figurines. A
// whole system font directory would be hundreds of megabytes.
var DefaultSystemFontPatterns = []string{
	"Roboto-*", "NotoSerif-*", "NotoSans-*", "NotoSansSymbols2-*",
}

// maxFontsPerScan bounds a scan of a misconfigured directory.
const maxFontsPerScan = 512

// Font is one font file the browser compiler may download.
type Font struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size"`
	// Version changes whenever the file does, so the URL can be cached
	// as immutable by the browser.
	Version string `json:"version"`

	path string
}

// FontSources says where to look for fonts.
type FontSources struct {
	// Dirs are scanned recursively and every font file is included
	// (uploaded fonts, TYPSTIFY_WASM_FONT_DIRS).
	Dirs []string
	// SystemDirs are scanned recursively but only files whose base name
	// matches one of Patterns (case-insensitive glob) are included.
	SystemDirs []string
	Patterns   []string
}

// FontIndex remembers the last scan so a font id can be resolved back to
// its file without exposing server paths to the client.
type FontIndex struct {
	mu       sync.Mutex
	byID     map[string]Font
	lastScan time.Time
}

func NewFontIndex() *FontIndex {
	return &FontIndex{byID: map[string]Font{}}
}

// Scan lists the fonts in src, de-duplicated by file name (the first
// source wins, so uploaded fonts override system ones), and records them
// for Lookup.
func (x *FontIndex) Scan(src FontSources) []Font {
	seen := map[string]bool{}
	var out []Font
	add := func(dir string, match func(name string) bool) {
		if dir == "" {
			return
		}
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if p == dir {
					return err
				}
				return nil
			}
			if len(out) >= maxFontsPerScan {
				return fs.SkipAll
			}
			if d.IsDir() || !d.Type().IsRegular() {
				return nil
			}
			name := d.Name()
			key := strings.ToLower(name)
			if !FontExt[filepath.Ext(key)] || seen[key] || !match(name) {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			abs, err := filepath.Abs(p)
			if err != nil {
				return nil
			}
			seen[key] = true
			out = append(out, Font{
				ID:      fontID(abs),
				Name:    name,
				Size:    info.Size(),
				Version: fontVersion(info.Size(), info.ModTime()),
				path:    abs,
			})
			return nil
		})
	}

	all := func(string) bool { return true }
	for _, d := range src.Dirs {
		add(d, all)
	}
	matchPattern := func(name string) bool {
		lower := strings.ToLower(name)
		for _, pat := range src.Patterns {
			if ok, _ := filepath.Match(strings.ToLower(pat), lower); ok {
				return true
			}
		}
		return false
	}
	for _, d := range src.SystemDirs {
		add(d, matchPattern)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })

	byID := make(map[string]Font, len(out))
	for _, f := range out {
		byID[f.ID] = f
	}
	x.mu.Lock()
	x.byID = byID
	x.lastScan = time.Now()
	x.mu.Unlock()
	return out
}

// RescanAllowed reports whether the last scan is older than interval, so
// callers can bound how often a lookup miss triggers a directory walk.
func (x *FontIndex) RescanAllowed(interval time.Duration) bool {
	x.mu.Lock()
	defer x.mu.Unlock()
	return time.Since(x.lastScan) >= interval
}

// Lookup returns the file path of a font id from the last Scan.
func (x *FontIndex) Lookup(id string) (string, bool) {
	x.mu.Lock()
	defer x.mu.Unlock()
	f, ok := x.byID[id]
	return f.path, ok
}

func fontID(abs string) string {
	sum := sha256.Sum256([]byte(abs))
	return hex.EncodeToString(sum[:8])
}

func fontVersion(size int64, mod time.Time) string {
	return strconv.FormatInt(size, 36) + "-" + strconv.FormatInt(mod.Unix(), 36)
}

// SystemFontDirs returns the operating system's font directories, the
// same places typst itself searches.
func SystemFontDirs() []string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		dirs := []string{filepath.Join(os.Getenv("WINDIR"), "Fonts")}
		if la := os.Getenv("LOCALAPPDATA"); la != "" {
			dirs = append(dirs, filepath.Join(la, "Microsoft", "Windows", "Fonts"))
		}
		return dirs
	case "darwin":
		return []string{"/Library/Fonts", "/System/Library/Fonts", filepath.Join(home, "Library", "Fonts")}
	default:
		dirs := []string{"/usr/share/fonts", "/usr/local/share/fonts"}
		if home != "" {
			dirs = append(dirs, filepath.Join(home, ".local", "share", "fonts"), filepath.Join(home, ".fonts"))
		}
		return dirs
	}
}

// SplitList splits an OS path list (TYPSTIFY_WASM_FONT_DIRS) or a comma
// list (TYPSTIFY_WASM_SYSTEM_FONTS), dropping empty items.
func SplitList(s string, sep rune) []string {
	var out []string
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return r == sep }) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
