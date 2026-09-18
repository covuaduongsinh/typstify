package server

import (
	"context"
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
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	OptionID string `json:"optionId,omitempty"`
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

	sessCtx, cancel := context.WithCancel(r.Context())
	defer cancel()

	session, err := s.appSrv.StartACPSession(sessCtx, root)
	if err != nil {
		_ = wsjson.Write(context.Background(), conn, agentServerMessage{Type: "error", Message: err.Error()})
		_ = conn.Close(websocket.StatusInternalError, "failed to start agent session")
		return
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), sessionCloseTimeout)
		defer closeCancel()
		_ = s.appSrv.CloseACPSession(closeCtx, session.SessionID)
	}()

	sub := &wsSubscriber{ctx: sessCtx, conn: conn}
	session.SubscribeUpdates(sessCtx, sub)

	_ = wsjson.Write(sessCtx, conn, agentServerMessage{Type: "ready", SessionID: session.SessionID})

	for {
		var msg agentClientMessage
		if err := wsjson.Read(sessCtx, conn, &msg); err != nil {
			return
		}

		switch msg.Type {
		case "prompt":
			go s.runPrompt(sessCtx, conn, session, msg.Text)
		case "cancel":
			go func() { _ = session.Cancel(sessCtx) }()
		case "permissionResponse":
			sub.resolvePermission(acp.PermissionOptionId(msg.OptionID))
		default:
			log.Printf("agent_ws: unknown message type %q", msg.Type)
		}
	}
}

func (s *Server) runPrompt(ctx context.Context, conn *websocket.Conn, session *agent.ACPSession, text string) {
	resp, err := session.Prompt(ctx, acp.ContentBlock{Text: &acp.ContentBlockText{Type: "text", Text: text}})
	if err != nil {
		if err == agent.ErrPromptBuffered {
			return
		}
		_ = wsjson.Write(ctx, conn, agentServerMessage{Type: "error", Message: err.Error()})
		return
	}
	_ = wsjson.Write(ctx, conn, agentServerMessage{Type: "turnEnd", Data: resp})
}
