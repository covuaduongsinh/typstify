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
