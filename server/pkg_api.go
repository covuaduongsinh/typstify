package server

import (
	"encoding/json"
	"net/http"
)

// handlePkgSearch searches the Typst Package Index (Tpix) for packages and
// templates -- the same search the desktop app's package manager view uses
// (typst/pkg.TypstPkgService.SearchPkgs, ui/pkgmgmt).
func (s *Server) handlePkgSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	pkgs, total, err := s.appSrv.PkgService().SearchPkgs(q.Get("namespace"), q.Get("kind"), q.Get("category"), q.Get("query"))
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"packages": pkgs, "total": total})
}

func (s *Server) handlePkgCached(w http.ResponseWriter, r *http.Request) {
	pkgs, err := s.appSrv.PkgService().CachedPkgs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pkgs)
}

func (s *Server) handlePkgDetail(w http.ResponseWriter, r *http.Request) {
	spec := r.URL.Query().Get("spec")
	if spec == "" {
		writeError(w, http.StatusBadRequest, "spec is required")
		return
	}
	pkg, err := s.appSrv.PkgService().GetPkgDetail(spec)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pkg)
}

type downloadPkgRequest struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Version   string `json:"version"`
}

func (s *Server) handlePkgDownload(w http.ResponseWriter, r *http.Request) {
	var req downloadPkgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Namespace == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "namespace and name are required")
		return
	}

	path, depCount, err := s.appSrv.PkgService().Download(req.Namespace, req.Name, req.Version)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": path, "dependencyCount": depCount})
}

// handlePkgPullDeps downloads every package the open project's typ files
// depend on into the local package cache (typst/pkg's tpix-backed
// dependency resolver), so compilation/preview doesn't have to fetch them
// on the fly.
func (s *Server) handlePkgPullDeps(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err := s.appSrv.PkgService().PullDependencies(root); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
