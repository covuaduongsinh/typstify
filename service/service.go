package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/typstify/tpix-cli"
	"github.com/typstify/tpix-cli/api"
	"looz.ws/typstify/agent"
	"looz.ws/typstify/lsp"
	"looz.ws/typstify/service/bus"
	"looz.ws/typstify/service/mcp"
	"looz.ws/typstify/service/net"
	"looz.ws/typstify/service/settings"
	"looz.ws/typstify/typst"
	"looz.ws/typstify/typst/pkg"
	"looz.ws/typstify/utils"
	"looz.ws/typstify/version"
	"looz.ws/typstify/widgets/console"
)

const (
	staticMcpServerPort = 5322
)

type ServiceFacade struct {
	// requestSwitch, invalidateWindow and onUIClose are wired by the Gio
	// desktop UI layer (see ui.NewUI/ui.registerViews) through
	// SetViewManager. They are opaque func hooks -- rather than the
	// concrete github.com/oligo/gioview/view types -- specifically so that
	// package service has no Gio dependency and cmd/typstify-server (the
	// headless web-mode entrypoint, docs/plans/plan_web_version.md) can be
	// built without pulling in gioui.org. fileChooserBuilder is opaque for
	// the same reason (it normally returns *gioview/explorer.FileChooser).
	requestSwitch      func(intent any)
	invalidateWindow   func()
	onUIClose          func()
	currentView        func() any
	settings           *settings.Settings
	eventbus           *bus.EventBus
	workspaceSrv       *WorkspaceService
	pkgService         *pkg.TypstPkgService
	windowSrv          *WindowService
	previewSrv         *lsp.PreviewService
	fileChooserBuilder func() any
	consoleState       *console.ConsoleState
	acpSessionManager  *agent.SessionManager
	acpMu              sync.Mutex
	acpCond            *sync.Cond
	acpStarting        bool
	mcpServer          *agent.McpServer // the built-in mcp server

	// projMu guards currentProjectDir and previewSrv: the web server
	// switches projects from one HTTP handler while others read them.
	projMu            sync.RWMutex
	currentProjectDir string

	// Window layout metrics for native webview positioning.
	// Set by the home view each frame.
	WindowContentWidth int
	ViewAreaTopOffset  int
}

func NewService(ctx context.Context) *ServiceFacade {
	eventbus := bus.NewEventBus(ctx, false)
	st := settings.NewSettings(eventbus)

	s := &ServiceFacade{
		eventbus:     eventbus,
		settings:     st,
		windowSrv:    NewWindowService(ctx, st),
		consoleState: console.NewConsoleState(1000),
	}
	s.acpCond = sync.NewCond(&s.acpMu)

	pkgSrv := pkg.NewTypstPkgService(st.Typst(), st.Tpix(), s.TpixClient())
	s.pkgService = pkgSrv

	eventbus.Subscribe(s, "service.onSettingUpdate", bus.TopicSettingsUpdated, func(topic string, data interface{}) {
		s.pkgService = pkg.NewTypstPkgService(st.Typst(), st.Tpix(), s.TpixClient())
	})

	s.workspaceSrv = NewWorkspaceService(st.General().RootDir, eventbus, s.TpixClient())

	// init executable lookup path.
	lsp.SetupCmdBuilder(s.settings.General().ExternalTinymist)
	typst.SetupCmdBuilder(s.settings.General().ExternalTypst)

	s.RegisterDevice()

	return s
}

func (s *ServiceFacade) EventBus() *bus.EventBus {
	return s.eventbus
}

func (s *ServiceFacade) Settings() *settings.Settings {
	return s.settings
}

func (s *ServiceFacade) PkgService() *pkg.TypstPkgService {
	return s.pkgService
}

func (s *ServiceFacade) Workspace() *WorkspaceService {
	return s.workspaceSrv
}

func (s *ServiceFacade) WindowService() *WindowService {
	return s.windowSrv
}

func (s *ServiceFacade) InitFileChooser(builder func() any) {
	s.fileChooserBuilder = builder
}

// FileChooser returns the desktop UI's *gioview/explorer.FileChooser as an
// opaque any; callers in package ui type-assert it back. nil (headless web
// mode, or before InitFileChooser was called) if unset.
func (s *ServiceFacade) FileChooser() any {
	if s.fileChooserBuilder == nil {
		return nil
	}
	return s.fileChooserBuilder()
}

// SetViewManager wires the desktop UI's view-switching/window-refresh/
// shutdown/current-view hooks into the service layer.
func (s *ServiceFacade) SetViewManager(requestSwitch func(intent any), invalidateWindow func(), onUIClose func(), currentView func() any) {
	s.requestSwitch = requestSwitch
	s.invalidateWindow = invalidateWindow
	s.onUIClose = onUIClose
	s.currentView = currentView
}

// RequestSwitch asks the desktop UI to switch views. intent is normally a
// gioview/view.Intent; it is a no-op (e.g. headless web mode, where there is
// no view manager) if SetViewManager was never called.
func (s *ServiceFacade) RequestSwitch(intent any) {
	if s.requestSwitch != nil {
		s.requestSwitch(intent)
	}
}

func (s *ServiceFacade) RefreshWindow() {
	if s.invalidateWindow != nil {
		s.invalidateWindow()
	}
}

func (s *ServiceFacade) Close(ctx context.Context) {
	if s.onUIClose != nil {
		s.onUIClose()
	}
	s.workspaceSrv.Close()
	s.windowSrv.Shutdown()
	s.windowSrv.Wait()
	lsp.StopLsp()
	if previewSrv := s.PreviewService(); previewSrv != nil {
		previewSrv.Destroy(ctx)
	}

	s.stopAcpSessionManager(ctx)
	if s.mcpServer != nil {
		s.mcpServer.Shutdown(ctx)
	}
	log.Println("service down")
}

func (s *ServiceFacade) RegisterDevice() {
	api := net.NewRemote()
	go func(dev settings.Device) {
		req := &net.DeviceInfo{
			DeviceID:   dev.ID,
			Hostname:   dev.Hostname,
			OS:         dev.OS,
			Platform:   dev.Platform,
			AppVersion: version.BinVersion,
		}

		tz, _ := time.Now().Zone()
		req.Timezone = tz

		err := api.RegisterDevice(req)
		if err != nil {
			log.Println("Register device failed: ", err)
		}
	}(s.settings.General().GetDeviceInfo())

}

func (s *ServiceFacade) CheckUpdate() *net.ReleaseInfo {
	api := net.NewRemote()
	device := s.settings.General().GetDeviceInfo()
	release, err := api.CheckUpdate(&net.UpdateCheckReq{
		DeviceID:       device.ID,
		CurrentVersion: version.BinVersion,
		UseBeta:        false,
	})

	if err != nil {
		log.Println("check update failed: ", err)
		return nil
	}

	latestVer := utils.ParseVersion(release.AppVersion)
	currentVer := utils.ParseVersion(version.BinVersion)

	if latestVer == nil {
		return nil
	}
	if currentVer == nil {
		return release
	}

	if currentVer.Compare(latestVer) < 0 {
		return release
	}

	return nil
}

func (s *ServiceFacade) SetProjectDir(dir string) {
	if dir == "" {
		return
	}

	s.projMu.Lock()
	s.currentProjectDir = dir
	s.projMu.Unlock()

	s.Workspace().SwitchWorkspace(dir)

	// init executable lookup path.
	lsp.SetupCmdBuilder(s.settings.General().ExternalTinymist)
	typst.SetupCmdBuilder(s.settings.General().ExternalTypst)

	// connect to LSP server in an eager way.
	client := lsp.GetLspClient(dir, s.Settings())
	if s.settings.Lsp().EnableLSPLogs != 0 {
		client.SetServreLogStreamer(s.consoleState)
	} else {
		client.SetServreLogStreamer(io.Discard)
	}

	previewMode := lsp.PreviewMode(s.Workspace().LoadWorkspaceSettings().PreviewMode)
	if previewMode == "" {
		previewMode = lsp.DocumentPreviewMode
	}

	previewSrv := lsp.NewPreviwService(client)
	// No explicit Destroy of the previous service: its preview lived on the
	// previous LSP client (stopped by GetLspClient when the workspace
	// changes), and Start kills tinymist's shared "default_preview" task
	// before starting a new one on the same client.
	s.projMu.Lock()
	s.previewSrv = previewSrv
	s.projMu.Unlock()
	go func() {
		previewSrv.Start(context.Background(),
			lsp.PreviewOptions{
				Mode:          previewMode,
				InvertColor:   "never",
				PartialRender: s.settings.Lsp().EnablePartialRenderPreview,
			}, nil)
	}()

	// stop the last acpSessionManager
	s.stopAcpSessionManager(context.Background())
}

func (s *ServiceFacade) RestartPreview(ctx context.Context, onFinish func()) {
	s.RestartPreviewWithEntry(ctx, "", onFinish)
}

func (s *ServiceFacade) RestartPreviewWithEntry(ctx context.Context, entryFile string, onFinish func()) {
	s.projMu.RLock()
	previewSrv, projectDir := s.previewSrv, s.currentProjectDir
	s.projMu.RUnlock()
	if previewSrv == nil {
		return
	}

	previewMode := lsp.PreviewMode(s.Workspace().LoadWorkspaceSettings().PreviewMode)
	if previewMode == "" {
		previewMode = lsp.DocumentPreviewMode
	}

	go func() {
		previewSrv.Start(context.Background(),
			lsp.PreviewOptions{
				Mode:          previewMode,
				ProjectRoot:   projectDir,
				EntryFile:     entryFile,
				InvertColor:   "never",
				PartialRender: s.settings.Lsp().EnablePartialRenderPreview,
			}, onFinish)
	}()
}

func (s *ServiceFacade) PreviewService() *lsp.PreviewService {
	s.projMu.RLock()
	defer s.projMu.RUnlock()
	return s.previewSrv
}

func (s *ServiceFacade) CurrentProjectDir() string {
	s.projMu.RLock()
	defer s.projMu.RUnlock()
	return s.currentProjectDir
}

func (s *ServiceFacade) Console() *console.ConsoleState {
	return s.consoleState
}

func (s *ServiceFacade) initMcpServer(ctx context.Context) {
	if s.mcpServer != nil {
		s.mcpServer.Shutdown(ctx)
	}

	serverPort := 0
	useStaticPort := s.settings.AcpAgent().UseStaticMcpPort == 1
	if useStaticPort {
		serverPort = staticMcpServerPort
	}
	s.mcpServer = agent.NewMcpServer(serverPort)

	projectDir, previewSrv := s.CurrentProjectDir(), s.PreviewService()
	client := lsp.GetLspClient(projectDir, s.Settings())

	// compilerTool := mcp.TypstCompilerHandler(s.CurrentProjectDir(), s.Settings().Typst())
	// agent.AddMcpTool(s.mcpServer, mcp.TypstCompilerTool, compilerTool)
	activeDocQuerier := func() *mcp.ActiveDocument {
		if s.currentView == nil {
			return nil
		}
		cv := s.currentView()
		if cv == nil {
			return nil
		}
		if provider, ok := cv.(mcp.ActiveDocProvider); ok {
			doc := provider.GetActiveDocument()
			return &doc
		}
		return nil
	}
	editorToolSrv := mcp.NewEditorMcpService(projectDir, s.settings, client, previewSrv, s.eventbus, activeDocQuerier)
	s.mcpServer.RegisterToolProvider(editorToolSrv)

	pkgToolSrv := mcp.NewPackageMcpService(projectDir, s.TpixClient(), s.PkgService())
	s.mcpServer.RegisterToolProvider(pkgToolSrv)
	s.mcpServer.RegisterResourceProvider(pkgToolSrv)

	if err := s.mcpServer.Run(); err != nil {
		// Run without the built-in tools rather than crashing (the static
		// MCP port may be taken, e.g. by a second instance). Drop the
		// server so it is neither advertised to agents nor shut down.
		log.Printf("built-in MCP tools disabled: %v", err)
		s.mcpServer = nil
	}
}

func (s *ServiceFacade) listMcpServer() []acp.McpServer {
	mcpServers := make([]acp.McpServer, 0)
	if s.mcpServer == nil { // failed to start: don't advertise it
		return mcpServers
	}

	// built-in mcp server
	ip, port := s.mcpServer.Addr()
	mcpServers = append(mcpServers, acp.McpServer{
		Http: &acp.McpServerHttpInline{
			Name:    agent.ServerName,
			Type:    "http",
			Url:     fmt.Sprintf("http://%s:%d", ip, port),
			Headers: []acp.HttpHeader{},
		},
	})

	return mcpServers
}

func (s *ServiceFacade) StartACPSession(ctx context.Context, projectDir string) (*agent.ACPSession, error) {
	as := s.settings.AcpAgent()

	s.acpMu.Lock()
	// If the configured agent changed, stop the running one so the next
	// getOrCreateAcpSessionManager call spawns a fresh one with the new config.
	if mgr := s.acpSessionManager; mgr != nil && !configEqual(mgr.Config(), as) {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := mgr.Close(closeCtx); err != nil {
			log.Printf("close old ACP session manager: %v", err)
		}
		s.acpSessionManager = nil
		log.Println("agent config changed, restarting session manager...")
	}
	s.acpMu.Unlock()

	mgr, err := s.getOrCreateAcpSessionManager(ctx)
	if err != nil {
		return nil, err
	}

	return mgr.NewSession(ctx, projectDir)
}

// getOrCreateAcpSessionManager returns the current ACP session manager,
// starting one if none exists. Concurrent callers that observe no manager
// coordinate through acpCond so only one of them actually spawns the agent
// process -- the rest wait for that spawn to finish and reuse its result.
// Without this, two /ws/agent connections opening close together (e.g. one
// racing a project switch) could each spawn their own `npx` agent process
// and independently publish s.acpSessionManager with no ordering guarantee,
// orphaning whichever process lost the race.
func (s *ServiceFacade) getOrCreateAcpSessionManager(ctx context.Context) (*agent.SessionManager, error) {
	s.acpMu.Lock()
	for s.acpSessionManager == nil && s.acpStarting {
		s.acpCond.Wait()
	}
	if mgr := s.acpSessionManager; mgr != nil {
		s.acpMu.Unlock()
		return mgr, nil
	}
	s.acpStarting = true
	s.acpMu.Unlock()

	mgr, err := s.startAcpSessionManager(ctx)

	s.acpMu.Lock()
	s.acpStarting = false
	if err == nil {
		s.acpSessionManager = mgr
	}
	s.acpCond.Broadcast()
	s.acpMu.Unlock()

	return mgr, err
}

func configEqual(a agent.AgentConfig, as *settings.AcpAgentSettings) bool {
	return a.Name == as.AgentName && a.Cmd == as.Cmd && strings.Join(a.Args, " ") == as.Args && strings.Join(a.Env, " ") == as.Env
}

func (s *ServiceFacade) CloseACPSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	if s.acpSessionManager == nil {
		return nil
	}

	return s.acpSessionManager.CloseSession(ctx, sessionID)
}

// defaultAgentConfig is the fallback when no agent is configured.
var defaultAgentConfig = agent.AgentConfig{
	Name: "Claude Code",
	Cmd:  "npx",
	Args: []string{"-y", "@agentclientprotocol/claude-agent-acp@0.50.0"},
}

func (s *ServiceFacade) buildAgentConfig() agent.AgentConfig {
	as := s.settings.AcpAgent()
	if as.Cmd == "" {
		return defaultAgentConfig
	}
	return agent.AgentConfig{
		Name: as.AgentName,
		Cmd:  as.Cmd,
		Args: strings.Fields(as.Args),
		Env:  strings.Fields(as.Env),
	}
}

// startAcpSessionManager spawns a new agent process for the currently
// configured AI agent and completes the ACP Initialize handshake. It does
// not touch s.acpSessionManager -- callers publish the result themselves
// under s.acpMu (see getOrCreateAcpSessionManager) so concurrent callers
// never race on that field.
func (s *ServiceFacade) startAcpSessionManager(ctx context.Context) (*agent.SessionManager, error) {
	cwd := s.CurrentProjectDir()
	if cwd == "" {
		return nil, errors.New("No project dir is open")
	}

	childCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	// start the mcp server
	s.initMcpServer(ctx)

	mgr := agent.NewSessionManager(s.listMcpServer())

	// Setting env ACP_DEBUG=1 will turn Typstify into ACP debug mode.
	acpDebug := os.Getenv("ACP_DEBUG") == "1"
	agentConfig := s.buildAgentConfig()

	// stream agent logs(usually streamed via stderr) to console. Some agents like Cline
	// write Device-auth flow guide to the console log stream, so user can complete the authentication flow.
	if err := mgr.Start(childCtx, agentConfig, acpDebug, s.consoleState); err != nil {
		return nil, err
	}

	return mgr, nil
}

func (s *ServiceFacade) stopAcpSessionManager(ctx context.Context) {
	s.acpMu.Lock()
	defer s.acpMu.Unlock()

	if s.acpSessionManager != nil {
		childCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		s.acpSessionManager.Close(childCtx)
		s.acpSessionManager = nil
	}
}

func (s *ServiceFacade) AcpSessionManager() *agent.SessionManager {
	mgr, err := s.getOrCreateAcpSessionManager(context.Background())
	if err != nil {
		log.Println("start ACP session manager failed: ", err)
		return nil
	}
	return mgr
}

func (s *ServiceFacade) TpixClient() *tpix.TpixSdk {
	var provider api.ApiKeyProvider
	if s.settings.Tpix().ApiKey != "" {
		provider = &tpixApiKeyProvider{setting: s.settings.Tpix()}
	}
	httpClient := api.NewHttpClient(provider)
	client := tpix.NewTpixSdk(httpClient)
	client.WithReporter(tpixCliReporter{w: s.consoleState}.Report)

	return client
}

func (s *ServiceFacade) Authenticated() bool {
	return s.settings.Tpix().ApiKey != ""
}
