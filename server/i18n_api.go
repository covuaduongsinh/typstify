package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"looz.ws/typstify/i18n"
)

type i18nRequest struct {
	Locale string   `json:"locale"`
	Keys   []string `json:"keys"`
}

// handleI18n translates a list of UI strings using the same message
// catalog the desktop app uses (i18n/translations, generated from
// golang.org/x/text) -- see docs/plans/plan_web_version.md Giai doan 5.
// Keys with no catalog entry for the requested locale fall back to
// themselves, identically to how the desktop app's Localizer.Translate
// behaves for an untranslated string. Unauthenticated: translated UI copy
// isn't sensitive, and the login page needs it before a session exists.
func (s *Server) handleI18n(w http.ResponseWriter, r *http.Request) {
	var req i18nRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// i18n.Locales IDs are lowercase (e.g. "zh-cn"); Get() compares case-
	// sensitively, so normalize a browser-style locale tag ("zh-CN") first.
	loc, _ := i18n.Get(strings.ToLower(req.Locale))
	result := make(map[string]string, len(req.Keys))
	for _, k := range req.Keys {
		result[k] = loc.Translate(k)
	}
	writeJSON(w, http.StatusOK, result)
}
