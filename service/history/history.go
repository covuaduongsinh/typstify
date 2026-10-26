// Package history keeps restorable versions of a project's files: a
// content-addressed store of file contents plus one manifest per version.
// It lives in the server's storage directory, never in the project, and
// works for any writer (people, co-editing rooms, the AI agent, Dropbox).
package history

import (
	"bufio"
	"compress/gzip"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Kinds of versions.
const (
	KindAuto    = "auto"    // taken periodically when something changed
	KindManual  = "manual"  // named by a user; kept forever
	KindRestore = "restore" // taken right before a restore overwrote files
)

// MaxFileSize skips larger files (videos, archives) from the history.
const MaxFileSize = 25 << 20

var (
	ErrNotFound = errors.New("không tìm thấy phiên bản")
	ErrNoFile   = errors.New("phiên bản không có tệp này")
)

// DefaultQuota bounds the stored contents of one project's history;
// beyond it the oldest automatic versions are dropped.
const DefaultQuota int64 = 2 << 30

var idPattern = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}-[0-9a-f]{8}$`)

// skipDirs are never versioned: VCS data, tool state and build output.
var skipDirs = map[string]bool{".git": true, ".typstify": true, "node_modules": true, "dist": true, ".tmp": true}

// FileEntry is one file of a version.
type FileEntry struct {
	Path    string    `json:"path"`
	Hash    string    `json:"hash"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

// Snapshot describes a version.
type Snapshot struct {
	ID       string    `json:"id"`
	At       time.Time `json:"at"`
	Author   string    `json:"author"`
	Label    string    `json:"label,omitempty"`
	Kind     string    `json:"kind"`
	Files    int       `json:"files"`
	Added    int       `json:"added"`
	Modified int       `json:"modified"`
	Deleted  int       `json:"deleted"`
}

// Change is a file's state now compared with a version.
type Change struct {
	Path   string `json:"path"`
	Status string `json:"status"` // unchanged, modified, deleted (gone now), added (new since)
	Size   int64  `json:"size"`
}

// Store is the history of one project.
type Store struct {
	dir   string
	quota int64
	mu    sync.Mutex
}

func newStore(dir string) *Store { return &Store{dir: dir, quota: DefaultQuota} }

func (s *Store) objectPath(hash string) string {
	return filepath.Join(s.dir, "objects", hash[:2], hash+".gz")
}

func (s *Store) manifestPath(id string) string {
	return filepath.Join(s.dir, "snapshots", id+".json.gz")
}

// writeAtomic writes data through a temp file, synced, then renamed.
func writeAtomic(p string, write func(io.Writer) error) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), "."+filepath.Base(p)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := write(tmp); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}

func gzipTo(w io.Writer, data []byte) error {
	gz := gzip.NewWriter(w)
	if _, err := gz.Write(data); err != nil {
		return err
	}
	return gz.Close()
}

func readGzip(p string) ([]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	return io.ReadAll(gz)
}

func hashOf(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func newID(at time.Time) string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return at.UTC().Format("20060102T150405") + "-" + hex.EncodeToString(b[:])
}

// scan lists the project's files. Hashes of files whose size and mtime
// match the previous version are reused instead of re-reading them. With
// store, new contents are saved as objects as they are read; without it
// (comparisons), nothing is written.
func (s *Store) scan(projectDir string, prev map[string]FileEntry, store bool) ([]FileEntry, error) {
	var out []FileEntry
	err := filepath.WalkDir(projectDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == projectDir {
				return err
			}
			return nil
		}
		if d.IsDir() {
			if p != projectDir && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > MaxFileSize {
			return nil
		}
		rel, err := filepath.Rel(projectDir, p)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if old, ok := prev[rel]; ok && old.Size == info.Size() && old.ModTime.Equal(info.ModTime()) {
			out = append(out, old)
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		h := hashOf(data)
		if _, err := os.Stat(s.objectPath(h)); store && errors.Is(err, os.ErrNotExist) {
			if err := writeAtomic(s.objectPath(h), func(w io.Writer) error { return gzipTo(w, data) }); err != nil {
				return err
			}
		}
		out = append(out, FileEntry{Path: rel, Hash: h, Size: int64(len(data)), ModTime: info.ModTime()})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, err
}

func (s *Store) loadManifest(id string) ([]FileEntry, error) {
	if !idPattern.MatchString(id) {
		return nil, ErrNotFound
	}
	data, err := readGzip(s.manifestPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var files []FileEntry
	return files, json.Unmarshal(data, &files)
}

func (s *Store) loadIndex() ([]Snapshot, error) {
	f, err := os.Open(filepath.Join(s.dir, "index.jsonl"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Snapshot
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		var snap Snapshot
		if json.Unmarshal(sc.Bytes(), &snap) == nil && snap.ID != "" {
			out = append(out, snap)
		}
	}
	return out, sc.Err()
}

func (s *Store) writeIndex(snaps []Snapshot) error {
	return writeAtomic(filepath.Join(s.dir, "index.jsonl"), func(w io.Writer) error {
		enc := json.NewEncoder(w)
		for _, snap := range snaps {
			if err := enc.Encode(snap); err != nil {
				return err
			}
		}
		return nil
	})
}

func byPath(files []FileEntry) map[string]FileEntry {
	m := make(map[string]FileEntry, len(files))
	for _, f := range files {
		m[f.Path] = f
	}
	return m
}

// Take records the project's current files as a new version. Unless
// force is set, nothing is recorded when no file changed since the last
// version (created is then false).
func (s *Store) Take(projectDir, author, label, kind string, force bool) (snap Snapshot, created bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snaps, err := s.loadIndex()
	if err != nil {
		return Snapshot{}, false, err
	}
	var prev map[string]FileEntry
	if len(snaps) > 0 {
		files, err := s.loadManifest(snaps[len(snaps)-1].ID)
		if err == nil {
			prev = byPath(files)
		}
	}
	files, err := s.scan(projectDir, prev, true)
	if err != nil {
		return Snapshot{}, false, err
	}

	now := time.Now()
	snap = Snapshot{ID: newID(now), At: now, Author: author, Label: strings.TrimSpace(label), Kind: kind, Files: len(files)}
	cur := byPath(files)
	for p, f := range cur {
		old, ok := prev[p]
		switch {
		case !ok:
			snap.Added++
		case old.Hash != f.Hash:
			snap.Modified++
		}
	}
	for p := range prev {
		if _, ok := cur[p]; !ok {
			snap.Deleted++
		}
	}
	if !force && len(snaps) > 0 && snap.Added+snap.Modified+snap.Deleted == 0 {
		return snaps[len(snaps)-1], false, nil
	}

	data, err := json.Marshal(files)
	if err != nil {
		return Snapshot{}, false, err
	}
	if err := writeAtomic(s.manifestPath(snap.ID), func(w io.Writer) error { return gzipTo(w, data) }); err != nil {
		return Snapshot{}, false, err
	}
	snaps = append(snaps, snap)
	if err := s.writeIndex(snaps); err != nil {
		return Snapshot{}, false, err
	}
	if kind != KindManual {
		s.pruneLocked(now)
	}
	return snap, true, nil
}

// List returns the versions, newest first.
func (s *Store) List() ([]Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snaps, err := s.loadIndex()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(snaps, func(i, j int) bool { return snaps[i].At.After(snaps[j].At) })
	return snaps, nil
}

// Get returns one version's description.
func (s *Store) Get(id string) (Snapshot, error) {
	snaps, err := s.List()
	if err != nil {
		return Snapshot{}, err
	}
	for _, snap := range snaps {
		if snap.ID == id {
			return snap, nil
		}
	}
	return Snapshot{}, ErrNotFound
}

// Read returns a file's content in a version.
func (s *Store) Read(id, rel string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	files, err := s.loadManifest(id)
	if err != nil {
		return nil, err
	}
	f, ok := byPath(files)[path.Clean(strings.TrimPrefix(filepath.ToSlash(rel), "/"))]
	if !ok {
		return nil, ErrNoFile
	}
	return readGzip(s.objectPath(f.Hash))
}

// Compare lists every file of the version and of the project now with
// its status relative to the version.
func (s *Store) Compare(id, projectDir string) ([]Change, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	files, err := s.loadManifest(id)
	if err != nil {
		return nil, err
	}
	then := byPath(files)
	// Re-scanning with the version as "previous" reuses its hashes for
	// untouched files; changed ones are hashed (and stored) now.
	nowFiles, err := s.scan(projectDir, then, false)
	if err != nil {
		return nil, err
	}
	now := byPath(nowFiles)
	var out []Change
	for p, f := range then {
		switch cur, ok := now[p]; {
		case !ok:
			out = append(out, Change{p, "deleted", f.Size})
		case cur.Hash != f.Hash:
			out = append(out, Change{p, "modified", f.Size})
		default:
			out = append(out, Change{p, "unchanged", f.Size})
		}
	}
	for p, f := range now {
		if _, ok := then[p]; !ok {
			out = append(out, Change{p, "added", f.Size})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// Writer applies a restore to the project: write puts a file's content,
// remove deletes one. Both receive project-relative slash paths.
type Writer struct {
	Write  func(rel string, data []byte) error
	Remove func(rel string) error
}

// Restore puts the version's files back: only `paths` when given, else
// every file of the version. deleteNewer also removes files created after
// the version (whole-project restores only). A KindRestore version of the
// current state is taken first, so a restore can itself be undone.
func (s *Store) Restore(id, projectDir, author string, paths []string, deleteNewer bool, w Writer) (Snapshot, error) {
	if _, err := s.Get(id); err != nil {
		return Snapshot{}, err
	}
	safety, _, err := s.Take(projectDir, author, "", KindRestore, true)
	if err != nil {
		return Snapshot{}, fmt.Errorf("không chụp được bản an toàn: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	files, err := s.loadManifest(id)
	if err != nil {
		return Snapshot{}, err
	}
	then := byPath(files)
	targets := paths
	if len(targets) == 0 {
		for p := range then {
			targets = append(targets, p)
		}
	}
	for _, rel := range targets {
		f, ok := then[rel]
		if !ok {
			// Restoring a file the version did not have means removing it.
			if len(paths) > 0 {
				if err := w.Remove(rel); err != nil && !errors.Is(err, os.ErrNotExist) {
					return safety, err
				}
			}
			continue
		}
		data, err := readGzip(s.objectPath(f.Hash))
		if err != nil {
			return safety, err
		}
		if err := w.Write(rel, data); err != nil {
			return safety, err
		}
	}
	if deleteNewer && len(paths) == 0 {
		safetyFiles, err := s.loadManifest(safety.ID)
		if err == nil {
			for _, f := range safetyFiles {
				if _, ok := then[f.Path]; !ok {
					if err := w.Remove(f.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
						return safety, err
					}
				}
			}
		}
	}
	return safety, nil
}

// pruneLocked thins automatic and restore versions (all kept for a day,
// one per hour for a week, one per day after) and deletes contents no
// remaining version uses. Named versions are never pruned.
func (s *Store) pruneLocked(now time.Time) {
	snaps, err := s.loadIndex()
	if err != nil || len(snaps) == 0 {
		return
	}
	seen := map[string]bool{}
	var keep, drop []Snapshot
	// Newest first, so each bucket keeps its latest version.
	for i := len(snaps) - 1; i >= 0; i-- {
		snap := snaps[i]
		age := now.Sub(snap.At)
		if snap.Kind == KindManual || age < 24*time.Hour || i == len(snaps)-1 {
			keep = append(keep, snap)
			continue
		}
		bucket := snap.At.UTC().Format("2006-01-02")
		if age < 7*24*time.Hour {
			bucket = snap.At.UTC().Format("2006-01-02T15")
		}
		if seen[bucket] {
			drop = append(drop, snap)
			continue
		}
		seen[bucket] = true
		keep = append(keep, snap)
	}
	sort.SliceStable(keep, func(i, j int) bool { return keep[i].At.Before(keep[j].At) })

	manifests := map[string][]FileEntry{}
	for _, snap := range keep {
		files, err := s.loadManifest(snap.ID)
		if errors.Is(err, ErrNotFound) {
			continue // lost in a crash: drop it from the index below
		}
		if err != nil {
			return // never delete contents on a partial view
		}
		manifests[snap.ID] = files
	}
	keep = s.enforceQuotaLocked(keep, manifests, &drop)

	var listed []Snapshot
	for _, snap := range keep {
		if _, ok := manifests[snap.ID]; ok {
			listed = append(listed, snap)
		}
	}
	// The index is rewritten before any manifest is deleted, so a crash
	// in between leaves only unreferenced files behind.
	if len(listed) != len(snaps) {
		if s.writeIndex(listed) != nil {
			return
		}
	}
	for _, snap := range drop {
		_ = os.Remove(s.manifestPath(snap.ID))
	}
	// Also sweeps contents no version uses any more.
	used := map[string]bool{}
	for _, snap := range listed {
		for _, f := range manifests[snap.ID] {
			used[f.Hash] = true
		}
	}
	_ = filepath.WalkDir(filepath.Join(s.dir, "objects"), func(p string, d fs.DirEntry, err error) error {
		// Temp files of an object being written are left alone.
		if err != nil || d.IsDir() {
			return nil
		}
		if h, ok := strings.CutSuffix(d.Name(), ".gz"); ok && !used[h] {
			_ = os.Remove(p)
		}
		return nil
	})
}

// enforceQuotaLocked drops the oldest automatic and restore versions
// (never named ones, never the newest) while the contents they use exceed
// the quota.
func (s *Store) enforceQuotaLocked(keep []Snapshot, manifests map[string][]FileEntry, drop *[]Snapshot) []Snapshot {
	if s.quota <= 0 {
		return keep
	}
	size := map[string]int64{}
	refs := map[string]int{}
	var total int64
	for _, snap := range keep {
		for _, f := range manifests[snap.ID] {
			if refs[f.Hash] == 0 {
				if info, err := os.Stat(s.objectPath(f.Hash)); err == nil {
					size[f.Hash] = info.Size()
					total += info.Size()
				}
			}
			refs[f.Hash]++
		}
	}
	for i := 0; total > s.quota && i < len(keep)-1; {
		snap := keep[i]
		if snap.Kind == KindManual {
			i++
			continue
		}
		for _, f := range manifests[snap.ID] {
			if refs[f.Hash]--; refs[f.Hash] == 0 {
				total -= size[f.Hash]
			}
		}
		*drop = append(*drop, snap)
		keep = append(keep[:i], keep[i+1:]...)
	}
	return keep
}

// Manager hands out the store of each project.
type Manager struct {
	Base string
	// Quota overrides DefaultQuota (bytes of stored contents per project).
	Quota int64

	mu     sync.Mutex
	stores map[string]*Store
}

// For returns the history of the project identified by key (a project
// id, or a hash of an unregistered project's path).
func (m *Manager) For(key string) *Store {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stores == nil {
		m.stores = map[string]*Store{}
	}
	sum := sha256.Sum256([]byte(key))
	k := hex.EncodeToString(sum[:8])
	st, ok := m.stores[k]
	if !ok {
		st = newStore(filepath.Join(m.Base, k))
		if m.Quota > 0 {
			st.quota = m.Quota
		}
		m.stores[k] = st
	}
	return st
}
