// Package server implements the HTTP/WebSocket API that backs Typstify's
// self-hosted web mode (docs/plans/plan_web_version.md, Giai doan 1). It is a
// thin transport layer around the existing service.ServiceFacade, lsp.Client
// and agent.SessionManager -- business logic is not duplicated here.
package server

import (
	"encoding/json"
	"errors"
	"net/http"
)

var errNoProjectOpen = errors.New("no project is open")

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
