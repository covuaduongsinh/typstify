package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/coder/acp-go-sdk"
	"looz.ws/typstify/utils"
	"looz.ws/typstify/version"
)

var (
	AuthRequiredErr = errors.New("Authentication required")
	// ErrSessionManagerClosed is returned by NewSession/LoadSession/
	// ResumeSession/Authenticate when the manager's agent process has
	// already been shut down (e.g. the project changed underneath an
	// in-flight call) -- callers should treat it as transient and retry
	// against a freshly created manager, not as a hard failure.
	ErrSessionManagerClosed = errors.New("agent session manager was closed (project changed)")
)

type AgentConfig struct {
	Name string
	Cmd  string
	Args []string
	Env  []string // KEY=value pairs, appended to process env
}

type AgentConn struct {
	cmd               *exec.Cmd
	Conn              *acp.ClientSideConnection
	AgentInfo         acp.Implementation
	AgentCapabilities acp.AgentCapabilities
	AuthMethods       []acp.AuthMethod
	Authenticated     atomic.Bool
}

func (c *AgentConn) Close() error {
	return c.cmd.Process.Kill()
}

type SessionManager struct {
	// conn maintains a reference for connection between the currently running agent
	// and the ACP client.
	conn *AgentConn
	// The registered agent config.
	agentConfig AgentConfig
	// Active ACP sessions which are either newly created, loaded or resumed from Agents.
	activeSessions []*ACPSession
	mu             sync.Mutex
	// closed is set once Close has torn down the agent process. Guards
	// NewSession/LoadSession/ResumeSession/Authenticate against issuing an
	// RPC against a connection whose process was already killed.
	closed bool

	// Optional mcp servers to use when initializing sessions
	mcpServers []acp.McpServer
}

func NewSessionManager(mcpServers []acp.McpServer) *SessionManager {
	servers := make([]acp.McpServer, 0) // mcpServers must not be nil
	servers = append(servers, mcpServers...)
	return &SessionManager{
		mcpServers: servers,
	}
}

func (sm *SessionManager) Config() AgentConfig {
	return sm.agentConfig
}

// mcpServersForConn returns sm.mcpServers filtered to only the transports
// the connected agent actually declared support for in its Initialize
// response. Sending an agent an MCP server over a transport it just told us
// it doesn't support is worse than sending nothing: observed live
// 2026-09-18 with Google Antigravity (McpCapabilities.Http == false, logged
// as "Agent does not support MCP over HTTP, built-in tools will not be
// accessible") -- a simple no-tool prompt still worked, but a real
// file-editing prompt (which needs the agent to reach for a tool) hung
// indefinitely with zero ACP updates, consistent with the agent getting
// stuck trying to use a server entry it cannot actually connect to.
func (sm *SessionManager) mcpServersForConn(conn *AgentConn) []acp.McpServer {
	if conn == nil {
		return []acp.McpServer{}
	}
	filtered := make([]acp.McpServer, 0, len(sm.mcpServers))
	for _, s := range sm.mcpServers {
		if s.Http != nil && !conn.AgentCapabilities.McpCapabilities.Http {
			continue
		}
		if s.Sse != nil && !conn.AgentCapabilities.McpCapabilities.Sse {
			continue
		}
		filtered = append(filtered, s)
	}
	return filtered
}

func (sm *SessionManager) AgentConn() *AgentConn {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.conn
}

// Start runs a agent through ACP client. The config is used to specify
// which agent to be started.
func (sm *SessionManager) Start(ctx context.Context, agentConfig AgentConfig, enableDebug bool, agentLogStreamer io.Writer) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.conn != nil {
		return nil
	}

	sm.agentConfig = agentConfig
	// Use a clean context other than the incoming ctx, to prevent the command
	// from being canceled accidentally.
	cmd := utils.BuildCmd(context.Background(), agentConfig.Cmd, agentConfig.Args...)
	// Agents expect a full user environment (API tokens, locale, etc.), which
	// GUI launchers do not provide. Overlay the login-shell env captured at
	// startup, then the user-configured env, which takes precedence.
	cmd.Env = utils.MergeEnv(os.Environ(), utils.LoginShellEnv(), utils.ParseEnv(agentConfig.Env))

	if agentLogStreamer != nil {
		cmd.Stderr = agentLogStreamer
	} else {
		cmd.Stderr = os.Stderr
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe error: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe error: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	in, out := stdin, stdout
	if enableDebug {
		in, out = duplicatedIO(stdin, stdout)
	}

	client := NewACPClient(sm)
	conn := acp.NewClientSideConnection(client, in, out)
	conn.SetLogger(slog.Default()) // TODO: redirect to app console.

	// Initialize
	initResp, err := conn.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientInfo: &acp.Implementation{
			Name:    "Typstify",
			Version: version.BinVersion,
		},
		ClientCapabilities: acp.ClientCapabilities{
			Meta: client.ExtensionCapabilities(),
			Fs: acp.FileSystemCapabilities{
				ReadTextFile:  true,
				WriteTextFile: true,
			},
			Terminal: true,
		},
	})

	err = checkACPErr(err)
	if err != nil {
		_ = cmd.Process.Kill()

		return err
	}

	// A non-compliant (or crashed-before-replying) agent can return a
	// nil-but-no-error Initialize response -- observed live 2026-09-18 with
	// the community "grok-build" ACP registry entry, which panicked the
	// whole app here. Fail the connection instead of dereferencing nil.
	if initResp.AgentInfo == nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("agent %q returned an invalid initialize response (missing agentInfo)", agentConfig.Name)
	}

	go func() {
		<-conn.Done()
		log.Println("Peer closed connections")
	}()

	sm.conn = &AgentConn{
		cmd:               cmd,
		Conn:              conn,
		AgentInfo:         *initResp.AgentInfo,
		AgentCapabilities: initResp.AgentCapabilities,
		AuthMethods:       initResp.AuthMethods,
	}

	log.Printf("Connected to %s (ACP version %v)", initResp.AgentInfo.Name, initResp.ProtocolVersion)
	if !initResp.AgentCapabilities.McpCapabilities.Http {
		log.Printf("Warning: Agent does not support MCP over HTTP, built-in tools will not be accessible")
	}

	return nil
}

func (sm *SessionManager) Authenticate(ctx context.Context, methodID string) error {
	sm.mu.Lock()
	conn := sm.conn
	closed := sm.closed
	sm.mu.Unlock()

	if closed {
		return ErrSessionManagerClosed
	}
	if conn == nil {
		return fmt.Errorf("Agent not initialized")
	}

	_, err := conn.Conn.Authenticate(ctx, acp.AuthenticateRequest{MethodId: methodID})
	err = checkACPErr(err)
	if err != nil {
		return err
	}

	return nil
}

func (sm *SessionManager) NewSession(ctx context.Context, cwd string) (*ACPSession, error) {
	sm.mu.Lock()
	conn := sm.conn
	closed := sm.closed
	sm.mu.Unlock()

	if closed {
		return nil, ErrSessionManagerClosed
	}
	if conn == nil {
		return nil, fmt.Errorf("not connected to agent")
	}

	cwd, err := filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}

	resp, err := conn.Conn.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        cwd,
		McpServers: sm.mcpServersForConn(conn),
	})

	err = checkACPErr(err)
	if err != nil {
		return nil, err
	}
	session := NewACPSession(string(resp.SessionId), cwd)
	if resp.Modes != nil {
		session.SetMode(*resp.Modes)
	}
	session.SetConn(conn)
	session.SetConfigOptions(resp.ConfigOptions)

	sm.mu.Lock()
	sm.activeSessions = append(sm.activeSessions, session)
	sm.mu.Unlock()
	log.Println("created new session: ", session.SessionID)
	return session, nil
}

// ListSessions from Agent. Sessions returned are not *active*, callers need to call LoadSession to get an active one.
func (sm *SessionManager) ListSessions(ctx context.Context, filterCwd string) ([]*ACPSession, error) {
	if sm.conn == nil {
		return nil, fmt.Errorf("not connected to agent")
	}

	// If the agent does not support loading sessions, return without error.
	listCap := sm.conn.AgentCapabilities.SessionCapabilities.List
	if listCap == nil {
		return nil, fmt.Errorf("Agent does not support listing sessions")
	}

	cwd, err := filepath.Abs(filterCwd)
	if err != nil {
		return nil, err
	}

	allAgentSessions := make([]*ACPSession, 0)

	cursor := ""
	first := true
	for cursor != "" || first {
		first = false
		resp, err := sm.conn.Conn.ListSessions(ctx, acp.ListSessionsRequest{
			Cwd:    &cwd,
			Cursor: &cursor,
		})

		err = checkACPErr(err)
		if err != nil {
			return nil, err
		}

		for _, sn := range resp.Sessions {
			// some fields is not populated, needs to call LoadSession to fill them.
			session := NewACPSession(string(sn.SessionId), cwd)

			var title, updatedAt string
			if sn.Title != nil {
				title = *sn.Title
			}
			if sn.UpdatedAt != nil {
				updatedAt = *sn.UpdatedAt
			}
			session.UpdateInfo(title, updatedAt)

			allAgentSessions = append(allAgentSessions, session)

		}

		if resp.NextCursor == nil {
			cursor = ""
		} else {
			cursor = *resp.NextCursor
		}
	}

	return allAgentSessions, nil
}

// LoadSession loads a session from the Agent. The Agent will replay the entire
// conversation to the Client in the form of session/update notifications.
func (sm *SessionManager) LoadSession(ctx context.Context, session *ACPSession) (*ACPSession, error) {
	if session.Active() {
		return session, nil
	}

	sm.mu.Lock()
	conn := sm.conn
	closed := sm.closed
	sm.mu.Unlock()

	if closed {
		return nil, ErrSessionManagerClosed
	}
	if conn == nil {
		return nil, fmt.Errorf("not connected to agent")
	}

	// If the agent does not support loading session, return without error.
	if !conn.AgentCapabilities.LoadSession {
		return nil, nil
	}

	// Agents will call 'session/update' before returing from LoadSession, so we have to make
	// it an active session before the rpc return.
	sm.mu.Lock()
	session.SetConn(conn)
	sm.activeSessions = append(sm.activeSessions, session)
	sm.mu.Unlock()

	resp, err := conn.Conn.LoadSession(ctx, acp.LoadSessionRequest{
		Cwd:        session.Cwd,
		McpServers: sm.mcpServersForConn(conn),
		SessionId:  acp.SessionId(session.SessionID),
	})

	err = checkACPErr(err)
	if err != nil {
		return nil, err
	}

	if resp.Modes != nil {
		session.SetMode(*resp.Modes)
	}
	session.SetConfigOptions(resp.ConfigOptions)

	return session, nil
}

// ResumeSession loads a session from the Agent. Unlike LoadSession, the Agent will NOT replay prior
// conversation to the client.
func (sm *SessionManager) ResumeSession(ctx context.Context, session *ACPSession) (*ACPSession, error) {
	if session.Active() {
		return session, nil
	}

	sm.mu.Lock()
	conn := sm.conn
	closed := sm.closed
	sm.mu.Unlock()

	if closed {
		return nil, ErrSessionManagerClosed
	}
	if conn == nil {
		return nil, fmt.Errorf("not connected to agent")
	}

	// If the agent does not support resume sessions, return without error.
	resumeCap := conn.AgentCapabilities.SessionCapabilities.Resume
	if resumeCap == nil {
		return nil, fmt.Errorf("Agent does not support resuming session")
	}

	// Agents will call 'session/update' before returing from ResumeSession, so we have to make
	// it an active session before the rpc return.
	sm.mu.Lock()
	session.SetConn(conn)
	sm.activeSessions = append(sm.activeSessions, session)
	sm.mu.Unlock()

	resp, err := conn.Conn.ResumeSession(ctx, acp.ResumeSessionRequest{
		Cwd:        session.Cwd,
		McpServers: sm.mcpServersForConn(conn),
		SessionId:  acp.SessionId(session.SessionID),
	})

	err = checkACPErr(err)
	if err != nil {
		return nil, err
	}

	if resp.Modes != nil {
		session.SetMode(*resp.Modes)
	}
	session.SetConfigOptions(resp.ConfigOptions)

	return session, nil
}

// CloseSession allow Clients to tell the Agent to cancel any ongoing work
// for a session and free any resources associated with that active session.
func (sm *SessionManager) CloseSession(ctx context.Context, sessionID string) error {
	sm.mu.Lock()
	session := sm.getActiveSession(sessionID)
	if session == nil {
		sm.mu.Unlock()
		return fmt.Errorf("no active session found: %s", sessionID)
	}
	sm.mu.Unlock()

	// Always release the session locally (subscriber goroutine, blocked
	// senders, terminals): previously an agent without session/close
	// support leaked all of that on every browser disconnect.
	defer func() {
		session.Close()
		sm.mu.Lock()
		sm.activeSessions = slices.DeleteFunc(sm.activeSessions, func(sn *ACPSession) bool {
			return sn.SessionID == sessionID
		})
		sm.mu.Unlock()
	}()

	// Tell the agent too, if it supports closing sessions.
	if conn := session.Conn(); conn != nil && conn.AgentCapabilities.SessionCapabilities.Close != nil {
		_, err := conn.Conn.CloseSession(ctx, acp.CloseSessionRequest{
			SessionId: acp.SessionId(sessionID),
		})
		if err := checkACPErr(err); err != nil {
			return err
		}
	}

	return nil
}

func (sm *SessionManager) getActiveSession(sessionID string) *ACPSession {
	idx := slices.IndexFunc(sm.activeSessions, func(sn *ACPSession) bool {
		return sn.SessionID == sessionID
	})

	if idx < 0 {
		return nil
	}

	return sm.activeSessions[idx]
}

func (sm *SessionManager) GetActiveSession(sessionID string) *ACPSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	return sm.getActiveSession(sessionID)
}

func (sm *SessionManager) Close(ctx context.Context) error {
	sm.mu.Lock()
	if sm.conn == nil {
		sm.mu.Unlock()
		return nil
	}

	activeSessions := sm.activeSessions
	sm.mu.Unlock()

	var err error
	for _, sn := range activeSessions {
		closeErr := sm.CloseSession(ctx, sn.SessionID)
		if closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()
	closeErr := sm.conn.Close()
	if closeErr != nil {
		err = errors.Join(err, closeErr)
	}
	sm.closed = true

	return err
}

func checkACPErr(err error) error {
	if err == nil {
		return nil
	}

	if re, ok := err.(*acp.RequestError); ok {
		if re.Code == -32000 {
			return AuthRequiredErr
		}
		return fmt.Errorf("ACP request error (%d): %s", re.Code, re.Message)
	} else {
		return fmt.Errorf("ACP request error: %w", err)
	}
}

func duplicatedIO(stdin io.WriteCloser, stdout io.ReadCloser) (io.WriteCloser, io.ReadCloser) {
	// Duplicate what the Client sends TO the Agent (Client -> Agent)
	// Anything written to loggedStdin goes to both the agent's stdin AND the console
	loggedStdin := struct {
		io.Writer
		io.Closer
	}{
		Writer: io.MultiWriter(stdin, os.Stderr),
		Closer: stdin,
	}

	// Duplicate what the Agent sends BACK to the Client (Agent -> Client)
	// As the ACP client reads from loggedStdout, a copy is automatically piped to the console
	loggedStdout := struct {
		io.Reader
		io.Closer
	}{
		Reader: io.TeeReader(stdout, os.Stderr),
		Closer: stdout,
	}

	return loggedStdin, loggedStdout

}
