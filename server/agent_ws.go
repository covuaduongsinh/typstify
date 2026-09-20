package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	acp "github.com/coder/acp-go-sdk"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"looz.ws/typstify/agent"
)

const sessionCloseTimeout = 10 * time.Second

// agentClientMessage is the wire format for browser -> server messages on
// /ws/agent.
type agentClientMessage struct {
	Type     string            `json:"type"`
	Text     string            `json:"text,omitempty"`
	OptionID string            `json:"optionId,omitempty"`
	Images   []imageAttachment `json:"images,omitempty"`
	ConfigID string            `json:"configId,omitempty"`
	Value    string            `json:"value,omitempty"`
}

// imageAttachment is a base64-encoded image pasted into the chat input,
// matching the desktop client's acp.ImageBlock construction
// (agent/view/inputbox.go).
type imageAttachment struct {
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

// agentAuthRequiredData is the payload of an "authRequired" server message.
type agentAuthRequiredData struct {
	AgentName   string           `json:"agentName"`
	AuthMethods []acp.AuthMethod `json:"authMethods"`
}

// agentServerMessage is the wire format for server -> browser messages. Data
// carries the underlying ACP type as-is (agent.UserMessageChunk,
// agent.ToolCall, acp.RequestPermissionRequest, ...) -- these already have
// the JSON shape defined by the Agent Client Protocol wire spec, since
// package agent exchanges them over JSON-RPC with the agent CLI, so no
// reshaping is needed for the browser to render them.
type agentServerMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId,omitempty"`
	Data      any    `json:"data,omitempty"`
	Message   string `json:"message,omitempty"`
}

// wsSubscriber adapts agent.SessionUpdateSubsciber to push every ACP session
// update straight to the browser. It only supports one outstanding
// permission request at a time per connection -- ACP turns are sequential in
// practice for the agents Typstify targets, so this is a reasonable v1
// simplification (see docs/plans/plan_web_version.md Giai doan 3).
type wsSubscriber struct {
	ctx  context.Context
	conn *websocket.Conn

	mu      sync.Mutex
	pending chan acp.PermissionOptionId
}

func (w *wsSubscriber) send(kind string, data any) {
	if err := wsjson.Write(w.ctx, w.conn, agentServerMessage{Type: kind, Data: data}); err != nil {
		log.Printf("agent_ws: write failed: %v", err)
	}
}

func (w *wsSubscriber) OnUserMessage(chunk agent.UserMessageChunk)   { w.send("userMessage", chunk) }
func (w *wsSubscriber) OnAgentMessage(chunk agent.AgentMessageChunk) { w.send("agentMessage", chunk) }
func (w *wsSubscriber) OnAgentThought(chunk agent.AgentThoughtChunk) { w.send("agentThought", chunk) }
func (w *wsSubscriber) OnToolCallInit(tc agent.ToolCall)             { w.send("toolCall", tc) }
func (w *wsSubscriber) OnToolCallUpdate(tc agent.ToolCallUpdate)     { w.send("toolCallUpdate", tc) }
func (w *wsSubscriber) OnPlan(plan agent.Plan)                       { w.send("plan", plan) }
func (w *wsSubscriber) OnConfigOptionUpdate(update agent.ConfigOptionUpdate) {
	w.send("configOptions", update.ConfigOptions)
}

func (w *wsSubscriber) OnRequestPermission(req agent.PermissionGrantRequest) {
	ch := make(chan acp.PermissionOptionId, 1)

	w.mu.Lock()
	w.pending = ch
	w.mu.Unlock()

	w.send("permissionRequest", req.Req)

	select {
	case optID := <-ch:
		req.ResponseChan <- optID
	case <-w.ctx.Done():
		// Connection is gone; nothing we can send back, the ACP session's
		// own cancellation/cleanup will unblock the caller.
	}
}

// resolvePermission delivers the browser's chosen option to a pending
// OnRequestPermission call, if any.
func (w *wsSubscriber) resolvePermission(optID acp.PermissionOptionId) {
	w.mu.Lock()
	ch := w.pending
	w.pending = nil
	w.mu.Unlock()

	if ch == nil {
		return
	}
	select {
	case ch <- optID:
	default:
	}
}

// agentAuthRequiredData builds the payload for an "authRequired" server
// message from whatever agent is currently connected (if any).
func (s *Server) agentAuthRequiredData() agentAuthRequiredData {
	data := agentAuthRequiredData{}
	if mgr := s.appSrv.AcpSessionManager(); mgr != nil && mgr.AgentConn() != nil {
		data.AgentName = mgr.AgentConn().AgentInfo.Name
		data.AuthMethods = mgr.AgentConn().AuthMethods
	}
	return data
}

// startSessionOrRequireAuth tries to start an ACP session, and if the agent
// reports it needs authentication (agent.AuthRequiredErr), sends an
// "authRequired" message with the agent's AuthMethods and then waits for
// the browser to send "retryAuth" (after driving POST /api/agent/auth/...
// and/or the SetupIntent-style out-of-band login flow the agent itself
// prints instructions for -- see GET /api/console) before trying again.
// Returns (nil, false) if the connection closes/context is cancelled first.
func (s *Server) startSessionOrRequireAuth(ctx context.Context, conn *websocket.Conn, root string) (*agent.ACPSession, bool) {
	for {
		session, err := s.appSrv.StartACPSession(ctx, root)
		if err == nil {
			return session, true
		}

		// The agent's session manager was torn down because the open project
		// changed while this connection was starting up. This connection was
		// opened for the old project root, so retrying it here would attach
		// a session for the wrong cwd to whatever agent is running now --
		// the frontend already opens a fresh /ws/agent connection for the
		// new project (AgentChat remounts on projectPath change), so just
		// close this one with a message that tells the user why, instead of
		// the generic "connection was lost" text.
		if errors.Is(err, agent.ErrSessionManagerClosed) {
			_ = wsjson.Write(context.Background(), conn, agentServerMessage{Type: "error", Message: "Project changed while connecting to the agent. Switch back to this project to reconnect."})
			_ = conn.Close(websocket.StatusNormalClosure, "project changed")
			return nil, false
		}

		if !errors.Is(err, agent.AuthRequiredErr) {
			_ = wsjson.Write(context.Background(), conn, agentServerMessage{Type: "error", Message: err.Error()})
			_ = conn.Close(websocket.StatusInternalError, "failed to start agent session")
			return nil, false
		}

		if err := wsjson.Write(ctx, conn, agentServerMessage{Type: "authRequired", Data: s.agentAuthRequiredData()}); err != nil {
			return nil, false
		}

		// Wait for the browser to ask us to retry (after it drove
		// POST /api/agent/auth/{methodId}), ignoring any other message type
		// that might arrive while we're gated on auth.
		for {
			var msg agentClientMessage
			if err := wsjson.Read(ctx, conn, &msg); err != nil {
				return nil, false
			}
			if msg.Type == "retryAuth" {
				break
			}
		}
	}
}

// handleAgentWS starts (or reuses) an ACP session against the currently
// configured AI agent for the open project, and bridges it to the browser
// over a WebSocket. One session per connection.
func (s *Server) handleAgentWS(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: s.wsOriginPatterns(r)})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	// The default 32KiB read limit (coder/websocket) is well under a single
	// pasted screenshot once base64-encoded -- every such paste silently
	// killed the connection (wsjson.Read erroring out of the loop below with
	// no message, surfacing to the browser as a generic "Connection to the
	// agent was lost"). 20MiB comfortably covers realistic screenshots.
	conn.SetReadLimit(20 << 20)

	sessCtx, cancel := context.WithCancel(r.Context())
	defer cancel()

	session, ok := s.startSessionOrRequireAuth(sessCtx, conn, root)
	if !ok {
		return
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), sessionCloseTimeout)
		defer closeCancel()
		// session/cancel is a base ACP capability every agent must support,
		// unlike session/close (gated by an optional capability -- see
		// CloseACPSession/CloseSession, a no-op when the agent doesn't
		// declare it). Send it whenever a turn is still in flight so the
		// agent process itself learns the browser is gone, regardless of
		// whether it supports close -- otherwise an abandoned turn (tab
		// closed, reload, network drop) lingers server-side forever with
		// nothing to tell the agent to stop, and on a long-lived
		// multi-session agent process this accumulates across
		// disconnects. Best-effort: the agent may already be gone too.
		if session.HasOngoingTurn() {
			_ = session.Cancel(closeCtx)
		}
		_ = s.appSrv.CloseACPSession(closeCtx, session.SessionID)
	}()

	sub := &wsSubscriber{ctx: sessCtx, conn: conn}
	session.SubscribeUpdates(sessCtx, sub)

	_ = wsjson.Write(sessCtx, conn, agentServerMessage{Type: "ready", SessionID: session.SessionID})
	if opts := session.ConfigOptions(); len(opts) > 0 {
		_ = wsjson.Write(sessCtx, conn, agentServerMessage{Type: "configOptions", Data: opts})
	}

	for {
		var msg agentClientMessage
		if err := wsjson.Read(sessCtx, conn, &msg); err != nil {
			return
		}

		switch msg.Type {
		case "prompt":
			go s.runPrompt(sessCtx, conn, session, msg.Text, msg.Images)
		case "setConfigOption":
			go func(configID, value string) {
				if err := session.UpdateConfig(sessCtx, acp.SessionConfigId(configID), acp.SessionConfigValueId(value)); err != nil {
					_ = wsjson.Write(sessCtx, conn, agentServerMessage{Type: "error", Message: err.Error()})
					return
				}
				// UpdateConfig's own RPC response already updated
				// session.configOptions -- push it now instead of waiting on
				// an agent-initiated session/update notification, since not
				// every agent sends one for a change it already applied and
				// returned synchronously (observed: Claude Code doesn't).
				_ = wsjson.Write(sessCtx, conn, agentServerMessage{Type: "configOptions", Data: session.ConfigOptions()})
			}(msg.ConfigID, msg.Value)
		case "cancel":
			go func() { _ = session.Cancel(sessCtx) }()
		case "permissionResponse":
			sub.resolvePermission(acp.PermissionOptionId(msg.OptionID))
		case "retryAuth":
			// The auth-required error surfaced from a Prompt() call, after
			// the session itself was already established (unlike Claude
			// Code, most agents report AuthRequiredErr at session start --
			// see startSessionOrRequireAuth -- but Claude Code's ACP bridge
			// only discovers it's logged out once it actually tries to
			// query the model). The frontend's AuthCard is showing because
			// it got an "authRequired" message; re-sending "ready" is what
			// clears that state (AgentChat.tsx only clears it on "ready"),
			// letting the user resend their prompt now that login (driven
			// via POST /api/agent/auth/{methodId}, same as the connect-time
			// flow) has hopefully completed. The session/turn itself needs
			// no restart -- only the next Prompt() call did.
			go func() { _ = wsjson.Write(sessCtx, conn, agentServerMessage{Type: "ready", SessionID: session.SessionID}) }()
		default:
			log.Printf("agent_ws: unknown message type %q", msg.Type)
		}
	}
}

func (s *Server) runPrompt(ctx context.Context, conn *websocket.Conn, session *agent.ACPSession, text string, images []imageAttachment) {
	blocks := make([]acp.ContentBlock, 0, 1+len(images))
	if text != "" {
		blocks = append(blocks, acp.TextBlock(text))
	}
	for _, img := range images {
		blocks = append(blocks, acp.ImageBlock(img.Data, img.MimeType))
	}

	resp, err := session.Prompt(ctx, blocks...)
	if err != nil {
		if err == agent.ErrPromptBuffered {
			return
		}
		if errors.Is(err, agent.AuthRequiredErr) {
			_ = wsjson.Write(ctx, conn, agentServerMessage{Type: "authRequired", Data: s.agentAuthRequiredData()})
			return
		}
		_ = wsjson.Write(ctx, conn, agentServerMessage{Type: "error", Message: err.Error()})
		return
	}
	_ = wsjson.Write(ctx, conn, agentServerMessage{Type: "turnEnd", Data: resp})
}
