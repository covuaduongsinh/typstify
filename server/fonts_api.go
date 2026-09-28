package server

import (
	"errors"
	"net/http"

	"looz.ws/typstify/service/fonts"
)

// handleFontsList lists uploaded font files (fonts.Dir's managed
// ExtraFontPath directory). Empty (not an error) when no font has ever
// been uploaded and the user hasn't set ExtraFontPath themselves.
func (s *Server) handleFontsList(w http.ResponseWriter, r *http.Request) {
	list, err := fonts.List(s.appSrv.Settings())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleFontUpload saves the raw request body as a font file named by the
// "filename" query param -- a plain binary POST (not multipart/form-data),
// matching this API's existing convention for raw-content writes (e.g. PUT
// /api/workspace/file).
func (s *Server) handleFontUpload(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("filename")
	if _, err := fonts.SanitizeFilename(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, fonts.MaxUploadSize)
	if err := fonts.Upload(s.appSrv.Settings(), name, r.Body); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "upload failed (file may be too large): "+err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleFontDelete removes one uploaded font file.
func (s *Server) handleFontDelete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, err := fonts.SanitizeFilename(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := fonts.Delete(s.appSrv.Settings(), name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
