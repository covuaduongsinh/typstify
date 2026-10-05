// Package vpssync đẩy file của project đang mở từ máy local lên server
// typstify-server (VPS) qua REST API với Bearer token. Chỉ hướng Local -> VPS:
// file bị xoá local không kéo theo xoá trên VPS trong PerformSync (chỉ xoá khi
// sự kiện xoá file đi qua DeleteFile).
//
// Không import Gio: package này dùng chung với cmd/typstify-server.
package vpssync

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"looz.ws/typstify/service/settings"
)

// defaultIgnoredDirs là các thư mục luôn bị bỏ qua khi quét, cả phía client
// lẫn server (.typstify chứa token và cấu hình máy, không được đưa lên VPS).
var defaultIgnoredDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"dist":         true,
	".tmp":         true,
	".typstify":    true,
}

var (
	errSyncInProgress = errors.New("đang đồng bộ, vui lòng đợi")
	errNoServerURL    = errors.New("chưa cấu hình Server URL")
	errNoProject      = errors.New("chưa mở dự án")
)

// FileItem là một dòng của manifest: đường dẫn tương đối (dùng '/'), sha256
// nội dung và metadata. Server và client cùng tính hash theo cách này nên
// so sánh trực tiếp được.
type FileItem struct {
	Path    string    `json:"path"`
	Hash    string    `json:"hash"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

// SyncResult là kết quả một lần PerformSync.
type SyncResult struct {
	Uploaded   []string  `json:"uploaded"`
	Errors     []string  `json:"errors"`
	DurationMs int64     `json:"duration_ms"`
	Timestamp  time.Time `json:"timestamp"`
}

// Status là trạng thái hiển thị ở UI (icon Synced / Syncing / Error).
type Status struct {
	Syncing      bool        `json:"syncing"`
	LastSyncTime time.Time   `json:"last_sync_time"`
	LastError    string      `json:"last_error"`
	LastResult   *SyncResult `json:"last_result,omitempty"`
}

// IsDefaultIgnored báo rel (đường dẫn tương đối, '/') có nằm trong thư mục
// bị bỏ qua cố định hay không.
func IsDefaultIgnored(rel string) bool {
	for _, seg := range strings.Split(filepath.ToSlash(rel), "/") {
		if defaultIgnoredDirs[seg] {
			return true
		}
	}
	return false
}

// IsIgnored khớp rel với các pattern do người dùng cấu hình (glob, so với
// đường dẫn đầy đủ hoặc tên file) và với danh sách bỏ qua cố định.
func IsIgnored(rel string, patterns []string) bool {
	if IsDefaultIgnored(rel) {
		return true
	}
	rel = filepath.ToSlash(rel)
	for _, pat := range patterns {
		pat = strings.TrimSpace(pat)
		if pat == "" {
			continue
		}
		if ok, _ := path.Match(pat, rel); ok {
			return true
		}
		if ok, _ := path.Match(pat, path.Base(rel)); ok {
			return true
		}
	}
	return false
}

// ScanManifest quét đệ quy root và trả về mọi file thường (bỏ qua thư mục
// trong defaultIgnoredDirs, symlink và thư mục). Dùng chung cho server
// (/api/sync/manifest) và client (manifest local).
func ScanManifest(root string) ([]FileItem, error) {
	items := []FileItem{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == root {
				return err
			}
			// Bỏ qua mục không đọc được thay vì huỷ cả lần quét.
			return nil
		}
		if d.IsDir() {
			if p != root && defaultIgnoredDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		hash, err := hashFile(p)
		if err != nil {
			return nil
		}
		items = append(items, FileItem{
			Path:    filepath.ToSlash(rel),
			Hash:    hash,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
		return nil
	})
	return items, err
}

func hashFile(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SyncEngine giữ cấu hình kết nối và trạng thái đồng bộ. settings và
// projectDir được truyền dưới dạng hàm để luôn đọc giá trị hiện tại (settings
// được nạp lại từ đĩa mỗi lần truy cập, như các section khác).
type SyncEngine struct {
	settings   func() *settings.VPSSyncSettings
	projectDir func() string
	client     *http.Client

	mu      sync.Mutex
	syncing bool
	status  Status
}

// NewSyncEngine tạo engine. projectDir trả về thư mục project đang mở ("" nếu
// chưa mở).
func NewSyncEngine(settingsFn func() *settings.VPSSyncSettings, projectDir func() string) *SyncEngine {
	return &SyncEngine{
		settings:   settingsFn,
		projectDir: projectDir,
		client:     &http.Client{Timeout: 2 * time.Minute},
	}
}

// Status trả về bản sao trạng thái hiện tại.
func (e *SyncEngine) Status() Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	st := e.status
	st.Syncing = e.syncing
	return st
}

// newRequest dựng request tới server, gắn Bearer token nếu có.
func (e *SyncEngine) newRequest(ctx context.Context, method, endpoint string, query url.Values, body io.Reader) (*http.Request, error) {
	base := strings.TrimRight(strings.TrimSpace(e.settings().ServerURL), "/")
	if base == "" {
		return nil, errNoServerURL
	}
	u, err := url.Parse(base + endpoint)
	if err != nil {
		return nil, fmt.Errorf("Server URL không hợp lệ: %w", err)
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if token := e.settings().Token; token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}

// do gửi request và trả về lỗi có nội dung từ server khi status không thành công.
func (e *SyncEngine) do(req *http.Request) ([]byte, error) {
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(data))
		var apiErr struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &apiErr) == nil && apiErr.Error != "" {
			msg = apiErr.Error
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("token không hợp lệ hoặc đã hết hạn (401): %s", msg)
		}
		return nil, fmt.Errorf("server trả về %d: %s", resp.StatusCode, msg)
	}
	return data, nil
}

// fetchManifest lấy manifest của VPS, đánh index theo đường dẫn.
func (e *SyncEngine) fetchManifest(ctx context.Context) (map[string]FileItem, error) {
	req, err := e.newRequest(ctx, http.MethodGet, "/api/sync/manifest", nil, nil)
	if err != nil {
		return nil, err
	}
	data, err := e.do(req)
	if err != nil {
		return nil, err
	}
	var items []FileItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("manifest từ server không đúng định dạng: %w", err)
	}
	m := make(map[string]FileItem, len(items))
	for _, it := range items {
		m[it.Path] = it
	}
	return m, nil
}

// TestConnection kiểm tra Server URL và Token bằng cách gọi manifest.
func (e *SyncEngine) TestConnection(ctx context.Context) error {
	_, err := e.fetchManifest(ctx)
	return err
}

// PushFile tải một file local (rel tương đối với project) lên VPS.
func (e *SyncEngine) PushFile(ctx context.Context, rel string) error {
	root := e.projectDir()
	if root == "" {
		return errNoProject
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return err
	}

	req, err := e.newRequest(ctx, http.MethodPost, "/api/sync/push",
		url.Values{"path": {rel}}, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	_, err = e.do(req)
	return err
}

// PullFile tải một file từ VPS về máy local (ghi đè hoặc tạo mới).
func (e *SyncEngine) PullFile(ctx context.Context, rel string) error {
	root := e.projectDir()
	if root == "" {
		return errNoProject
	}
	req, err := e.newRequest(ctx, http.MethodGet, "/api/sync/pull",
		url.Values{"path": {rel}}, nil)
	if err != nil {
		return err
	}
	data, err := e.do(req)
	if err != nil {
		return err
	}

	dest := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0644)
}

// DeleteFile xoá rel trên VPS (dùng khi file bị xoá ở local).
func (e *SyncEngine) DeleteFile(ctx context.Context, rel string) error {
	body, err := json.Marshal(map[string]string{"path": rel})
	if err != nil {
		return err
	}
	req, err := e.newRequest(ctx, http.MethodPost, "/api/sync/delete", nil, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	_, err = e.do(req)
	return err
}

// PerformSync thực hiện đồng bộ hai chiều (Local <-> VPS):
// - Đẩy file local mới/đổi lên VPS.
// - Tải file VPS mới/đổi về máy local.
func (e *SyncEngine) PerformSync(ctx context.Context) (*SyncResult, error) {
	e.mu.Lock()
	if e.syncing {
		e.mu.Unlock()
		return nil, errSyncInProgress
	}
	e.syncing = true
	e.mu.Unlock()

	start := time.Now()
	res, err := e.performSync(ctx, start)

	e.mu.Lock()
	e.syncing = false
	e.status.LastResult = res
	if err != nil {
		e.status.LastError = err.Error()
	} else {
		e.status.LastError = ""
		e.status.LastSyncTime = time.Now()
	}
	e.mu.Unlock()
	return res, err
}

func (e *SyncEngine) performSync(ctx context.Context, start time.Time) (*SyncResult, error) {
	root := e.projectDir()
	if root == "" {
		return nil, errNoProject
	}
	patterns := e.settings().IgnorePatterns

	local, err := ScanManifest(root)
	if err != nil {
		return nil, err
	}
	remote, err := e.fetchManifest(ctx)
	if err != nil {
		return nil, err
	}

	localMap := make(map[string]FileItem, len(local))
	for _, f := range local {
		localMap[f.Path] = f
	}

	res := &SyncResult{Uploaded: []string{}, Errors: []string{}}

	// 1. Chiều Local -> VPS (Push các file local mới hoặc sửa sau)
	for _, f := range local {
		if IsIgnored(f.Path, patterns) {
			continue
		}
		r, existsOnRemote := remote[f.Path]
		if existsOnRemote && r.Hash == f.Hash {
			continue
		}
		// Nếu remote chưa có HOẶC local mới hơn remote
		if !existsOnRemote || f.ModTime.After(r.ModTime) {
			if err := e.PushFile(ctx, f.Path); err != nil {
				res.Errors = append(res.Errors, "push "+f.Path+": "+err.Error())
				continue
			}
			res.Uploaded = append(res.Uploaded, f.Path)
		}
	}

	// 2. Chiều VPS -> Local (Pull các file remote mới hoặc sửa sau)
	for rPath, r := range remote {
		if IsIgnored(rPath, patterns) {
			continue
		}
		l, existsOnLocal := localMap[rPath]
		if existsOnLocal && l.Hash == r.Hash {
			continue
		}
		// Nếu local chưa có HOẶC remote mới hơn local
		if !existsOnLocal || r.ModTime.After(l.ModTime) {
			if err := e.PullFile(ctx, rPath); err != nil {
				res.Errors = append(res.Errors, "pull "+rPath+": "+err.Error())
				continue
			}
			res.Uploaded = append(res.Uploaded, rPath+" (downloaded)")
		}
	}

	res.DurationMs = time.Since(start).Milliseconds()
	res.Timestamp = time.Now()
	return res, nil
}

// OnFileChanged được gọi khi một file đang mở bị ghi (từ event bus). Khi
// AutoSync bật, đẩy file đó lên VPS trong nền. Không chặn caller.
func (e *SyncEngine) OnFileChanged(absPath string) {
	cfg := e.settings()
	if !cfg.Enabled || !cfg.AutoSync {
		return
	}
	root := e.projectDir()
	if root == "" {
		return
	}
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return
	}
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "../") || rel == ".." || IsIgnored(rel, cfg.IgnorePatterns) {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		var err error
		if _, statErr := os.Stat(absPath); os.IsNotExist(statErr) {
			err = e.DeleteFile(ctx, rel)
		} else {
			err = e.PushFile(ctx, rel)
		}

		e.mu.Lock()
		defer e.mu.Unlock()
		if err != nil {
			e.status.LastError = err.Error()
			return
		}
		e.status.LastError = ""
		e.status.LastSyncTime = time.Now()
	}()
}

// LoginWithPassword đăng nhập với Server Password (hoặc username+password) để lấy Bearer token từ server.
func LoginWithPassword(ctx context.Context, serverURL, username, password string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(serverURL), "/")
	if base == "" {
		return "", errNoServerURL
	}
	body, err := json.Marshal(map[string]string{
		"username": username,
		"password": password,
		"label":    "typstify-desktop-sync",
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/auth/token", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("không thể kết nối tới server: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var apiErr struct {
			Error string `json:"error"`
		}
		msg := strings.TrimSpace(string(data))
		if json.Unmarshal(data, &apiErr) == nil && apiErr.Error != "" {
			msg = apiErr.Error
		}
		return "", fmt.Errorf("đăng nhập thất bại (%d): %s", resp.StatusCode, msg)
	}

	var res struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &res); err != nil || res.Token == "" {
		return "", fmt.Errorf("server không trả về token hợp lệ")
	}
	return res.Token, nil
}

// StartBackgroundLoop bắt đầu vòng lặp đồng bộ định kỳ theo IntervalSec.
func (e *SyncEngine) StartBackgroundLoop(ctx context.Context) {
	go func() {
		for {
			interval := 300 * time.Second
			cfg := e.settings()
			if cfg.IntervalSec > 0 {
				interval = time.Duration(cfg.IntervalSec) * time.Second
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
				cfg = e.settings()
				if cfg.Enabled && cfg.AutoSync && e.projectDir() != "" {
					syncCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
					e.PerformSync(syncCtx)
					cancel()
				}
			}
		}
	}()
}

