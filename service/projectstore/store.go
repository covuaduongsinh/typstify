// Package projectstore abstracts "a project's files" behind one interface
// with two implementations: the local filesystem (unchanged desktop
// behavior) and a remote Typstify server (Giai đoạn C.1, "Remote Project"
// mode) -- so a minimal remote file browser/editor can reuse the exact
// same shape without the local editing pipeline (LSP, compile, preview,
// Dropbox sync -- all of ui/editors and service/workspace.go) needing to
// know or care which one it's talking to.
package projectstore

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	"looz.ws/typstify/service/remote"
)

// FileInfo describes one entry in a project's file tree.
type FileInfo struct {
	Name    string
	Path    string // slash-separated, relative to the project root
	IsDir   bool
	Size    int64
	ModTime time.Time
}

// ProjectStore reads and writes a project's files. Every path is relative
// to the project root ("" for the root itself); implementations resolve
// it against whatever "the project" means for them (a local directory, or
// whatever project a remote server currently has open).
type ProjectStore interface {
	// Tree lists one directory's entries, non-recursive.
	Tree(ctx context.Context, path string) ([]FileInfo, error)
	ReadFile(ctx context.Context, path string) ([]byte, error)
	WriteFile(ctx context.Context, path string, content []byte) error
	CreateFile(ctx context.Context, path string, isDir bool) error
	DeleteFile(ctx context.Context, path string) error
	RenameFile(ctx context.Context, oldPath, newPath string) error
}

// LocalStore implements ProjectStore against a directory on this
// machine's own filesystem. It exists so a remote-aware caller (e.g. a
// file tree widget) can be written once against ProjectStore and handed
// either implementation -- it is not used by the existing local editing
// pipeline (ui/editors, service/workspace.go), which keeps its own
// existing, unrelated direct os.* calls untouched.
type LocalStore struct {
	Root string
}

func NewLocalStore(root string) *LocalStore { return &LocalStore{Root: root} }

func (s *LocalStore) resolve(relPath string) (string, error) {
	abs := filepath.Join(s.Root, filepath.FromSlash(relPath))
	rel, err := filepath.Rel(s.Root, abs)
	if err != nil || rel == ".." || len(rel) >= 2 && rel[:2] == ".."+string(filepath.Separator) {
		return "", os.ErrPermission
	}
	return abs, nil
}

func (s *LocalStore) Tree(ctx context.Context, path string) ([]FileInfo, error) {
	dir, err := s.resolve(path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	out := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if name == ".typstify" || name == ".git" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(s.Root, filepath.Join(dir, name))
		if err != nil {
			continue
		}
		out = append(out, FileInfo{
			Name:    name,
			Path:    filepath.ToSlash(rel),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func (s *LocalStore) ReadFile(ctx context.Context, path string) ([]byte, error) {
	abs, err := s.resolve(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(abs)
}

func (s *LocalStore) WriteFile(ctx context.Context, path string, content []byte) error {
	abs, err := s.resolve(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return err
	}
	return os.WriteFile(abs, content, 0644)
}

func (s *LocalStore) CreateFile(ctx context.Context, path string, isDir bool) error {
	abs, err := s.resolve(path)
	if err != nil {
		return err
	}
	if isDir {
		return os.MkdirAll(abs, 0755)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(abs, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if f != nil {
		f.Close()
	}
	return err
}

func (s *LocalStore) DeleteFile(ctx context.Context, path string) error {
	abs, err := s.resolve(path)
	if err != nil {
		return err
	}
	return os.RemoveAll(abs)
}

func (s *LocalStore) RenameFile(ctx context.Context, oldPath, newPath string) error {
	oldAbs, err := s.resolve(oldPath)
	if err != nil {
		return err
	}
	newAbs, err := s.resolve(newPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(newAbs), 0755); err != nil {
		return err
	}
	return os.Rename(oldAbs, newAbs)
}

var _ ProjectStore = (*LocalStore)(nil)

// RemoteStore implements ProjectStore against whatever project a remote
// Typstify server currently has open (see remote.Client's file methods --
// it is server-global state, not per-request, so OpenProject/
// CreateProject must be called once before Tree/ReadFile/etc. are useful).
type RemoteStore struct {
	Client *remote.Client
}

func NewRemoteStore(client *remote.Client) *RemoteStore { return &RemoteStore{Client: client} }

func (s *RemoteStore) Tree(ctx context.Context, path string) ([]FileInfo, error) {
	entries, err := s.Client.Tree(path)
	if err != nil {
		return nil, err
	}
	out := make([]FileInfo, len(entries))
	for i, e := range entries {
		out[i] = FileInfo{Name: e.Name, Path: e.Path, IsDir: e.IsDir, Size: e.Size, ModTime: e.ModTime}
	}
	return out, nil
}

func (s *RemoteStore) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return s.Client.ReadFile(path)
}

func (s *RemoteStore) WriteFile(ctx context.Context, path string, content []byte) error {
	return s.Client.WriteFile(path, content)
}

func (s *RemoteStore) CreateFile(ctx context.Context, path string, isDir bool) error {
	return s.Client.CreateFile(path, isDir)
}

func (s *RemoteStore) DeleteFile(ctx context.Context, path string) error {
	return s.Client.DeleteFile(path)
}

func (s *RemoteStore) RenameFile(ctx context.Context, oldPath, newPath string) error {
	return s.Client.RenameFile(oldPath, newPath)
}

var _ ProjectStore = (*RemoteStore)(nil)
