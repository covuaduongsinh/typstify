package wasmassets

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

var (
	ErrBadSpec  = errors.New("tên package không hợp lệ")
	ErrNotFound = errors.New("không tìm thấy package")
)

const (
	// maxPackageBytes bounds both a package served from disk and one
	// downloaded from the registry (uncompressed).
	maxPackageBytes = 128 << 20
	maxPackageFiles = 20000
	// The in-memory cache of built archives is bounded in bytes; an archive
	// larger than maxCachedArchive is rebuilt on each request instead.
	maxCacheBytes    = 96 << 20
	maxCachedArchive = 24 << 20
	// maxDownloads bounds concurrent registry downloads; a spec that failed
	// to download is not retried for failedDownloadTTL.
	maxDownloads      = 2
	failedDownloadTTL = 10 * time.Minute
)

var (
	identRe   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	versionRe = regexp.MustCompile(`^[0-9]{1,6}\.[0-9]{1,6}\.[0-9]{1,6}$`)
)

// Spec identifies a package as in `@namespace/name:version`.
type Spec struct {
	Namespace, Name, Version string
}

func (s Spec) String() string { return "@" + s.Namespace + "/" + s.Name + ":" + s.Version }

// ParseSpec validates the three parts of a package spec. Validation is
// what keeps them safe to join into a file path.
func ParseSpec(namespace, name, version string) (Spec, error) {
	if !identRe.MatchString(namespace) || !identRe.MatchString(name) || !versionRe.MatchString(version) {
		return Spec{}, ErrBadSpec
	}
	return Spec{namespace, name, version}, nil
}

// Archive is a package packed as .tar.gz, the format typst.ts unpacks.
type Archive struct {
	Data []byte
	ETag string
	// Immutable is true for registry packages: a published version never
	// changes, so the browser may cache it forever.
	Immutable bool
}

// Downloader fetches the registry tarball of a @preview package.
type Downloader func(ctx context.Context, spec Spec) (io.ReadCloser, error)

// Packages resolves package archives for the browser compiler.
type Packages struct {
	// SearchDirs returns the package roots (<root>/<namespace>/<name>/<version>)
	// to search, evaluated on each request because settings can change.
	SearchDirs func() []string
	// CacheDir returns where @preview downloads are unpacked, the same
	// cache the typst CLI uses. Empty disables downloading.
	CacheDir func() string
	// Download fetches from the registry; defaults to packages.typst.org.
	Download Downloader

	group singleflight.Group // de-duplicates concurrent downloads and builds

	mu         sync.Mutex
	archives   map[string]Archive // key: dir + "\x00" + stamp
	order      []string
	cacheBytes int
	failed     map[string]time.Time // spec -> when its download failed

	downloadSlots chan struct{}
	slotsOnce     sync.Once
}

// Archive returns the .tar.gz of spec, downloading @preview packages
// into the cache when no search dir has them.
func (p *Packages) Archive(ctx context.Context, spec Spec) (Archive, error) {
	dir, err := p.Dir(ctx, spec)
	if err != nil {
		return Archive{}, err
	}
	a, err := p.build(dir)
	if err != nil {
		return Archive{}, err
	}
	a.Immutable = spec.Namespace == "preview"
	return a, nil
}

// Dir returns the directory holding spec's files (with its typst.toml),
// downloading @preview packages into the cache when no search dir has them.
func (p *Packages) Dir(ctx context.Context, spec Spec) (string, error) {
	dir, err := p.locate(spec)
	if errors.Is(err, ErrNotFound) && spec.Namespace == "preview" && p.CacheDir != nil && p.CacheDir() != "" {
		dir, err = p.fetchOnce(ctx, spec)
	}
	return dir, err
}

func (p *Packages) dirs() []string {
	var out []string
	if p.SearchDirs != nil {
		out = append(out, p.SearchDirs()...)
	}
	if p.CacheDir != nil {
		out = append(out, p.CacheDir())
	}
	return out
}

func (p *Packages) locate(spec Spec) (string, error) {
	for _, root := range p.dirs() {
		if root == "" {
			continue
		}
		dir := filepath.Join(root, spec.Namespace, spec.Name, spec.Version)
		if info, err := os.Stat(filepath.Join(dir, "typst.toml")); err == nil && info.Mode().IsRegular() {
			return dir, nil
		}
	}
	return "", ErrNotFound
}

// fetchOnce downloads spec into the cache. Concurrent requests for the
// same spec share one download, at most maxDownloads run at once, and a
// failure is remembered for failedDownloadTTL so a loop of requests for a
// missing package does not turn into a loop of registry requests.
func (p *Packages) fetchOnce(ctx context.Context, spec Spec) (string, error) {
	key := spec.String()
	p.mu.Lock()
	if t, ok := p.failed[key]; ok && time.Since(t) < failedDownloadTTL {
		p.mu.Unlock()
		return "", ErrNotFound
	}
	p.mu.Unlock()

	v, err, _ := p.group.Do("dl\x00"+key, func() (any, error) {
		if dir, err := p.locate(spec); err == nil {
			return dir, nil
		}
		p.slotsOnce.Do(func() { p.downloadSlots = make(chan struct{}, maxDownloads) })
		select {
		case p.downloadSlots <- struct{}{}:
		case <-ctx.Done():
			return "", ctx.Err()
		}
		defer func() { <-p.downloadSlots }()
		dir, err := p.download(ctx, spec)
		if err != nil && ctx.Err() == nil {
			p.mu.Lock()
			if p.failed == nil {
				p.failed = map[string]time.Time{}
			}
			for k, t := range p.failed {
				if time.Since(t) >= failedDownloadTTL {
					delete(p.failed, k)
				}
			}
			p.failed[key] = time.Now()
			p.mu.Unlock()
		}
		return dir, err
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

type packFile struct {
	rel  string
	abs  string
	mode fs.FileMode
	size int64
	mod  time.Time
}

// build packs dir, reusing the cached archive while no file changed.
func (p *Packages) build(dir string) (Archive, error) {
	// A package root may itself be a symlink (e.g. a linked dev checkout);
	// files inside it that are symlinks are still skipped.
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		dir = real
	}
	var files []packFile
	var total int64
	h := sha256.New()
	err := filepath.WalkDir(dir, func(fp string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil // symlinks and devices are never packed
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, fp)
		if err != nil {
			return err
		}
		total += info.Size()
		if total > maxPackageBytes || len(files) >= maxPackageFiles {
			return fmt.Errorf("package quá lớn")
		}
		f := packFile{filepath.ToSlash(rel), fp, info.Mode().Perm(), info.Size(), info.ModTime()}
		files = append(files, f)
		fmt.Fprintf(h, "%s\x00%d\x00%d\n", f.rel, f.size, f.mod.UnixNano())
		return nil
	})
	if err != nil {
		return Archive{}, err
	}
	stamp := hex.EncodeToString(h.Sum(nil)[:12])
	key := dir + "\x00" + stamp

	p.mu.Lock()
	if a, ok := p.archives[key]; ok {
		p.mu.Unlock()
		return a, nil
	}
	p.mu.Unlock()

	v, err, _ := p.group.Do("build\x00"+key, func() (any, error) { return p.pack(key, stamp, files) })
	if err != nil {
		return Archive{}, err
	}
	return v.(Archive), nil
}

func (p *Packages) pack(key, stamp string, files []packFile) (Archive, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, f := range files {
		hdr := &tar.Header{Name: f.rel, Mode: int64(f.mode), Size: f.size, ModTime: f.mod, Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			return Archive{}, err
		}
		src, err := os.Open(f.abs)
		if err != nil {
			return Archive{}, err
		}
		_, err = io.CopyN(tw, src, f.size)
		src.Close()
		if err != nil {
			return Archive{}, err
		}
	}
	if err := tw.Close(); err != nil {
		return Archive{}, err
	}
	if err := gz.Close(); err != nil {
		return Archive{}, err
	}

	a := Archive{Data: buf.Bytes(), ETag: `"` + stamp + `"`}
	if len(a.Data) > maxCachedArchive {
		return a, nil
	}
	p.mu.Lock()
	if p.archives == nil {
		p.archives = map[string]Archive{}
	}
	if _, ok := p.archives[key]; !ok {
		p.archives[key] = a
		p.order = append(p.order, key)
		p.cacheBytes += len(a.Data)
		for p.cacheBytes > maxCacheBytes && len(p.order) > 1 {
			old := p.order[0]
			p.cacheBytes -= len(p.archives[old].Data)
			delete(p.archives, old)
			p.order = p.order[1:]
		}
	}
	p.mu.Unlock()
	return a, nil
}

// download unpacks the registry tarball into <cache>/preview/<name>/<version>,
// writing to a temporary sibling first so a failed download never leaves
// a half-extracted package that locate would accept.
func (p *Packages) download(ctx context.Context, spec Spec) (string, error) {
	dl := p.Download
	if dl == nil {
		dl = RegistryDownload
	}
	body, err := dl(ctx, spec)
	if err != nil {
		return "", err
	}
	defer body.Close()

	parent := filepath.Join(p.CacheDir(), spec.Namespace, spec.Name)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp(parent, "."+spec.Version+".tmp-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	if err := extractTarGz(io.LimitReader(body, maxPackageBytes), tmp); err != nil {
		return "", fmt.Errorf("giải nén %s: %w", spec, err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "typst.toml")); err != nil {
		return "", fmt.Errorf("%s: thiếu typst.toml", spec)
	}
	final := filepath.Join(parent, spec.Version)
	if err := os.Rename(tmp, final); err != nil {
		// Another process may have unpacked it meanwhile.
		if _, statErr := os.Stat(filepath.Join(final, "typst.toml")); statErr == nil {
			return final, nil
		}
		return "", err
	}
	return final, nil
}

// extractTarGz unpacks regular files and directories only, rejecting any
// entry that would land outside dest.
func extractTarGz(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var total int64
	files := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := path.Clean(strings.TrimPrefix(strings.ReplaceAll(hdr.Name, "\\", "/"), "./"))
		if name == "." {
			continue
		}
		if path.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") || strings.Contains(name, ":") {
			return fmt.Errorf("đường dẫn không hợp lệ trong gói: %q", hdr.Name)
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			files++
			total += hdr.Size
			if total > maxPackageBytes || files > maxPackageFiles {
				return fmt.Errorf("package quá lớn")
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			_, err = io.CopyN(f, tr, hdr.Size)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				return err
			}
		default:
			// Symlinks, hard links and devices are skipped.
		}
	}
}

var registryClient = &http.Client{Timeout: 60 * time.Second}

// RegistryDownload fetches a @preview package from packages.typst.org,
// the registry the typst CLI uses.
func RegistryDownload(ctx context.Context, spec Spec) (io.ReadCloser, error) {
	if spec.Namespace != "preview" {
		return nil, ErrNotFound
	}
	url := "https://packages.typst.org/preview/" + spec.Name + "-" + spec.Version + ".tar.gz"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := registryClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusNotFound {
		res.Body.Close()
		return nil, ErrNotFound
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("packages.typst.org: %s", res.Status)
	}
	return res.Body, nil
}

// TypstDataPackageDir is where the typst CLI keeps local packages by
// default ({data-dir}/typst/packages), e.g. where
// scripts/install-chessbook.sh puts @local/chessbook.
func TypstDataPackageDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" && runtime.GOOS != "windows" {
		return filepath.Join(d, "typst", "packages")
	}
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		if d := os.Getenv("APPDATA"); d != "" {
			return filepath.Join(d, "typst", "packages")
		}
	case "darwin":
		if home != "" {
			return filepath.Join(home, "Library", "Application Support", "typst", "packages")
		}
	default:
		if home != "" {
			return filepath.Join(home, ".local", "share", "typst", "packages")
		}
	}
	return ""
}
