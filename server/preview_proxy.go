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
