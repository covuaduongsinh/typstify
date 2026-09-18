package server

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"looz.ws/typstify/lsp"
)

// lspClientMessage is the wire format for browser -> server messages on
// /ws/lsp. Path is always project-root-relative and slash-separated.
type lspClientMessage struct {
	Type      string `json:"type"`
	ID        string `json:"id,omitempty"`
	Path      string `json:"path,omitempty"`
	Content   string `json:"content,omitempty"`
	Line      int    `json:"line,omitempty"`
	Character int    `json:"character,omitempty"`
}

// lspServerMessage is the wire format for server -> browser messages.
type lspServerMessage struct {
	Type        string `json:"type"`
	ID          string `json:"id,omitempty"`
	Path        string `json:"path,omitempty"`
	Items       any    `json:"items,omitempty"`
	Hover       any    `json:"hover,omitempty"`
	Symbols     any    `json:"symbols,omitempty"`
	Diagnostics any    `json:"diagnostics,omitempty"`
	Message     string `json:"message,omitempty"`
}

// watchedPaths tracks which project-relative files this WS connection has
// opened, so the diagnostics pump only polls documents the client cares
// about.
type watchedPaths struct {
	mu sync.Mutex
	m  map[string]bool
}

func newWatchedPaths() *watchedPaths { return &watchedPaths{m: make(map[string]bool)} }

func (w *watchedPaths) add(p string) {
	w.mu.Lock()
	w.m[p] = true
	w.mu.Unlock()
}

func (w *watchedPaths) remove(p string) {
	w.mu.Lock()
	delete(w.m, p)
	w.mu.Unlock()
}

func (w *watchedPaths) list() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]string, 0, len(w.m))
	for p := range w.m {
		out = append(out, p)
	}
	return out
}

// handleLspWS bridges one browser editor session to the tinymist LSP client
// already managed by ServiceFacade/lsp.GetLspClient. It reuses lsp.Client's
// existing synchronous Complete/Hover/DocumentSymbols calls (jsonrpc2
// Call+Await under the hood) and lsp.Client's poll-friendly Diagnostics()
// accessor -- no changes to package lsp were needed.
func (s *Server) handleLspWS(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	client := lsp.GetLspClient(root, s.appSrv.Settings())
	if client == nil {
		writeError(w, http.StatusServiceUnavailable, "LSP client is not available (is tinymist installed?)")
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: s.wsOriginPatterns(r)})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	sessCtx, cancel := context.WithCancel(r.Context())
	defer cancel()

	watched := newWatchedPaths()
	go s.pumpDiagnostics(sessCtx, conn, client, watched, root)

	for {
		var msg lspClientMessage
		if err := wsjson.Read(sessCtx, conn, &msg); err != nil {
			return
		}

		switch msg.Type {
		case "didOpen", "didChange":
			p, err := resolveInRoot(root, msg.Path)
			if err != nil {
				continue
			}
			watched.add(msg.Path)
			client.OnEditorUpdated(p, strings.NewReader(msg.Content))

		case "didClose":
			p, err := resolveInRoot(root, msg.Path)
			if err != nil {
				continue
			}
			watched.remove(msg.Path)
			client.OnEditorClosed(p)

		case "didSave":
			p, err := resolveInRoot(root, msg.Path)
			if err != nil {
				continue
			}
			client.OnEditorSaved(p)

		case "complete":
			p, err := resolveInRoot(root, msg.Path)
			if err != nil {
				continue
			}
			items, err := client.Complete(sessCtx, p, msg.Line, msg.Character)
			if err != nil {
				_ = wsjson.Write(sessCtx, conn, lspServerMessage{Type: "error", ID: msg.ID, Message: err.Error()})
				continue
			}
			_ = wsjson.Write(sessCtx, conn, lspServerMessage{Type: "completeResult", ID: msg.ID, Items: items})

		case "hover":
			p, err := resolveInRoot(root, msg.Path)
			if err != nil {
				continue
			}
			hover, err := client.Hover(sessCtx, p, msg.Line, msg.Character)
			if err != nil {
				_ = wsjson.Write(sessCtx, conn, lspServerMessage{Type: "error", ID: msg.ID, Message: err.Error()})
				continue
			}
			_ = wsjson.Write(sessCtx, conn, lspServerMessage{Type: "hoverResult", ID: msg.ID, Hover: hover})

		case "documentSymbols":
			p, err := resolveInRoot(root, msg.Path)
			if err != nil {
				continue
			}
			symbols, err := client.DocumentSymbols(sessCtx, p)
			if err != nil {
				_ = wsjson.Write(sessCtx, conn, lspServerMessage{Type: "error", ID: msg.ID, Message: err.Error()})
				continue
			}
			_ = wsjson.Write(sessCtx, conn, lspServerMessage{Type: "documentSymbolsResult", ID: msg.ID, Symbols: symbols})

		default:
			log.Printf("lsp_ws: unknown message type %q", msg.Type)
		}
	}
}

// pumpDiagnostics polls lsp.Client.Diagnostics for each watched document and
// pushes it to the browser when tinymist has refreshed it. lsp.Client has no
// push/callback API for diagnostics (the desktop UI reads it once per Gio
// frame instead), so polling on a short interval is the least invasive way
// to bridge it to a WebSocket without touching package lsp.
func (s *Server) pumpDiagnostics(ctx context.Context, conn *websocket.Conn, client *lsp.Client, watched *watchedPaths, root string) {
	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, rel := range watched.list() {
				abs, err := resolveInRoot(root, rel)
				if err != nil {
					continue
				}
				diag := client.Diagnostics(abs)
				if diag == nil || !diag.Refreshed() {
					continue
				}
				if err := wsjson.Write(ctx, conn, lspServerMessage{
					Type:        "diagnostics",
					Path:        rel,
					Diagnostics: diag.Diagnostics,
				}); err != nil {
					return
				}
			}
		}
	}
}

// wsOriginPatterns restricts WebSocket upgrades to same-Host requests. The
// frontend is always served from the same origin as this API in the
// self-hosted deployment model, so this is a safe default; a reverse proxy
// that rewrites Host may need this revisited.
func (s *Server) wsOriginPatterns(r *http.Request) []string {
	return []string{r.Host}
}
