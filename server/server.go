package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"looz.ws/typstify/service"
	"looz.ws/typstify/service/mcp"
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
	// compileSlots bounds concurrent typst compiles from /api/export and
	// /api/preview/pdf: every request spawns a typst process.
	compileSlots chan struct{}
	// authInProgress counts agent logins waiting in handleAgentAuth; the
	// OAuth callback relay only works while one is pending.
	authInProgress atomic.Int32
	// consoleTextFn overrides the console source (tests).
	consoleTextFn func() string

	activeDocMu sync.RWMutex
	activeDoc   mcp.ActiveDocument
}

type webActiveDocProvider struct {
	server *Server
}

func (p *webActiveDocProvider) GetActiveDocument() mcp.ActiveDocument {
	p.server.activeDocMu.RLock()
	doc := p.server.activeDoc
	p.server.activeDocMu.RUnlock()

	if doc.File != "" {
		return doc
	}

	root := p.server.appSrv.CurrentProjectDir()
	if root != "" {
		mainTyp := filepath.Join(root, "main.typ")
		if _, err := os.Stat(mainTyp); err == nil {
			return mcp.ActiveDocument{File: mainTyp}
		}
		entries, _ := os.ReadDir(root)
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".typ") {
				return mcp.ActiveDocument{File: filepath.Join(root, e.Name())}
			}
		}
	}
	return doc
}

func (s *Server) SetActiveFile(absPath string, cursorPos int) {
	s.activeDocMu.Lock()
	defer s.activeDocMu.Unlock()
	s.activeDoc = mcp.ActiveDocument{
		File:      absPath,
		CursorPos: cursorPos,
	}
}

func New(appSrv *service.ServiceFacade, opts Options) *Server {
	var storageDir string
	if appSrv != nil && appSrv.Settings() != nil && appSrv.Settings().General() != nil {
		storageDir = appSrv.Settings().General().RootDir
	}
	s := &Server{
		appSrv: appSrv,
		opts:   opts,
		auth:   newAuthManager(opts.Password, storageDir),
		mux:    http.NewServeMux(),

		compileSlots: make(chan struct{}, maxConcurrentCompiles),
	}
	if s.appSrv != nil {
		s.appSrv.SetViewManager(nil, nil, nil, func() any {
			return &webActiveDocProvider{server: s}
		})
	}
	s.routes()
	return s
}

// Handler returns the root http.Handler to pass to http.Server.
func (s *Server) Handler() http.Handler {
	return withLogging(withBodyLimit(s.mux))
}

func (s *Server) routes() {
	// Unauthenticated / Auth routes.
	s.mux.HandleFunc("GET /api/health", handleHealth)
	s.mux.HandleFunc("POST /api/auth/register", s.auth.handleRegister)
	s.mux.HandleFunc("POST /api/auth/login", s.auth.handleLogin)
	s.mux.HandleFunc("POST /api/auth/logout", s.auth.handleLogout)
	s.mux.HandleFunc("GET /api/auth/status", s.auth.handleStatus)
	s.handle("POST /api/auth/change-password", s.auth.handleChangePassword)
	s.mux.HandleFunc("POST /api/auth/token", s.auth.handleIssueToken)
	s.handle("GET /api/auth/tokens", s.auth.handleListTokens)
	s.handle("DELETE /api/auth/tokens/{hash}", s.auth.handleRevokeToken)
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
	s.handle("POST /api/preview/render", s.handlePreviewRender)
	s.handle("POST /api/preview/anchors", s.handlePreviewAnchors)

	// VPS sync: desktop pushes project files here (see sync_handler.go).
	s.handle("GET /api/sync/manifest", s.handleSyncManifest)
	s.handle("GET /api/sync/pull", s.handleSyncPull)
	s.handle("POST /api/sync/push", s.handleSyncPush)
	s.handle("POST /api/sync/delete", s.handleSyncDelete)

	// Typst package manager (Tpix).
	s.handle("GET /api/packages/search", s.handlePkgSearch)
	s.handle("GET /api/packages/cached", s.handlePkgCached)
	s.handle("GET /api/packages/detail", s.handlePkgDetail)
	s.handle("POST /api/packages/download", s.handlePkgDownload)
	s.handle("POST /api/packages/pull-deps", s.handlePkgPullDeps)

	// Settings.
	s.handle("GET /api/settings/general", settingsGetHandler(s.appSrv.Settings().General))
	s.handle("PUT /api/settings/general", settingsPutHandler(s.appSrv.Settings().General))
	s.handle("GET /api/settings/editor", settingsGetHandler(s.appSrv.Settings().Editor))
	s.handle("PUT /api/settings/editor", settingsPutHandler(s.appSrv.Settings().Editor))
	s.handle("GET /api/settings/typst", settingsGetHandler(s.appSrv.Settings().Typst))
	s.handle("PUT /api/settings/typst", settingsPutHandler(s.appSrv.Settings().Typst))
	s.handle("GET /api/settings/lsp", settingsGetHandler(s.appSrv.Settings().Lsp))
	s.handle("PUT /api/settings/lsp", settingsPutHandler(s.appSrv.Settings().Lsp))
	s.handle("GET /api/settings/agent", settingsGetHandler(s.appSrv.Settings().AcpAgent))
	s.handle("PUT /api/settings/agent", settingsPutHandler(s.appSrv.Settings().AcpAgent))
	s.handle("GET /api/settings/meta", s.handleSettingsMeta)
	s.handle("GET /api/settings/fonts", s.handleFontsList)
	s.handle("POST /api/settings/fonts", s.handleFontUpload)
	s.handle("DELETE /api/settings/fonts/{name}", s.handleFontDelete)
	// No /api/settings/tpix: it holds the package-registry API key, which
	// the web UI never uses, so it is not exposed to the browser at all.
	// No /api/settings/agent in settings-sync scope: AcpAgentSettings is
	// tied to the agent CLI installed on each machine (Cmd/Args/Env/AgentID
	// are machine-local), so it is excluded from the meta-based sync loop
	// even though its GET/PUT routes remain for the web UI.

	// Dropbox sync & integration.
	s.handle("GET /api/dropbox/status", s.handleDropboxStatus)
	s.handle("POST /api/dropbox/auth/url", s.handleDropboxAuthURL)
	s.handle("POST /api/dropbox/auth/callback", s.handleDropboxAuthCallback)
	s.handle("POST /api/dropbox/auth/token", s.handleDropboxSaveToken)
	s.handle("POST /api/dropbox/auth/disconnect", s.handleDropboxDisconnect)
	s.handle("POST /api/dropbox/sync", s.handleDropboxSync)
	s.handle("GET /api/dropbox/projects", s.handleDropboxListProjects)
	s.handle("POST /api/dropbox/import", s.handleDropboxImport)

	// AI agent registry / selection / auth.
	s.handle("GET /api/agent/registry", s.handleAgentRegistry)
	s.handle("POST /api/agent/select", s.handleAgentSelect)
	s.handle("POST /api/agent/auth/callback", s.handleAgentAuthCallback)
	s.handle("POST /api/agent/auth/{methodId}", s.handleAgentAuth)
	s.handle("POST /api/agent/preferred-config", s.handleSavePreferredConfig)
	s.handle("GET /api/agent/sessions", s.handleAgentSessions)
	s.handle("GET /api/console", s.handleConsole)

	// LSP + AI agent, over WebSocket.
	s.handle("GET /ws/lsp", s.handleLspWS)
	s.handle("GET /ws/agent", s.handleAgentWS)

	// Preview: reverse-proxied to tinymist's own preview server.
	s.handle("GET /api/preview/status", s.handlePreviewStatus)
	s.handle("POST /api/preview/restart", s.handlePreviewRestart)
	s.handle("POST /api/preview/cursor", s.handlePreviewCursor)
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
			// Only /assets/* filenames are content-hashed by Vite (a given
			// path's bytes then never change, so caching for a year is
			// safe); root-level files copied verbatim from web/public
			// (favicon.svg, icons.svg) keep their name across a rebuild and
			// must stay revalidated instead. index.html (served below, and
			// via the SPA fallback for client-routed paths) always needs a
			// fresh fetch too, or a redeployed build could keep serving a
			// stale shell referencing assets that no longer exist.
			if strings.HasPrefix(r.URL.Path, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			fileServer.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
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
