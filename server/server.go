package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"looz.ws/typstify/service"
)

// Options configures a Server.
type Options struct {
	// Password gates every API/WS/preview route behind a login (see auth.go).
	// Empty disables auth entirely -- only acceptable for loopback/dev use.
	Password string
	// StaticDir, when non-empty, serves the built web frontend (web/dist)
	// for any path not matched below, with SPA fallback to index.html.
	StaticDir string
	// ProjectRoot, when non-empty, confines handleOpenProject/
	// handleCreateProject to this directory (or subdirectories of it) --
	// e.g. the Docker image sets it to /data, the only path backed by a
	// persistent volume (see Dockerfile's VOLUME ["/data"]). Opening or
	// creating a project outside of it (e.g. under /app, which is baked
	// into the image layer) silently gets wiped on the next deploy: real
	// user data was lost this way before this guard existed. Empty means
	// unrestricted -- the desktop app and bare (non-Docker) server runs
	// have no such distinction between ephemeral and persistent storage.
	ProjectRoot string
}

// Server is the HTTP/WebSocket API described in Giai doan 1 of
// docs/plans/plan_web_version.md. It is a transport layer only: all actual
// work is delegated to the existing service.ServiceFacade (and, through it,
// lsp.Client and agent.SessionManager), so this package intentionally holds
// no business logic of its own.
type Server struct {
	appSrv *service.ServiceFacade
	opts   Options
	auth   *authManager
	mux    *http.ServeMux
}

func New(appSrv *service.ServiceFacade, opts Options) *Server {
	s := &Server{
		appSrv: appSrv,
		opts:   opts,
		auth:   newAuthManager(opts.Password),
		mux:    http.NewServeMux(),
	}
	s.routes()
	return s
}

// Handler returns the root http.Handler to pass to http.Server.
func (s *Server) Handler() http.Handler {
	return withLogging(s.mux)
}

func (s *Server) routes() {
	// Unauthenticated.
	s.mux.HandleFunc("GET /api/health", handleHealth)
	s.mux.HandleFunc("POST /api/auth/login", s.auth.handleLogin)
	s.mux.HandleFunc("POST /api/auth/logout", s.auth.handleLogout)
	s.mux.HandleFunc("GET /api/auth/status", s.auth.handleStatus)
	s.mux.HandleFunc("POST /api/i18n", s.handleI18n)

	// Workspace / project / file management.
	s.handle("GET /api/workspace/current", s.handleCurrentProject)
	s.handle("GET /api/workspace/recent", s.handleRecentProjects)
	s.handle("POST /api/workspace/open", s.handleOpenProject)
	s.handle("POST /api/workspace/create", s.handleCreateProject)
	s.handle("GET /api/workspace/tree", s.handleTree)
	s.handle("GET /api/workspace/file", s.handleFileGet)
	s.handle("PUT /api/workspace/file", s.handleFilePut)
	s.handle("POST /api/workspace/file", s.handleFileCreate)
	s.handle("DELETE /api/workspace/file", s.handleFileDelete)
	s.handle("POST /api/workspace/rename", s.handleFileRename)

	// Export (PDF/PNG/SVG download).
	s.handle("GET /api/export", s.handleExport)
	// Static-PDF preview, served inline for an <iframe>/<embed> -- see
	// handlePreviewPdf's doc comment for why this exists alongside the
	// WebSocket-based /preview/ proxy below.
	s.handle("GET /api/preview/pdf", s.handlePreviewPdf)

	// Typst package manager (Tpix).
	s.handle("GET /api/packages/search", s.handlePkgSearch)
	s.handle("GET /api/packages/cached", s.handlePkgCached)
	s.handle("GET /api/packages/detail", s.handlePkgDetail)
	s.handle("POST /api/packages/download", s.handlePkgDownload)
	s.handle("POST /api/packages/pull-deps", s.handlePkgPullDeps)

	// Settings.
	s.handle("GET /api/settings/general", settingsGetHandler(s.appSrv.Settings().General))
	s.handle("PUT /api/settings/general", settingsPutHandler(s.appSrv.Settings().General))
	s.handle("GET /api/settings/typst", settingsGetHandler(s.appSrv.Settings().Typst))
	s.handle("PUT /api/settings/typst", settingsPutHandler(s.appSrv.Settings().Typst))
	s.handle("GET /api/settings/lsp", settingsGetHandler(s.appSrv.Settings().Lsp))
	s.handle("PUT /api/settings/lsp", settingsPutHandler(s.appSrv.Settings().Lsp))
	s.handle("GET /api/settings/agent", settingsGetHandler(s.appSrv.Settings().AcpAgent))
	s.handle("PUT /api/settings/agent", settingsPutHandler(s.appSrv.Settings().AcpAgent))
	s.handle("GET /api/settings/tpix", settingsGetHandler(s.appSrv.Settings().Tpix))
	s.handle("PUT /api/settings/tpix", settingsPutHandler(s.appSrv.Settings().Tpix))

	// AI agent registry / selection / auth.
	s.handle("GET /api/agent/registry", s.handleAgentRegistry)
	s.handle("POST /api/agent/select", s.handleAgentSelect)
	s.handle("POST /api/agent/auth/{methodId}", s.handleAgentAuth)
	s.handle("GET /api/console", s.handleConsole)

	// LSP + AI agent, over WebSocket.
	s.handle("GET /ws/lsp", s.handleLspWS)
	s.handle("GET /ws/agent", s.handleAgentWS)

	// Preview: reverse-proxied to tinymist's own preview server.
	s.handle("GET /api/preview/status", s.handlePreviewStatus)
	s.handle("POST /api/preview/restart", s.handlePreviewRestart)
	s.mux.Handle(previewPathPrefix, s.auth.require(s.handlePreviewProxy))

	if s.opts.StaticDir != "" {
		s.mux.Handle("/", s.staticHandler())
	}
}

func (s *Server) handle(pattern string, fn http.HandlerFunc) {
	s.mux.HandleFunc(pattern, s.auth.require(fn))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// staticHandler serves the built frontend, falling back to index.html for
// any path that isn't a real file on disk (client-side routed SPA paths).
//
// It also handles a WebSocket-upgrade special case at the root path: see
// handlePreviewRootWebSocket in preview_proxy.go for why tinymist's preview
// WebSocket needs to be reachable at "/" alongside the SPA.
func (s *Server) staticHandler() http.Handler {
	fileServer := http.FileServer(http.Dir(s.opts.StaticDir))
	indexFile := filepath.Join(s.opts.StaticDir, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.handlePreviewRootWebSocket(w, r) {
			return
		}

		p := filepath.Join(s.opts.StaticDir, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, indexFile)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
