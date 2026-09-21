package server

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"looz.ws/typstify/service/settings"
	"looz.ws/typstify/utils"
)

// handleAgentRegistry proxies the ACP agent registry (already fetched and
// cached in-memory by settings.FetchAgentRegistry -- same code path the
// desktop UI's agent picker uses).
func (s *Server) handleAgentRegistry(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	reg, err := settings.FetchAgentRegistry(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reg)
}

type selectAgentRequest struct {
	AgentID string `json:"agentId"`
}

// handleAgentSelect resolves a registry entry into AcpAgentSettings
// (Cmd/Args/Env) via the same settings.ResolveAgentCommand the desktop UI
// uses, and saves it. For agents whose distribution kind resolved to
// nothing usable (e.g. a binary agent with no archive available for this
// platform), it reports that clearly rather than silently saving an empty
// command.
func (s *Server) handleAgentSelect(w http.ResponseWriter, r *http.Request) {
	var req selectAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	entry := settings.LookupAgent(req.AgentID)
	if entry == nil {
		writeError(w, http.StatusNotFound, "unknown agent id (call GET /api/agent/registry first): "+req.AgentID)
		return
	}

	resolved := settings.ResolveAgentCommand(entry)
	if resolved.Cmd == "" {
		writeError(w, http.StatusUnprocessableEntity,
			"this agent has no usable distribution for this server's platform/architecture")
		return
	}

	// LookupExecutable never returns an error -- on failure it just logs a
	// warning and echoes resolved.Cmd back unresolved (utils/executable.go),
	// so a successful lookup is the only case that yields an absolute path.
	// Catch a bad cmd here, at selection time, instead of only discovering it
	// when the agent actually gets spawned (a raw "executable file not found"
	// surfacing as a generic WebSocket disconnect in the chat UI).
	if !filepath.IsAbs(utils.LookupExecutable(resolved.Cmd)) {
		writeError(w, http.StatusUnprocessableEntity,
			"executable not found on this server: \""+resolved.Cmd+"\" -- install it, or add it to PATH, then try again")
		return
	}

	agentSettings := s.appSrv.Settings().AcpAgent()
	agentSettings.AgentID = entry.ID
	agentSettings.AgentName = entry.Name
	agentSettings.Cmd = resolved.Cmd
	agentSettings.Args = resolved.Args
	// Always overwrite, including with "": leaving a previous agent's Env
	// (e.g. antigravity-acp's TEMP/TMP override) in place when switching to
	// one that doesn't need any is the same class of bug as the Args one
	// documented in service/settings/base.go -- a leftover value from
	// before, silently reused.
	agentSettings.Env = resolved.Env

	if err := agentSettings.Save(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, agentSettings)
}

// handleAgentAuth drives the ACP Authenticate RPC for one auth method. It
// blocks until the agent responds -- for an OAuth-device-flow method this
// can take as long as the user needs to open the login link the agent
// prints to stderr (see GET /api/console) and complete it, so the frontend
// should call this from a WebSocket-driven async flow and not block its own
// UI on it (see server/agent_ws.go's startSessionOrRequireAuth /
// "authRequired" and "retryAuth" messages).
func (s *Server) handleAgentAuth(w http.ResponseWriter, r *http.Request) {
	methodID := r.PathValue("methodId")
	if methodID == "" {
		writeError(w, http.StatusBadRequest, "methodId is required")
		return
	}

	mgr := s.appSrv.AcpSessionManager()
	if mgr == nil {
		writeError(w, http.StatusConflict, "no agent connection is active")
		return
	}

	if err := mgr.Authenticate(r.Context(), methodID); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type savePreferredConfigRequest struct {
	ConfigID string `json:"configId"`
	Value    string `json:"value"`
}

// handleSavePreferredConfig persists a session config choice (e.g. the
// model or mode picker) so it's re-applied to every future session via
// server/agent_ws.go's applyPreferredConfig, instead of resetting to the
// agent's own default each time a new one spawns.
func (s *Server) handleSavePreferredConfig(w http.ResponseWriter, r *http.Request) {
	var req savePreferredConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ConfigID == "" {
		writeError(w, http.StatusBadRequest, "configId is required")
		return
	}

	agentSettings := s.appSrv.Settings().AcpAgent()
	preferred := parsePreferredConfig(agentSettings.PreferredConfig)
	preferred[req.ConfigID] = req.Value

	pairs := make([]string, 0, len(preferred))
	for id, value := range preferred {
		pairs = append(pairs, id+"="+value)
	}
	sort.Strings(pairs) // deterministic on-disk order, easier to diff/read
	agentSettings.PreferredConfig = strings.Join(pairs, " ")

	if err := agentSettings.Save(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
