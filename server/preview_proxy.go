package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

const previewPathPrefix = "/preview/"

var errPreviewNotReady = errors.New("preview server is not ready yet")

// previewReverseProxy builds a fresh reverse proxy to tinymist's preview
// HTTP+WS server (started via lsp.PreviewService, listening on
// 127.0.0.1:<random port>). Built per-request since the target address can
// change across preview restarts.
func (s *Server) previewReverseProxy() (*httputil.ReverseProxy, error) {
	previewSrv := s.appSrv.PreviewService()
	if previewSrv == nil {
		return nil, errPreviewNotReady
	}
	target := previewSrv.Address()
	if target == "" {
		return nil, errPreviewNotReady
	}

	targetURL, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	baseDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		baseDirector(req)
		// tinymist's preview server only binds to 127.0.0.1 and rejects
		// (silently closes) requests whose Host header isn't a local
		// address, as anti-DNS-rebinding protection -- a common pattern for
		// localhost-only dev servers. httputil.ReverseProxy's default
		// Director rewrites req.URL for dialing but leaves the Host header
		// as whatever the public-facing request had (e.g. typst.dsc.edu.vn),
		// which tinymist then rejects with an immediate connection close
		// (surfaces here as "proxy error: EOF", and in the browser as a
		// WebSocket connect/close loop -- reproduced live in production).
		req.Host = targetURL.Host
		// Real browsers also send an Origin header on WebSocket upgrades
		// (curl doesn't unless told to, which is why this half of the fix
		// wasn't caught by a plain curl test). tinymist checks Origin
		// too -- confirmed by reproducing the exact "502 Bad Gateway"
		// production failure locally once Origin was spoofed to
		// https://typst.dsc.edu.vn alongside Host. Rewrite it to match the
		// local target so tinymist's same-origin check passes.
		if req.Header.Get("Origin") != "" {
			req.Header.Set("Origin", "http://"+targetURL.Host)
		}
	}
	proxy.ModifyResponse = injectBaseHref
	return proxy, nil
}

// handlePreviewProxy reverse-proxies tinymist's preview page and its static
// assets under /preview/. Go's httputil.ReverseProxy transparently proxies
// WebSocket upgrades too, but (see handlePreviewRootWebSocket below)
// tinymist's own preview page does NOT actually open its WebSocket against
// this prefixed path, so that part of the generic WS support goes unused
// here -- it's still correct to leave it in place for any GET/asset traffic
// that isn't a WS upgrade.
func (s *Server) handlePreviewProxy(w http.ResponseWriter, r *http.Request) {
	proxy, err := s.previewReverseProxy()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	baseDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		baseDirector(req)
		req.URL.Path = strings.TrimPrefix(req.URL.Path, strings.TrimSuffix(previewPathPrefix, "/"))
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
	}
	proxy.ServeHTTP(w, r)
}

// handlePreviewRootWebSocket proxies tinymist's preview WebSocket. Verified
// live against a real tinymist build (see risk #2 in
// docs/plans/plan_web_version.md): its preview page's JS always opens its
// WebSocket against "ws://<origin>/" -- the ORIGIN ROOT -- regardless of the
// path the page itself was loaded from (it is not derived from
// location.pathname/location.href as originally assumed). So on top of the
// /preview/-prefixed proxy above (for the HTML page + its static assets),
// WebSocket upgrade requests hitting the server root must ALSO be routed to
// tinymist, unprefixed. It returns true if it handled the request.
func (s *Server) handlePreviewRootWebSocket(w http.ResponseWriter, r *http.Request) bool {
	if !isWebSocketUpgrade(r) {
		return false
	}

	// This path is mounted outside s.handle (it shares "/" with the static
	// frontend), so it must do its own auth: without it anyone could open
	// the live preview socket and read the open document. The Origin check
	// stops other sites from riding the user's cookie.
	if s.auth.enabled() && !s.auth.validRequest(r) {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return true
	}
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "cross-origin websocket rejected")
		return true
	}

	proxy, err := s.previewReverseProxy()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return true
	}
	proxy.ServeHTTP(w, r)
	return true
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

func injectBaseHref(resp *http.Response) error {
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	marker := []byte("<head>")
	inject := []byte(`<head><base href="` + previewPathPrefix + `">`)
	if bytes.Contains(body, marker) {
		body = bytes.Replace(body, marker, inject, 1)
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	return nil
}

func (s *Server) handlePreviewStatus(w http.ResponseWriter, r *http.Request) {
	previewSrv := s.appSrv.PreviewService()
	ready := previewSrv != nil && previewSrv.Address() != ""
	writeJSON(w, http.StatusOK, map[string]bool{"ready": ready})
}

type restartPreviewRequest struct {
	EntryFile string `json:"entryFile"`
}

func (s *Server) handlePreviewRestart(w http.ResponseWriter, r *http.Request) {
	var req restartPreviewRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // entryFile is optional

	entryFile := req.EntryFile
	if entryFile != "" {
		root, err := s.projectRoot()
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		abs, err := resolveInRoot(root, entryFile)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		entryFile = abs // tinymist requires an absolute path here
	}

	s.appSrv.RestartPreviewWithEntry(r.Context(), entryFile, nil)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type cursorPositionRequest struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// handlePreviewCursor forwards an editor cursor-position change to tinymist's
// preview server, the same call the desktop editor makes on every selection
// change (see PreviewService.ScrollOnSelectionChange). This drives the "Đồng
// bộ chính xác" (tinymist-native) view mode in the web frontend, which embeds
// /preview/ directly rather than re-deriving page positions heuristically.
func (s *Server) handlePreviewCursor(w http.ResponseWriter, r *http.Request) {
	var req cursorPositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	previewSrv := s.appSrv.PreviewService()
	if previewSrv == nil {
		writeError(w, http.StatusServiceUnavailable, errPreviewNotReady.Error())
		return
	}

	previewSrv.ScrollOnSelectionChange(r.Context())
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
