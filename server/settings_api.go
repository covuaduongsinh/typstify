package server

import (
	"encoding/json"
	"net/http"

	"looz.ws/typstify/service/settings"
)

// settingsGetHandler and settingsPutHandler are generic over the concrete
// settings.Model types (GeneralSettings, TypstSettings, ...) so each section
// only needs one line of wiring in Server.routes.

func settingsGetHandler[T settings.Model](get func() T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, get())
	}
}

func settingsPutHandler[T settings.Model](get func() T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cur := get()

		if err := json.NewDecoder(r.Body).Decode(cur); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := cur.Validate(); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := cur.Save(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, cur)
	}
}

// handleSettingsMeta reports, for every settings section, when it was last
// written -- used by the desktop<->web settings-sync feature to decide
// which side of each section is newer, in one round trip instead of one
// GET per section.
func (s *Server) handleSettingsMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.appSrv.Settings().Meta())
}
