// Package universe reads the Typst Universe package index
// (packages.typst.org) for the template gallery and the package browser,
// and creates projects from package templates.
package universe

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// DefaultIndexURL lists every published version of every @preview package.
const DefaultIndexURL = "https://packages.typst.org/preview/index.json"

const (
	// maxIndexBytes bounds the downloaded index (about 2.4 MB in 2026-10).
	maxIndexBytes = 32 << 20
	// refreshAfter is how long a fetched index is used before refetching.
	refreshAfter = 12 * time.Hour
	// retryAfter spaces out refetch attempts while the registry is down.
	retryAfter = 5 * time.Minute
)

// Template is a package's [template] section.
type Template struct {
	Path       string `json:"path"`
	Entrypoint string `json:"entrypoint"`
	Thumbnail  string `json:"thumbnail,omitempty"`
}

// Package is one package at its latest version.
type Package struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description,omitempty"`
	Authors     []string  `json:"authors,omitempty"`
	License     string    `json:"license,omitempty"`
	Repository  string    `json:"repository,omitempty"`
	Homepage    string    `json:"homepage,omitempty"`
	Keywords    []string  `json:"keywords,omitempty"`
	Categories  []string  `json:"categories,omitempty"`
	Disciplines []string  `json:"disciplines,omitempty"`
	Compiler    string    `json:"compiler,omitempty"`
	Template    *Template `json:"template,omitempty"`
	UpdatedAt   int64     `json:"updatedAt,omitempty"`
}

// Query selects packages from the index.
type Query struct {
	// Kind is "template" (only packages with a template), "package"
	// (only the others) or empty (all).
	Kind     string
	Text     string // matched against name, description and keywords
	Category string
	Offset   int
	Limit    int
}

// Page is one page of query results.
type Page struct {
	Packages []Package `json:"packages"`
	Total    int       `json:"total"`
}

// Index is the Universe index, fetched lazily, kept in memory and on disk.
// A fetch failure falls back to the last good copy (memory, then disk).
type Index struct {
	// URL defaults to DefaultIndexURL.
	URL string
	// CacheFile, when set, keeps the raw index across restarts.
	CacheFile string
	Client    *http.Client

	group      singleflight.Group
	mu         sync.Mutex
	packages   []Package // latest version of each package, by name
	fetchedAt  time.Time
	triedAt    time.Time
	lastErr    error // last failed first load, returned for retryAfter
	refreshing bool
}

// Search returns the packages matching q, most recently updated first.
func (ix *Index) Search(ctx context.Context, q Query) (Page, error) {
	all, err := ix.load(ctx)
	if err != nil {
		return Page{}, err
	}
	text := strings.ToLower(strings.TrimSpace(q.Text))
	var hits []Package
	for _, p := range all {
		switch q.Kind {
		case "template":
			if p.Template == nil {
				continue
			}
		case "package":
			if p.Template != nil {
				continue
			}
		}
		if q.Category != "" && !slices.Contains(p.Categories, q.Category) {
			continue
		}
		if text != "" && !matches(p, text) {
			continue
		}
		hits = append(hits, p)
	}
	slices.SortStableFunc(hits, func(a, b Package) int {
		if text != "" {
			// An exact or prefix name match first when searching.
			if ra, rb := nameRank(a.Name, text), nameRank(b.Name, text); ra != rb {
				return ra - rb
			}
		}
		return cmp.Compare(b.UpdatedAt, a.UpdatedAt)
	})
	limit := q.Limit
	if limit <= 0 || limit > 200 {
		limit = 48
	}
	off := min(max(q.Offset, 0), len(hits))
	end := min(off+limit, len(hits))
	return Page{Packages: hits[off:end], Total: len(hits)}, nil
}

// Lookup returns the latest version of name.
func (ix *Index) Lookup(ctx context.Context, name string) (Package, bool, error) {
	all, err := ix.load(ctx)
	if err != nil {
		return Package{}, false, err
	}
	i, ok := slices.BinarySearchFunc(all, name, func(p Package, n string) int { return strings.Compare(p.Name, n) })
	if !ok {
		return Package{}, false, nil
	}
	return all[i], true, nil
}

func matches(p Package, text string) bool {
	if strings.Contains(strings.ToLower(p.Name), text) || strings.Contains(strings.ToLower(p.Description), text) {
		return true
	}
	for _, k := range p.Keywords {
		if strings.Contains(strings.ToLower(k), text) {
			return true
		}
	}
	return false
}

func nameRank(name, text string) int {
	switch {
	case name == text:
		return 0
	case strings.HasPrefix(name, text):
		return 1
	}
	return 2
}

// load returns the current package list. Only the very first load waits
// for the network; a stale list is served while a refresh runs behind it.
// While the registry is unreachable and no copy exists, the last error is
// returned for retryAfter instead of a new 60-second attempt per call.
func (ix *Index) load(ctx context.Context) ([]Package, error) {
	ix.mu.Lock()
	if ix.packages == nil && ix.CacheFile != "" {
		ix.loadDiskLocked()
	}
	if ix.packages != nil {
		list := ix.packages
		if time.Since(ix.fetchedAt) >= refreshAfter && time.Since(ix.triedAt) >= retryAfter && !ix.refreshing {
			ix.refreshing, ix.triedAt = true, time.Now()
			go func() { _ = <-ix.refresh() }()
		}
		ix.mu.Unlock()
		return list, nil
	}
	if ix.lastErr != nil && time.Since(ix.triedAt) < retryAfter {
		err := ix.lastErr
		ix.mu.Unlock()
		return nil, err
	}
	ix.triedAt = time.Now()
	ix.mu.Unlock()
	// The fetch is shared and outlives this request: a caller that gives
	// up only stops waiting for it.
	select {
	case err := <-ix.refresh():
		ix.mu.Lock()
		defer ix.mu.Unlock()
		if err != nil {
			ix.lastErr = fmt.Errorf("không tải được danh sách gói Typst Universe: %w", err)
			return nil, ix.lastErr
		}
		ix.lastErr = nil
		return ix.packages, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (ix *Index) loadDiskLocked() {
	raw, err := os.ReadFile(ix.CacheFile)
	if err != nil {
		return
	}
	list, err := parseIndex(raw)
	if err != nil {
		return
	}
	ix.packages = list
	if info, err := os.Stat(ix.CacheFile); err == nil {
		ix.fetchedAt = info.ModTime()
	}
}

// refresh fetches the index in the background (with its own timeout);
// concurrent callers share one fetch. The channel gets its result.
func (ix *Index) refresh() <-chan error {
	out := make(chan error, 1)
	ch := ix.group.DoChan("index", func() (any, error) {
		defer func() {
			ix.mu.Lock()
			ix.refreshing = false
			ix.mu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		raw, err := ix.fetch(ctx)
		if err != nil {
			return nil, err
		}
		list, err := parseIndex(raw)
		if err != nil {
			return nil, err
		}
		ix.mu.Lock()
		ix.packages, ix.fetchedAt = list, time.Now()
		ix.mu.Unlock()
		if ix.CacheFile != "" {
			_ = writeAtomic(ix.CacheFile, raw)
		}
		return nil, nil
	})
	go func() { out <- (<-ch).Err }()
	return out
}

func (ix *Index) fetch(ctx context.Context) ([]byte, error) {
	url := ix.URL
	if url == "" {
		url = DefaultIndexURL
	}
	client := ix.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, maxIndexBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxIndexBytes {
		return nil, errors.New("danh sách gói quá lớn")
	}
	return raw, nil
}

// parseIndex keeps the latest version of each package, sorted by name.
func parseIndex(raw []byte) ([]Package, error) {
	var all []Package
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, err
	}
	latest := make(map[string]Package, len(all)/3)
	for _, p := range all {
		if p.Name == "" {
			continue
		}
		if cur, ok := latest[p.Name]; !ok || compareVersions(p.Version, cur.Version) > 0 {
			latest[p.Name] = p
		}
	}
	out := make([]Package, 0, len(latest))
	for _, p := range latest {
		out = append(out, p)
	}
	slices.SortFunc(out, func(a, b Package) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// compareVersions compares "major.minor.patch" numerically.
func compareVersions(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < max(len(pa), len(pb)); i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	_, err = tmp.Write(data)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
	}
	return err
}
