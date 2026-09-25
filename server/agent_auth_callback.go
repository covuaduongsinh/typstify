package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// OAuth "loopback" logins (Antigravity's "Log in with Google", Gemini CLI,
// ...) make the agent listen on 127.0.0.1:<port> and ask Google to redirect
// the browser there with ?code=...&state=... . In this deployment the agent
// runs inside the server's container, so that redirect hits 127.0.0.1 on
// the *user's* machine and fails (ERR_CONNECTION_REFUSED) -- the code never
// reaches the agent and handleAgentAuth waits forever.
//
// handleAgentAuthCallback completes the flow: the user pastes the failed
// page's address, and the server -- which shares the agent's loopback
// interface -- performs that exact request.

const agentCallbackTimeout = 15 * time.Second

var redirectURIParam = regexp.MustCompile(`redirect_uri=([^&\s"'<>]+)`)

var (
	errCallbackFormat  = errors.New("Địa chỉ không đúng dạng. Hãy dán toàn bộ địa chỉ trang lỗi, bắt đầu bằng http://127.0.0.1:…")
	errCallbackNoCode  = errors.New("Địa chỉ thiếu tham số code. Hãy dán đúng địa chỉ trình duyệt mở ra sau khi đăng nhập Google.")
	errCallbackNoLogin = errors.New("Chưa thấy liên kết đăng nhập của trợ lý AI. Hãy bấm Đăng nhập và mở liên kết trước.")
	errCallbackPort    = errors.New("Địa chỉ không khớp với liên kết đăng nhập đang chờ. Hãy đăng nhập lại và dán địa chỉ mới nhất.")
)

// isLoopbackHost reports whether host (no port) names this machine.
func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// loginRedirectTargets returns the loopback host:port pairs named by
// redirect_uri parameters in the agent's console output (its login URL).
func loginRedirectTargets(consoleText string) map[string]bool {
	targets := map[string]bool{}
	for _, m := range redirectURIParam.FindAllStringSubmatch(consoleText, -1) {
		raw, err := url.QueryUnescape(m[1])
		if err != nil {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil || u.Port() == "" || !isLoopbackHost(u.Hostname()) {
			continue
		}
		targets[u.Port()] = true
	}
	return targets
}

// allowedCallbackTarget validates a pasted callback address. It must be a
// plain-http loopback URL carrying an OAuth code/error, on a port the
// agent's own login URL redirects to -- so this endpoint can only complete
// the login in progress, never reach other local services.
func allowedCallbackTarget(consoleText, raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "http" || u.Port() == "" || !isLoopbackHost(u.Hostname()) {
		return nil, errCallbackFormat
	}
	q := u.Query()
	if q.Get("code") == "" && q.Get("error") == "" {
		return nil, errCallbackNoCode
	}
	targets := loginRedirectTargets(consoleText)
	if len(targets) == 0 {
		return nil, errCallbackNoLogin
	}
	if !targets[u.Port()] {
		return nil, errCallbackPort
	}
	// Always talk to the IPv4 loopback the agent listens on.
	u.Host = net.JoinHostPort("127.0.0.1", u.Port())
	u.User = nil
	u.Fragment = ""
	return u, nil
}

type agentAuthCallbackRequest struct {
	URL string `json:"url"`
}

var htmlTag = regexp.MustCompile(`<[^>]*>`)

func (s *Server) handleAgentAuthCallback(w http.ResponseWriter, r *http.Request) {
	if s.authInProgress.Load() <= 0 {
		writeError(w, http.StatusConflict, "Không có lượt đăng nhập nào đang chờ. Hãy bấm Đăng nhập trước.")
		return
	}

	var req agentAuthCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}

	target, err := allowedCallbackTarget(s.consoleText(), req.URL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	client := &http.Client{
		Timeout: agentCallbackTimeout,
		// The agent's page may redirect to a hosted "you can close this
		// tab" page; that belongs to the browser, not to us.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Get(target.String())
	if err != nil {
		writeError(w, http.StatusBadGateway,
			"Trợ lý AI không nhận mã đăng nhập (có thể đã hết hạn). Hãy bấm Đăng nhập lại. Chi tiết: "+err.Error())
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	message := strings.Join(strings.Fields(htmlTag.ReplaceAllString(string(body), " ")), " ")
	if len(message) > 500 {
		message = message[:500] + "…"
	}

	if resp.StatusCode >= 400 {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Trợ lý AI báo lỗi %d: %s", resp.StatusCode, message))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": resp.StatusCode, "message": message})
}

// consoleText is the agent/app console log, where agents print login URLs.
func (s *Server) consoleText() string {
	if s.consoleTextFn != nil {
		return s.consoleTextFn()
	}
	return s.appSrv.Console().Text()
}
