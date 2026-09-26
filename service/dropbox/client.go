package dropbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"looz.ws/typstify/service/settings"
)

const (
	defaultAPIHost     = "https://api.dropboxapi.com"
	defaultContentHost = "https://content.dropboxapi.com"
	defaultOAuthHost   = "https://api.dropbox.com"
)

type AccountInfo struct {
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

type FileMetadata struct {
	Tag            string    `json:".tag"` // "file" or "folder"
	Name           string    `json:"name"`
	PathLower      string    `json:"path_lower"`
	PathDisplay    string    `json:"path_display"`
	ID             string    `json:"id"`
	ClientModified time.Time `json:"client_modified"`
	ServerModified time.Time `json:"server_modified"`
	Size           int64     `json:"size"`
	ContentHash    string    `json:"content_hash"`
}

type ListFolderResult struct {
	Entries []FileMetadata `json:"entries"`
	Cursor  string         `json:"cursor"`
	HasMore bool           `json:"has_more"`
}

type Client struct {
	httpClient  *http.Client
	settings    *settings.DropboxSettings
	apiHost     string
	contentHost string
	oauthHost   string
	mu          sync.Mutex
}

func NewClient(st *settings.DropboxSettings) *Client {
	return &Client{
		httpClient:  &http.Client{Timeout: 60 * time.Second},
		settings:    st,
		apiHost:     defaultAPIHost,
		contentHost: defaultContentHost,
		oauthHost:   defaultOAuthHost,
	}
}

// SetHosts overrides the default endpoints for testing.
func (c *Client) SetHosts(apiHost, contentHost, oauthHost string) {
	c.apiHost = apiHost
	c.contentHost = contentHost
	c.oauthHost = oauthHost
}

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.settings.AccessToken == "" && c.settings.RefreshToken == "" {
		return "", errors.New("chưa kết nối tài khoản Dropbox")
	}

	// Check if access token is expired or close to expiring (within 1 min)
	if c.settings.RefreshToken != "" && (c.settings.AccessToken == "" || (!c.settings.TokenExpiry.IsZero() && time.Until(c.settings.TokenExpiry) < time.Minute)) {
		if err := c.refreshTokenLocked(ctx); err != nil {
			if c.settings.AccessToken == "" {
				return "", fmt.Errorf("làm mới token thất bại: %w", err)
			}
			// Fallback to existing access token if refresh fails
		}
	}

	return c.settings.AccessToken, nil
}

func (c *Client) refreshTokenLocked(ctx context.Context) error {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", c.settings.RefreshToken)
	if c.settings.AppKey != "" {
		data.Set("client_id", c.settings.AppKey)
	}
	if c.settings.AppSecret != "" {
		data.Set("client_secret", c.settings.AppSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.oauthHost+"/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("Dropbox token refresh HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	c.settings.AccessToken = tokenResp.AccessToken
	if tokenResp.ExpiresIn > 0 {
		c.settings.TokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	_ = c.settings.Save()
	return nil
}

// ExchangeAuthCode exchanges authorization code for access & refresh tokens.
func (c *Client) ExchangeAuthCode(ctx context.Context, code, redirectURI, appKey, appSecret string) (*AccountInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	if redirectURI != "" {
		data.Set("redirect_uri", redirectURI)
	}
	if appKey != "" {
		data.Set("client_id", appKey)
	}
	if appSecret != "" {
		data.Set("client_secret", appSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.oauthHost+"/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("đổi mã xác thực thất bại (%d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		AccountID    string `json:"account_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	c.settings.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		c.settings.RefreshToken = tokenResp.RefreshToken
	}
	if tokenResp.ExpiresIn > 0 {
		c.settings.TokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	if appKey != "" {
		c.settings.AppKey = appKey
	}
	if appSecret != "" {
		c.settings.AppSecret = appSecret
	}

	// Fetch account details
	c.mu.Unlock()
	acc, err := c.GetCurrentAccount(ctx)
	c.mu.Lock()
	if err == nil && acc != nil {
		c.settings.AccountID = acc.AccountID
		c.settings.AccountName = acc.DisplayName
		c.settings.AccountEmail = acc.Email
	}
	_ = c.settings.Save()

	return acc, nil
}

// GetCurrentAccount fetches the logged-in user profile.
func (c *Client) GetCurrentAccount(ctx context.Context) (*AccountInfo, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiHost+"/2/users/get_current_account", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("lấy thông tin tài khoản thất bại (%d): %s", resp.StatusCode, string(body))
	}

	var raw struct {
		AccountID string `json:"account_id"`
		Name      struct {
			DisplayName string `json:"display_name"`
		} `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	return &AccountInfo{
		AccountID:   raw.AccountID,
		DisplayName: raw.Name.DisplayName,
		Email:       raw.Email,
	}, nil
}

// ListFolder lists all files and folders recursively under remotePath.
func (c *Client) ListFolder(ctx context.Context, remotePath string, recursive bool) ([]FileMetadata, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	// Normalize root path: Dropbox expects "" for root folder
	pathArg := remotePath
	if pathArg == "/" || pathArg == "" {
		pathArg = ""
	} else if !strings.HasPrefix(pathArg, "/") {
		pathArg = "/" + pathArg
	}

	payload, _ := json.Marshal(map[string]any{
		"path":                                pathArg,
		"recursive":                           recursive,
		"include_media_info":                  false,
		"include_deleted":                     false,
		"include_has_explicit_shared_members": false,
		"include_mounted_folders":             true,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiHost+"/2/files/list_folder", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == 409 {
		// Folder does not exist yet; treat as empty
		return []FileMetadata{}, nil
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("list_folder (%d): %s", resp.StatusCode, string(body))
	}

	var res ListFolderResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	entries := res.Entries

	// Handle pagination if has_more
	cursor := res.Cursor
	for res.HasMore && cursor != "" {
		contPayload, _ := json.Marshal(map[string]string{"cursor": cursor})
		contReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiHost+"/2/files/list_folder/continue", bytes.NewReader(contPayload))
		if err != nil {
			break
		}
		contReq.Header.Set("Authorization", "Bearer "+token)
		contReq.Header.Set("Content-Type", "application/json")

		contResp, err := c.httpClient.Do(contReq)
		if err != nil {
			break
		}
		if contResp.StatusCode >= 400 {
			contResp.Body.Close()
			break
		}
		var contRes ListFolderResult
		if err := json.NewDecoder(contResp.Body).Decode(&contRes); err != nil {
			contResp.Body.Close()
			break
		}
		contResp.Body.Close()
		entries = append(entries, contRes.Entries...)
		cursor = contRes.Cursor
		res.HasMore = contRes.HasMore
	}

	return entries, nil
}

// UploadFile uploads content to remotePath.
func (c *Client) UploadFile(ctx context.Context, remotePath string, content io.Reader, clientModTime time.Time) (*FileMetadata, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(remotePath, "/") {
		remotePath = "/" + remotePath
	}

	argMap := map[string]any{
		"path":            remotePath,
		"mode":            "overwrite",
		"autorename":      false,
		"mute":            false,
		"strict_conflict": false,
	}
	if !clientModTime.IsZero() {
		argMap["client_modified"] = clientModTime.UTC().Format("2006-01-02T15:04:05Z")
	}

	argJSON, _ := json.Marshal(argMap)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.contentHost+"/2/files/upload", content)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Dropbox-API-Arg", string(argJSON))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("upload (%d): %s", resp.StatusCode, string(body))
	}

	var meta FileMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// DownloadFile downloads the remote file at remotePath.
func (c *Client) DownloadFile(ctx context.Context, remotePath string) (io.ReadCloser, *FileMetadata, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, nil, err
	}

	if !strings.HasPrefix(remotePath, "/") {
		remotePath = "/" + remotePath
	}

	argJSON, _ := json.Marshal(map[string]string{"path": remotePath})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.contentHost+"/2/files/download", nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Dropbox-API-Arg", string(argJSON))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, nil, fmt.Errorf("download (%d): %s", resp.StatusCode, string(body))
	}

	var meta FileMetadata
	if apiResult := resp.Header.Get("Dropbox-API-Result"); apiResult != "" {
		_ = json.Unmarshal([]byte(apiResult), &meta)
	}

	return resp.Body, &meta, nil
}

// CreateFolder creates a folder on Dropbox.
func (c *Client) CreateFolder(ctx context.Context, remotePath string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(remotePath, "/") {
		remotePath = "/" + remotePath
	}

	payload, _ := json.Marshal(map[string]any{
		"path":       remotePath,
		"autorename": false,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiHost+"/2/files/create_folder_v2", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 409 {
		// Already exists, ignore
		return nil
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("create_folder (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Delete deletes a file or folder on Dropbox.
func (c *Client) Delete(ctx context.Context, remotePath string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(remotePath, "/") {
		remotePath = "/" + remotePath
	}

	payload, _ := json.Marshal(map[string]string{"path": remotePath})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiHost+"/2/files/delete_v2", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 409 || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("delete (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}
