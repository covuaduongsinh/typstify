package server

import "net/http"

// handleConsole returns the buffered application/agent console log as plain
// text (same content the desktop app's Console panel shows -- see
// widgets/console). Used by the web frontend to surface login URLs/codes an
// agent CLI prints to stderr during authentication, since ACP's
// Authenticate RPC itself doesn't carry that text -- see server/agent_ws.go
// and docs/plans (AI subscription agents plan, Giai doan 3).
func (s *Server) handleConsole(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(s.appSrv.Console().Text()))
}
