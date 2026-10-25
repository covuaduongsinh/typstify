package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"looz.ws/typstify/service"
	"looz.ws/typstify/service/collab"
	"looz.ws/typstify/service/mcp"
	"looz.ws/typstify/service/projects"
	"looz.ws/typstify/service/wasmassets"
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
	// StorageDir overrides where accounts, sessions and the project
	// registry are stored (default: the General settings' RootDir). Tests
	// point it at a temporary directory.
	StorageDir string
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
	// projects maps project directories to owners, members and share
	// links (projects_api.go).
	projects   *projects.Registry
	storageDir string
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

	// Real-time co-editing rooms (collab_ws.go).
	collab *collab.Hub

	// Fonts and package archives for the in-browser compiler (wasm_api.go).
	wasmFonts    *wasmassets.FontIndex
	wasmPackages *wasmassets.Packages
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
	storageDir := opts.StorageDir
	if storageDir == "" && appSrv != nil && appSrv.Settings() != nil && appSrv.Settings().General() != nil {
		storageDir = appSrv.Settings().General().RootDir
	}
	s := &Server{
		appSrv:     appSrv,
		opts:       opts,
		auth:       newAuthManager(opts.Password, storageDir),
		projects:   openProjectRegistry(storageDir),
		storageDir: storageDir,
		mux:        http.NewServeMux(),

		compileSlots: make(chan struct{}, maxConcurrentCompiles),
		wasmFonts:    wasmassets.NewFontIndex(),
	}
	s.wasmPackages = s.newWasmPackages()
	s.collab = s.newCollabHub()
	go s.runCollabDiskWatch(context.Background())
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

	// MFA (TOTP) management for the logged-in account.
	s.handle("POST /api/auth/mfa/setup", s.auth.handleMFASetup)
	s.handle("POST /api/auth/mfa/enable", s.auth.handleMFAEnable)
	s.handle("POST /api/auth/mfa/disable", s.auth.handleMFADisable)

	// Project registry: ownership, members, share links (projects_api.go).
	s.handle("GET /api/projects", s.handleProjectsList)
	s.handle("POST /api/projects", s.handleProjectsCreate)
	s.handle("POST /api/projects/join", s.handleProjectJoin)
	s.handle("POST /api/projects/{id}/open", s.handleProjectOpen)
	s.handle("DELETE /api/projects/{id}", s.handleProjectDelete)
	s.handle("GET /api/projects/{id}/members", s.handleProjectMembers)
	s.handle("PUT /api/projects/{id}/members/{user}", s.handleProjectSetMember)
	s.handle("DELETE /api/projects/{id}/members/{user}", s.handleProjectRemoveMember)
	s.handle("GET /api/projects/{id}/shares", s.handleProjectShares)
	s.handle("POST /api/projects/{id}/shares", s.handleProjectCreateShare)
	s.handle("DELETE /api/projects/{id}/shares/{shareId}", s.handleProjectRevokeShare)

	// Workspace / project / file management. Reads need at least viewer and
	// writes at least editor on the open project, when it is registered.
	s.handle("GET /api/workspace/current", s.handleCurrentProject)
	s.handle("GET /api/workspace/role", s.handleWorkspaceRole)
	s.handle("GET /api/workspace/recent", s.handleRecentProjects)
	s.handle("POST /api/workspace/open", s.handleOpenProject)
	s.handle("POST /api/workspace/create", s.handleCreateProject)
	s.handleRole("GET /api/workspace/tree", projects.RoleViewer, s.handleTree)
	s.handleRole("GET /api/workspace/file", projects.RoleViewer, s.handleFileGet)
	s.handleRole("PUT /api/workspace/file", projects.RoleEditor, s.handleFilePut)
	s.handleRole("POST /api/workspace/file", projects.RoleEditor, s.handleFileCreate)
	s.handleRole("DELETE /api/workspace/file", projects.RoleEditor, s.handleFileDelete)
	s.handleRole("POST /api/workspace/rename", projects.RoleEditor, s.handleFileRename)

	// Export (PDF/PNG/SVG download).
	s.handleRole("GET /api/export", projects.RoleViewer, s.handleExport)
	// Static-PDF preview, served inline for an <iframe>/<embed> -- see
	// handlePreviewPdf's doc comment for why this exists alongside the
	// WebSocket-based /preview/ proxy below.
	s.handleRole("GET /api/preview/pdf", projects.RoleViewer, s.handlePreviewPdf)
	s.handleRole("POST /api/preview/render", projects.RoleViewer, s.handlePreviewRender)
	s.handleRole("POST /api/preview/anchors", projects.RoleViewer, s.handlePreviewAnchors)

	// VPS sync: desktop pushes project files here (see sync_handler.go).
	s.handleRole("GET /api/sync/manifest", projects.RoleViewer, s.handleSyncManifest)
	s.handleRole("GET /api/sync/pull", projects.RoleViewer, s.handleSyncPull)
	s.handleRole("POST /api/sync/push", projects.RoleEditor, s.handleSyncPush)
	s.handleRole("POST /api/sync/delete", projects.RoleEditor, s.handleSyncDelete)

	// Typst package manager (Tpix).
	s.handle("GET /api/packages/search", s.handlePkgSearch)
	s.handle("GET /api/packages/cached", s.handlePkgCached)
	s.handle("GET /api/packages/detail", s.handlePkgDetail)
	s.handle("POST /api/packages/download", s.handlePkgDownload)
	s.handle("POST /api/packages/pull-deps", s.handlePkgPullDeps)

	// Fonts and package archives for the in-browser compiler. Shared by
	// every project, so they only need a signed-in user (wasm_api.go).
	s.handle("GET /api/wasm/fonts", s.handleWasmFonts)
	s.handle("GET /api/wasm/fonts/{id}", s.handleWasmFont)
	s.handle("GET /api/wasm/packages/{namespace}/{name}/{version}", s.handleWasmPackage)

	// Settings.
	s.handle("GET /api/settings/general", settingsGetHandler(s.appSrv.Settings().General))
	// Settings are server-wide (and include executable paths), so changing
	// them is admin-only.
	s.handleAdmin("PUT /api/settings/general", settingsPutHandler(s.appSrv.Settings().General))
	s.handle("GET /api/settings/editor", settingsGetHandler(s.appSrv.Settings().Editor))
	s.handleAdmin("PUT /api/settings/editor", settingsPutHandler(s.appSrv.Settings().Editor))
	s.handle("GET /api/settings/typst", settingsGetHandler(s.appSrv.Settings().Typst))
	s.handleAdmin("PUT /api/settings/typst", settingsPutHandler(s.appSrv.Settings().Typst))
	s.handle("GET /api/settings/lsp", settingsGetHandler(s.appSrv.Settings().Lsp))
	s.handleAdmin("PUT /api/settings/lsp", settingsPutHandler(s.appSrv.Settings().Lsp))
	s.handle("GET /api/settings/agent", settingsGetHandler(s.appSrv.Settings().AcpAgent))
	s.handleAdmin("PUT /api/settings/agent", settingsPutHandler(s.appSrv.Settings().AcpAgent))
	s.handle("GET /api/settings/meta", s.handleSettingsMeta)
	s.handle("GET /api/settings/fonts", s.handleFontsList)
	s.handleAdmin("POST /api/settings/fonts", s.handleFontUpload)
	s.handleAdmin("DELETE /api/settings/fonts/{name}", s.handleFontDelete)
	// No /api/settings/tpix: it holds the package-registry API key, which
	// the web UI never uses, so it is not exposed to the browser at all.
	// No /api/settings/agent in settings-sync scope: AcpAgentSettings is
	// tied to the agent CLI installed on each machine (Cmd/Args/Env/AgentID
	// are machine-local), so it is excluded from the meta-based sync loop
	// even though its GET/PUT routes remain for the web UI.

	// Dropbox sync & integration.
	s.handle("GET /api/dropbox/status", s.handleDropboxStatus)
	s.handleAdmin("POST /api/dropbox/auth/url", s.handleDropboxAuthURL)
	s.handleAdmin("POST /api/dropbox/auth/callback", s.handleDropboxAuthCallback)
	s.handleAdmin("POST /api/dropbox/auth/token", s.handleDropboxSaveToken)
	s.handleAdmin("POST /api/dropbox/auth/disconnect", s.handleDropboxDisconnect)
	s.handleRole("POST /api/dropbox/sync", projects.RoleEditor, s.handleDropboxSync)
	s.handle("GET /api/dropbox/projects", s.handleDropboxListProjects)
	s.handle("POST /api/dropbox/import", s.handleDropboxImport)

	// AI agent registry / selection / auth. The agent can edit files and
	// run commands, so using it needs editor rights; configuring it is
	// admin-only.
	s.handle("GET /api/agent/registry", s.handleAgentRegistry)
	s.handleAdmin("POST /api/agent/select", s.handleAgentSelect)
	s.handleAdmin("POST /api/agent/auth/callback", s.handleAgentAuthCallback)
	s.handleAdmin("POST /api/agent/auth/{methodId}", s.handleAgentAuth)
	s.handleAdmin("POST /api/agent/preferred-config", s.handleSavePreferredConfig)
	s.handle("GET /api/agent/sessions", s.handleAgentSessions)
	s.handleAdmin("GET /api/console", s.handleConsole)

	// LSP + AI agent, over WebSocket.
	s.handleRole("GET /ws/lsp", projects.RoleViewer, s.handleLspWS)
	s.handleRole("GET /ws/agent", projects.RoleEditor, s.handleAgentWS)
	// Co-editing: viewers join to watch; each message is checked against
	// the sender's role (editor: text, commenter: comments).
	s.handleRole("GET /ws/collab", projects.RoleViewer, s.handleCollabWS)

	// Preview: reverse-proxied to tinymist's own preview server.
	s.handle("GET /api/preview/status", s.handlePreviewStatus)
	s.handleRole("POST /api/preview/restart", projects.RoleViewer, s.handlePreviewRestart)
	s.handleRole("POST /api/preview/cursor", projects.RoleViewer, s.handlePreviewCursor)
	s.mux.Handle(previewPathPrefix, s.auth.require(s.roleGate(projects.RoleViewer, s.handlePreviewProxy)))

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
