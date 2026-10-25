package universe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// DefaultThumbnailURL is where Universe publishes template thumbnails:
// {base}/{name}-{version}.webp and {base}/{name}-{version}-small.webp.
const DefaultThumbnailURL = "https://packages.typst.org/preview/thumbnails"

const (
	maxThumbnailBytes = 4 << 20
	// missingTTL is how long a thumbnail the registry does not have is not
	// asked for again.
	missingTTL = time.Hour
)

var ErrNoThumbnail = errors.New("không có ảnh xem trước")

// Thumbnails fetches template thumbnails once and keeps them on disk, so
// browsers load them from this server instead of a third party.
type Thumbnails struct {
	BaseURL  string // defaults to DefaultThumbnailURL
	CacheDir string // empty: no disk cache
	Client   *http.Client

	group   singleflight.Group // one upstream request per file at a time
	mu      sync.Mutex
	missing map[string]time.Time // file -> when the registry answered 404
}

// Get returns the WebP thumbnail of name@version; small selects the
// gallery size. name and version must already be validated (they become
// a file name).
func (t *Thumbnails) Get(ctx context.Context, name, version string, small bool) ([]byte, error) {
	file := name + "-" + version
	if small {
		file += "-small"
	}
	file += ".webp"
	if t.CacheDir != "" {
		if data, err := os.ReadFile(filepath.Join(t.CacheDir, file)); err == nil {
			return data, nil
		}
	}
	t.mu.Lock()
	at, gone := t.missing[file]
	t.mu.Unlock()
	if gone && time.Since(at) < missingTTL {
		return nil, ErrNoThumbnail
	}
	v, err, _ := t.group.Do(file, func() (any, error) { return t.fetch(ctx, file) })
	if errors.Is(err, ErrNoThumbnail) {
		t.mu.Lock()
		// Names come from requests: keep the map from growing without bound.
		if t.missing == nil || len(t.missing) >= 10000 {
			t.missing = map[string]time.Time{}
		}
		t.missing[file] = time.Now()
		t.mu.Unlock()
	}
	if err != nil {
		return nil, err
	}
	return v.([]byte), nil
}

func (t *Thumbnails) fetch(ctx context.Context, file string) ([]byte, error) {
	base := t.BaseURL
	if base == "" {
		base = DefaultThumbnailURL
	}
	client := t.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/"+file, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil, ErrNoThumbnail
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, maxThumbnailBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxThumbnailBytes {
		return nil, errors.New("ảnh xem trước quá lớn")
	}
	if t.CacheDir != "" {
		_ = writeAtomic(filepath.Join(t.CacheDir, file), data) // a published version's image never changes
	}
	return data, nil
}
