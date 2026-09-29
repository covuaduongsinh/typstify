// Package remote is a small HTTP client for a self-hosted Typstify web
// server (cmd/typstify-server), used by the desktop app to talk to a
// Typstify instance running on a different machine -- settings sync today,
// and the foundation for remote-agent (Giai đoạn B) and remote-project
// (Giai đoạn C) modes later. It is deliberately thin: it reuses
// service/net's HttpInvoker for the transport instead of a second HTTP
// client implementation.
package remote

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	typstnet "looz.ws/typstify/service/net"
)

// ErrTokenInvalid is returned when the server rejects the configured
// Bearer token (missing, expired, or revoked) -- callers should prompt the
// user to reconnect (RemoteSettings.Token needs to be replaced via Login).
var ErrTokenInvalid = errors.New("remote: token is invalid or has been revoked")

type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewClient builds a client for the server at baseURL (e.g.
// "https://typstify.example.com"), authenticating with token (obtained via
// Login and persisted in settings.RemoteSettings.Token). token may be
// empty for calling Login itself.
func NewClient(baseURL, token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    baseURL,
		token:      token,
	}
}

func (c *Client) invoker(path string) *typstnet.HttpInvoker {
	addr, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		addr = c.baseURL + path
	}
	invoker := typstnet.NewInvoker(c.httpClient, addr)
	invoker.SetAuthorization(c.token)
	return invoker
}

type loginRequest struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password"`
	Label    string `json:"label,omitempty"`
}

// Login exchanges username+password (username may be empty for a
// single-admin-password server) for a long-lived Bearer token via
// POST /api/auth/token. It does not mutate this Client's own token -- the
// caller decides whether/how to persist the returned token (typically into
// settings.RemoteSettings.Token) and construct a new authenticated Client
// with it.
func (c *Client) Login(username, password, label string) (string, error) {
	var resp struct {
		Token string `json:"token"`
	}
	invoker := typstnet.NewInvoker(c.httpClient, mustJoin(c.baseURL, "/api/auth/token"))
	if err := invoker.Post(loginRequest{Username: username, Password: password, Label: label}).Decode(&resp); err != nil {
		return "", translateErr(err)
	}
	return resp.Token, nil
}

// GetSettingsMeta fetches when each settings section was last written on
// the remote server -- see service/settings.Settings.Meta for the local
// equivalent this is compared against.
func (c *Client) GetSettingsMeta() (map[string]time.Time, error) {
	var meta map[string]time.Time
	if err := c.get("/api/settings/meta", &meta); err != nil {
		return nil, err
	}
	return meta, nil
}

// GetSettings fetches one settings section's current JSON representation.
// out is typically a pointer to the same settings.Model type as the
// section (e.g. *settings.GeneralSettings), or a json.RawMessage if the
// caller wants to pass it through to that model's ApplyRemote unchanged.
func (c *Client) GetSettings(section string, out any) error {
	return c.get("/api/settings/"+section, out)
}

// PutSettings pushes this machine's local value of a settings section to
// the remote server, which will persist it stamped with its own "now" --
// correct, since from the remote's point of view this is an ordinary
// local write.
func (c *Client) PutSettings(section string, in any) error {
	if err := c.invoker("/api/settings/" + section).Put(in).Decode(nil); err != nil {
		return translateErr(err)
	}
	return nil
}

// AgentSessionSummary mirrors server/agent_api.go's acpSessionSummary.
type AgentSessionSummary struct {
	SessionID string `json:"sessionId"`
	Title     string `json:"title"`
	// RFC3339, empty when the agent didn't report one.
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// ListAgentSessions lists past AI Agent sessions for whichever project the
// remote server currently has open, via GET /api/agent/sessions -- used by
// remote-agent mode (Giai đoạn B) to find the most recent session to
// resume instead of always starting a fresh one.
func (c *Client) ListAgentSessions() ([]AgentSessionSummary, error) {
	var sessions []AgentSessionSummary
	if err := c.get("/api/agent/sessions", &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// --- Remote project (Giai đoạn C.1): whichever project the remote server
// currently has open (server/files_api.go's projectRoot -- a single global
// "current project" per server instance, same as what a browser tab
// operates against). OpenProject switches it; Tree/ReadFile/WriteFile/
// CreateFile/DeleteFile/RenameFile all then operate relative to it.

// TreeEntry mirrors server/files_api.go's treeEntry.
type TreeEntry struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"` // slash-separated, relative to the project root
	IsDir   bool      `json:"isDir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

// RemoteProjectSummary mirrors service/workspace.go's WorkspaceState, the
// subset GET /api/workspace/recent actually needs to list projects the
// remote server has open before.
type RemoteProjectSummary struct {
	Path         string    `json:"Path"`
	LastAccessAt time.Time `json:"LastAccessAt"`
}

// CurrentProject returns the path of whichever project the remote server
// currently has open, or "" if none.
func (c *Client) CurrentProject() (string, error) {
	var resp struct {
		Path string `json:"path"`
	}
	if err := c.get("/api/workspace/current", &resp); err != nil {
		return "", err
	}
	return resp.Path, nil
}

// RecentProjects lists projects previously opened on the remote server.
func (c *Client) RecentProjects() ([]RemoteProjectSummary, error) {
	var out []RemoteProjectSummary
	if err := c.get("/api/workspace/recent", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// OpenProject switches the remote server's current project to path
// (absolute, on the REMOTE machine's filesystem -- not this one's),
// creating no new files. Every subsequent Tree/ReadFile/WriteFile/... call
// operates relative to whatever project is current, so this must be
// called (directly, or via CreateProject) before any of them.
func (c *Client) OpenProject(path string) (string, error) {
	var resp struct {
		Path string `json:"path"`
	}
	req := struct {
		Path string `json:"path"`
	}{Path: path}
	if err := c.invoker("/api/workspace/open").Post(req).Decode(&resp); err != nil {
		return "", translateErr(err)
	}
	return resp.Path, nil
}

// CreateProject creates a new project directory (with a starter main.typ)
// at path on the remote machine and makes it the current project.
func (c *Client) CreateProject(path string) (string, error) {
	var resp struct {
		Path string `json:"path"`
	}
	req := struct {
		Path string `json:"path"`
	}{Path: path}
	if err := c.invoker("/api/workspace/create").Post(req).Decode(&resp); err != nil {
		return "", translateErr(err)
	}
	return resp.Path, nil
}

// Tree lists one directory's entries (non-recursive) of the current
// remote project. relPath is slash-separated and relative to the project
// root; "" lists the root itself.
func (c *Client) Tree(relPath string) ([]TreeEntry, error) {
	var entries []TreeEntry
	q := url.Values{}
	if relPath != "" {
		q.Set("path", relPath)
	}
	if err := c.get("/api/workspace/tree?"+q.Encode(), &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// ReadFile fetches a file's raw content from the current remote project.
// Unlike GetSettings/PutSettings this is NOT JSON -- server/files_api.go's
// handleFileGet/handleFilePut read and write the request/response body
// as-is, so this bypasses HttpInvoker (which always JSON-encodes) for a
// plain HTTP request instead.
func (c *Client) ReadFile(relPath string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.fileURL(relPath), nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, translateErr(typstnet.NetworkError{StatusCode: resp.StatusCode})
	}
	return io.ReadAll(resp.Body)
}

// WriteFile overwrites a file's full content in the current remote
// project (creating parent directories as needed, matching
// handleFilePut). See ReadFile for why this bypasses HttpInvoker.
func (c *Client) WriteFile(relPath string, content []byte) error {
	req, err := http.NewRequest(http.MethodPut, c.fileURL(relPath), bytes.NewReader(content))
	if err != nil {
		return err
	}
	c.authorize(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return translateErr(typstnet.NetworkError{StatusCode: resp.StatusCode})
	}
	return nil
}

func (c *Client) fileURL(relPath string) string {
	q := url.Values{}
	q.Set("path", relPath)
	return mustJoin(c.baseURL, "/api/workspace/file") + "?" + q.Encode()
}

func (c *Client) authorize(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

// CreateFile creates an empty file, or an empty directory when isDir, in
// the current remote project.
func (c *Client) CreateFile(relPath string, isDir bool) error {
	req := struct {
		Path  string `json:"path"`
		IsDir bool   `json:"isDir"`
	}{Path: relPath, IsDir: isDir}
	if err := c.invoker("/api/workspace/file").Post(req).Decode(nil); err != nil {
		return translateErr(err)
	}
	return nil
}

// DeleteFile removes a file or directory (recursively) from the current
// remote project.
func (c *Client) DeleteFile(relPath string) error {
	q := url.Values{}
	q.Set("path", relPath)
	if err := c.invoker("/api/workspace/file?" + q.Encode()).Delete(nil).Decode(nil); err != nil {
		return translateErr(err)
	}
	return nil
}

// RenameFile moves/renames a file or directory within the current remote
// project.
func (c *Client) RenameFile(from, to string) error {
	req := struct {
		From string `json:"from"`
		To   string `json:"to"`
	}{From: from, To: to}
	if err := c.invoker("/api/workspace/rename").Post(req).Decode(nil); err != nil {
		return translateErr(err)
	}
	return nil
}

func (c *Client) get(path string, out any) error {
	if err := c.invoker(path).Get(nil).Decode(out); err != nil {
		return translateErr(err)
	}
	return nil
}

func translateErr(err error) error {
	var netErr typstnet.NetworkError
	if errors.As(err, &netErr) && netErr.StatusCode == http.StatusUnauthorized {
		return ErrTokenInvalid
	}
	return err
}

func mustJoin(base, path string) string {
	addr, err := url.JoinPath(base, path)
	if err != nil {
		return base + path
	}
	return addr
}
