package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	acp "github.com/coder/acp-go-sdk"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Wire message shapes for /ws/agent -- mirrors server/agent_ws.go's
// unexported agentClientMessage/agentServerMessage exactly. Duplicated
// here since those aren't exported (this IS the wire contract the browser
// frontend's separate TS client also implements independently).
type agentWireClientMsg struct {
	Type     string           `json:"type"`
	Text     string           `json:"text,omitempty"`
	OptionID string           `json:"optionId,omitempty"`
	Images   []agentWireImage `json:"images,omitempty"`
	ConfigID string           `json:"configId,omitempty"`
	Value    string           `json:"value,omitempty"`
}

type agentWireImage struct {
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

type agentWireServerMsg struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionId,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Message   string          `json:"message,omitempty"`
}

var _ ChatSession = (*RemoteChatSession)(nil)

// ErrRemotePromptBuffered mirrors ErrPromptBuffered for a remote session --
// see RemoteChatSession.Prompt.
var ErrRemotePromptBuffered = errors.New("remote agent: a prompt turn is ongoing, prompt was buffered for later")

// RemoteChatSession is a ChatSession backed by /ws/agent on a self-hosted
// Typstify server instead of a locally-spawned agent process (Giai đoạn
// B, "Remote Agent mode") -- so a chat started on the web (or another
// desktop) can be continued here against the exact same running agent
// session/process, instead of this client spawning its own independent
// one.
//
// Several pieces of ACPSession's state never cross this wire at all: the
// SERVER's own local ACPSession.SubscribeUpdates dispatch (session.go)
// swallows SessionInfoUpdate (title), UsageUpdate and
// AvailableCommandsUpdate internally -- there is no OnSessionInfoUpdate/
// OnUsageUpdate/OnAvailableCommandsUpdate on SessionUpdateSubsciber to
// forward them through -- so the existing web frontend has no visibility
// into them either. This client matches that: Title is whatever the
// caller already knew (e.g. from GET /api/agent/sessions) and never
// updates live; AvailableCommands/Usage stay at their zero value.
type RemoteChatSession struct {
	id         string
	workingDir string
	title      string

	conn *websocket.Conn
	done chan struct{}

	active atomic.Bool

	turnMu      sync.Mutex
	turnPending bool
	turnResult  chan agentWireServerMsg

	configMu      sync.Mutex
	configOptions []acp.SessionConfigOption

	subMu sync.Mutex
	sub   SessionUpdateSubsciber

	closeOnce sync.Once
}

// DialRemoteChatSession connects to serverURL's /ws/agent (authenticating
// with a Bearer token) and either starts a new session for workingDir
// (sessionID == "") or re-attaches to an existing one -- resume selects
// Load (replays prior messages) vs Resume (doesn't), matching
// server/agent_ws.go. title is used as this session's display title (see
// the type doc for why it can never update live); pass "" for a brand new
// session.
func DialRemoteChatSession(ctx context.Context, serverURL, token, workingDir, sessionID, title string, resume bool) (*RemoteChatSession, error) {
	wsURL, err := remoteAgentWSURL(serverURL, sessionID, resume)
	if err != nil {
		return nil, err
	}

	header := http.Header{}
	if token != "" {
		header.Set("Authorization", "Bearer "+token)
	}

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		return nil, fmt.Errorf("remote agent: connect failed: %w", err)
	}
	// Match the server's own bump (server/agent_ws.go) -- large pasted
	// images/screenshots exceed the 32KiB default read limit otherwise.
	conn.SetReadLimit(20 << 20)

	rs := &RemoteChatSession{
		workingDir: workingDir,
		title:      title,
		conn:       conn,
		done:       make(chan struct{}),
	}

	readyErr := make(chan error, 1)
	go rs.readLoop(readyErr)

	select {
	case err := <-readyErr:
		if err != nil {
			conn.CloseNow()
			return nil, err
		}
	case <-ctx.Done():
		conn.CloseNow()
		return nil, ctx.Err()
	}

	return rs, nil
}

func remoteAgentWSURL(serverURL, sessionID string, resume bool) (string, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return "", fmt.Errorf("invalid server URL: %w", err)
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	case "ws", "wss":
		// already fine
	default:
		return "", fmt.Errorf("unsupported server URL scheme %q", u.Scheme)
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/ws/agent"

	if sessionID != "" {
		q := u.Query()
		q.Set("sessionId", sessionID)
		if resume {
			q.Set("resume", "1")
		}
		u.RawQuery = q.Encode()
	}

	return u.String(), nil
}

// readLoop is the single goroutine reading every message for this
// connection's lifetime, dispatching each to whichever of (a) the pending
// Prompt() call, (b) the subscribed UI, or (c) internal state
// (configOptions/active/id) it belongs to. The first "ready" (or an early
// "error"/close) is reported via readyErr; DialRemoteChatSession blocks on
// that before handing the session to its caller.
func (rs *RemoteChatSession) readLoop(readyErr chan<- error) {
	first := true
	reportReady := func(err error) {
		if first {
			first = false
			readyErr <- err
		}
	}

	defer func() {
		rs.active.Store(false)
		rs.closeOnce.Do(func() { close(rs.done) })
	}()

	for {
		var msg agentWireServerMsg
		if err := wsjson.Read(context.Background(), rs.conn, &msg); err != nil {
			reportReady(fmt.Errorf("remote agent: connection closed: %w", err))
			rs.failPendingTurn(fmt.Errorf("remote agent: connection lost: %w", err))
			return
		}

		switch msg.Type {
		case "ready":
			rs.id = msg.SessionID
			rs.active.Store(true)
			reportReady(nil)

		case "configOptions":
			var opts []acp.SessionConfigOption
			_ = json.Unmarshal(msg.Data, &opts)
			rs.configMu.Lock()
			rs.configOptions = opts
			rs.configMu.Unlock()
			rs.publish(ConfigOptionUpdate{ConfigOptions: opts})

		case "userMessage":
			var chunk UserMessageChunk
			_ = json.Unmarshal(msg.Data, &chunk)
			rs.publish(chunk)

		case "agentMessage":
			var chunk AgentMessageChunk
			_ = json.Unmarshal(msg.Data, &chunk)
			rs.publish(chunk)

		case "agentThought":
			var chunk AgentThoughtChunk
			_ = json.Unmarshal(msg.Data, &chunk)
			rs.publish(chunk)

		case "toolCall":
			var tc ToolCall
			_ = json.Unmarshal(msg.Data, &tc)
			rs.publish(tc)

		case "toolCallUpdate":
			var tc ToolCallUpdate
			_ = json.Unmarshal(msg.Data, &tc)
			rs.publish(tc)

		case "plan":
			var plan Plan
			_ = json.Unmarshal(msg.Data, &plan)
			rs.publish(plan)

		case "permissionRequest":
			var req acp.RequestPermissionRequest
			_ = json.Unmarshal(msg.Data, &req)
			rs.handlePermissionRequest(req)

		case "turnEnd", "error", "authRequired":
			rs.deliverTurnResult(msg)

		default:
			// Unknown/forward-compatible message type: ignore.
		}
	}
}

func (rs *RemoteChatSession) publish(update any) {
	rs.subMu.Lock()
	sub := rs.sub
	rs.subMu.Unlock()
	if sub == nil {
		return
	}

	switch u := update.(type) {
	case UserMessageChunk:
		sub.OnUserMessage(u)
	case AgentMessageChunk:
		sub.OnAgentMessage(u)
	case AgentThoughtChunk:
		sub.OnAgentThought(u)
	case ToolCall:
		sub.OnToolCallInit(u)
	case ToolCallUpdate:
		sub.OnToolCallUpdate(u)
	case Plan:
		sub.OnPlan(u)
	case ConfigOptionUpdate:
		sub.OnConfigOptionUpdate(u)
	}
}

// handlePermissionRequest hands the request to the subscribed UI (exactly
// as a local session's grantChan does) and, once the UI resolves it,
// forwards the chosen option back to the server as a "permissionResponse"
// -- the server-side ACPSession.RequestPermission is the one actually
// blocking a real tool call, waiting for that.
func (rs *RemoteChatSession) handlePermissionRequest(req acp.RequestPermissionRequest) {
	rs.subMu.Lock()
	sub := rs.sub
	rs.subMu.Unlock()
	if sub == nil {
		return
	}

	respCh := make(chan acp.PermissionOptionId, 1)
	go func() {
		select {
		case optID := <-respCh:
			_ = rs.send(agentWireClientMsg{Type: "permissionResponse", OptionID: string(optID)})
		case <-rs.done:
		}
	}()

	sub.OnRequestPermission(PermissionGrantRequest{Req: req, ResponseChan: respCh})
}

// deliverTurnResult unblocks whichever Prompt() call is currently
// waiting, if any -- see failPendingTurn for the connection-lost case.
func (rs *RemoteChatSession) deliverTurnResult(msg agentWireServerMsg) {
	rs.turnMu.Lock()
	ch := rs.turnResult
	rs.turnMu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- msg:
	default:
	}
}

func (rs *RemoteChatSession) failPendingTurn(err error) {
	rs.deliverTurnResult(agentWireServerMsg{Type: "error", Message: err.Error()})
}

func (rs *RemoteChatSession) send(msg agentWireClientMsg) error {
	return wsjson.Write(context.Background(), rs.conn, msg)
}

// --- ChatSession ---

func (rs *RemoteChatSession) ID() string         { return rs.id }
func (rs *RemoteChatSession) WorkingDir() string { return rs.workingDir }
func (rs *RemoteChatSession) Title() string      { return rs.title }
func (rs *RemoteChatSession) Active() bool       { return rs.active.Load() }

func (rs *RemoteChatSession) AgentInfo() acp.Implementation {
	if !rs.Active() {
		return acp.Implementation{}
	}
	return acp.Implementation{Name: "Remote agent"}
}

func (rs *RemoteChatSession) Usage() UsageUpdate { return UsageUpdate{} }

func (rs *RemoteChatSession) HasOngoingTurn() bool {
	rs.turnMu.Lock()
	defer rs.turnMu.Unlock()
	return rs.turnPending
}

func (rs *RemoteChatSession) AvailableCommands() []acp.AvailableCommand { return nil }

func (rs *RemoteChatSession) ConfigOptions() []acp.SessionConfigOption {
	rs.configMu.Lock()
	defer rs.configMu.Unlock()
	return rs.configOptions
}

func (rs *RemoteChatSession) UpdateConfig(ctx context.Context, configID acp.SessionConfigId, value any) error {
	var strVal string
	switch v := value.(type) {
	case bool:
		if v {
			strVal = "true"
		} else {
			strVal = "false"
		}
	case acp.SessionConfigValueId:
		strVal = string(v)
	case string:
		strVal = v
	default:
		return fmt.Errorf("remote agent: unsupported config value type %T", value)
	}
	return rs.send(agentWireClientMsg{Type: "setConfigOption", ConfigID: string(configID), Value: strVal})
}

// Prompt sends contents as a new turn and blocks for its result, matching
// ACPSession.Prompt's contract. If a turn from THIS client is already
// ongoing, it still sends (the server's own ACPSession.Prompt buffers
// content arriving mid-turn and folds it into the current one -- see
// session.go), but returns ErrRemotePromptBuffered immediately instead of
// waiting for a turnEnd that won't be this call's, matching how
// agent/view/chat.go already treats ErrPromptBuffered as a non-error.
func (rs *RemoteChatSession) Prompt(ctx context.Context, contents ...acp.ContentBlock) (PromptResponse, error) {
	promptMsg := agentWireClientMsg{Type: "prompt", Text: contentBlocksToText(contents), Images: contentBlocksToImages(contents)}

	rs.turnMu.Lock()
	if rs.turnPending {
		rs.turnMu.Unlock()
		if err := rs.send(promptMsg); err != nil {
			return PromptResponse{}, err
		}
		return PromptResponse{}, ErrRemotePromptBuffered
	}
	rs.turnPending = true
	resultCh := make(chan agentWireServerMsg, 1)
	rs.turnResult = resultCh
	rs.turnMu.Unlock()

	defer func() {
		rs.turnMu.Lock()
		rs.turnPending = false
		rs.turnResult = nil
		rs.turnMu.Unlock()
	}()

	if err := rs.send(promptMsg); err != nil {
		return PromptResponse{}, err
	}

	select {
	case msg := <-resultCh:
		switch msg.Type {
		case "turnEnd":
			var resp PromptResponse
			_ = json.Unmarshal(msg.Data, &resp)
			return resp, nil
		case "authRequired":
			return PromptResponse{}, errors.New("the remote agent needs re-authentication -- sign in again on the remote server")
		default: // "error"
			return PromptResponse{}, errors.New(msg.Message)
		}
	case <-ctx.Done():
		return PromptResponse{}, ctx.Err()
	case <-rs.done:
		return PromptResponse{}, errors.New("remote agent: connection closed")
	}
}

func (rs *RemoteChatSession) Cancel(ctx context.Context) error {
	return rs.send(agentWireClientMsg{Type: "cancel"})
}

func (rs *RemoteChatSession) SubscribeUpdates(ctx context.Context, sub SessionUpdateSubsciber) {
	rs.subMu.Lock()
	rs.sub = sub
	rs.subMu.Unlock()
}

// GetTerminal always returns nil -- terminal output isn't (yet) forwarded
// over /ws/agent, so agent/view/toolcall.go degrades to a plain
// "Terminal: <id>" label, same as it does for any unknown terminal.
func (rs *RemoteChatSession) GetTerminal(terminalID string) *ACPTerminal { return nil }

// Close disconnects from the remote server. Safe to call more than once.
// The server-side session itself is left running -- other clients (e.g.
// a browser tab) may still be attached to it.
func (rs *RemoteChatSession) Close() {
	_ = rs.conn.Close(websocket.StatusNormalClosure, "")
}

func contentBlocksToText(blocks []acp.ContentBlock) string {
	var sb strings.Builder
	for _, b := range blocks {
		if b.Text != nil {
			if sb.Len() > 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(b.Text.Text)
		}
	}
	return sb.String()
}

func contentBlocksToImages(blocks []acp.ContentBlock) []agentWireImage {
	var out []agentWireImage
	for _, b := range blocks {
		if b.Image != nil {
			out = append(out, agentWireImage{Data: b.Image.Data, MimeType: b.Image.MimeType})
		}
	}
	return out
}
