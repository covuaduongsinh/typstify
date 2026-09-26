package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"looz.ws/typstify/service/dropbox"
)

type dropboxStatusResponse struct {
	Connected        bool                 `json:"connected"`
	Account          *dropbox.AccountInfo `json:"account,omitempty"`
	AppKey           string               `json:"appKey,omitempty"`
	SyncFolder       string               `json:"syncFolder"`
	AutoSync         bool                 `json:"autoSync"`
	AutoSyncInterval int                  `json:"autoSyncInterval"`
	SyncOnSave       bool                 `json:"syncOnSave"`
	LastSyncTime     *time.Time           `json:"lastSyncTime,omitempty"`
	IsSyncing        bool                 `json:"isSyncing"`
	LastResult       *dropbox.SyncResult  `json:"lastResult,omitempty"`
}

func (s *Server) handleDropboxStatus(w http.ResponseWriter, r *http.Request) {
	st := s.appSrv.Settings().Dropbox()
	syncer := s.appSrv.DropboxSyncer()

	resp := dropboxStatusResponse{
		Connected:        st.IsConnected(),
		AppKey:           st.AppKey,
		SyncFolder:       st.SyncFolder,
		AutoSync:         st.AutoSync,
		AutoSyncInterval: st.AutoSyncInterval,
		SyncOnSave:       st.SyncOnSave,
		IsSyncing:        syncer.IsSyncing(),
		LastResult:       syncer.LastResult(),
	}

	if !st.LastSyncTime.IsZero() {
		resp.LastSyncTime = &st.LastSyncTime
	}

	if st.IsConnected() {
		resp.Account = &dropbox.AccountInfo{
			AccountID:   st.AccountID,
			DisplayName: st.AccountName,
			Email:       st.AccountEmail,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

type dropboxAuthURLRequest struct {
	AppKey      string `json:"appKey"`
	RedirectURI string `json:"redirectUri"`
}

func (s *Server) handleDropboxAuthURL(w http.ResponseWriter, r *http.Request) {
	var req dropboxAuthURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}

	appKey := req.AppKey
	if appKey == "" {
		appKey = s.appSrv.Settings().Dropbox().AppKey
	}
	if appKey == "" {
		writeError(w, http.StatusBadRequest, "vui lòng cung cấp Dropbox App Key")
		return
	}

	u, _ := url.Parse("https://www.dropbox.com/oauth2/authorize")
	q := u.Query()
	q.Set("client_id", appKey)
	q.Set("response_type", "code")
	q.Set("token_access_type", "offline")
	if req.RedirectURI != "" {
		q.Set("redirect_uri", req.RedirectURI)
	}
	u.RawQuery = q.Encode()

	writeJSON(w, http.StatusOK, map[string]string{"url": u.String()})
}

type dropboxAuthCallbackRequest struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirectUri"`
	AppKey      string `json:"appKey"`
	AppSecret   string `json:"appSecret"`
}

func (s *Server) handleDropboxAuthCallback(w http.ResponseWriter, r *http.Request) {
	var req dropboxAuthCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}

	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "thiếu authorization code")
		return
	}

	client := s.appSrv.DropboxClient()
	acc, err := client.ExchangeAuthCode(r.Context(), req.Code, req.RedirectURI, req.AppKey, req.AppSecret)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"account": acc,
	})
}

type dropboxSaveTokenRequest struct {
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	AppKey           string `json:"appKey"`
	AppSecret        string `json:"appSecret"`
	SyncFolder       string `json:"syncFolder"`
	AutoSync         *bool  `json:"autoSync"`
	AutoSyncInterval *int   `json:"autoSyncInterval"`
	SyncOnSave       *bool  `json:"syncOnSave"`
}

func (s *Server) handleDropboxSaveToken(w http.ResponseWriter, r *http.Request) {
	var req dropboxSaveTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}

	st := s.appSrv.Settings().Dropbox()
	if req.AccessToken != "" {
		st.AccessToken = req.AccessToken
	}
	if req.RefreshToken != "" {
		st.RefreshToken = req.RefreshToken
	}
	if req.AppKey != "" {
		st.AppKey = req.AppKey
	}
	if req.AppSecret != "" {
		st.AppSecret = req.AppSecret
	}
	if req.SyncFolder != "" {
		st.SyncFolder = req.SyncFolder
	}
	if req.AutoSync != nil {
		st.AutoSync = *req.AutoSync
	}
	if req.AutoSyncInterval != nil && *req.AutoSyncInterval > 0 {
		st.AutoSyncInterval = *req.AutoSyncInterval
	}
	if req.SyncOnSave != nil {
		st.SyncOnSave = *req.SyncOnSave
	}

	_ = st.Save()

	client := s.appSrv.DropboxClient()
	if st.IsConnected() {
		acc, err := client.GetCurrentAccount(r.Context())
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("kết nối Dropbox thất bại: %v", err))
			return
		}
		st.AccountID = acc.AccountID
		st.AccountName = acc.DisplayName
		st.AccountEmail = acc.Email
		_ = st.Save()
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "account": acc})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDropboxDisconnect(w http.ResponseWriter, r *http.Request) {
	st := s.appSrv.Settings().Dropbox()
	st.Clear()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type dropboxSyncRequest struct {
	Mode string `json:"mode"` // "two-way", "push", "pull"
}

func (s *Server) handleDropboxSync(w http.ResponseWriter, r *http.Request) {
	projDir := s.appSrv.CurrentProjectDir()
	if projDir == "" {
		writeError(w, http.StatusConflict, "chưa có dự án nào đang mở")
		return
	}

	st := s.appSrv.Settings().Dropbox()
	if !st.IsConnected() {
		writeError(w, http.StatusBadRequest, "chưa kết nối tài khoản Dropbox")
		return
	}

	var req dropboxSyncRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	mode := dropbox.SyncModeTwoWay
	switch strings.ToLower(req.Mode) {
	case "push":
		mode = dropbox.SyncModePush
	case "pull":
		mode = dropbox.SyncModePull
	default:
		mode = dropbox.SyncModeTwoWay
	}

	syncer := s.appSrv.DropboxSyncer()
	res, err := syncer.SyncProject(r.Context(), projDir, mode)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleDropboxListProjects(w http.ResponseWriter, r *http.Request) {
	st := s.appSrv.Settings().Dropbox()
	if !st.IsConnected() {
		writeError(w, http.StatusBadRequest, "chưa kết nối tài khoản Dropbox")
		return
	}

	client := s.appSrv.DropboxClient()
	rootFolder := st.SyncFolder
	if rootFolder == "" {
		rootFolder = "/Typstify"
	}

	entries, err := client.ListFolder(r.Context(), rootFolder, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type projectItem struct {
		Name           string    `json:"name"`
		Path           string    `json:"path"`
		ServerModified time.Time `json:"server_modified"`
	}

	var projects []projectItem
	for _, e := range entries {
		if e.Tag == "folder" {
			projects = append(projects, projectItem{
				Name:           e.Name,
				Path:           e.PathDisplay,
				ServerModified: e.ServerModified,
			})
		}
	}

	writeJSON(w, http.StatusOK, projects)
}

type dropboxImportRequest struct {
	RemotePath string `json:"remotePath"`
	TargetDir  string `json:"targetDir"`
}

func (s *Server) handleDropboxImport(w http.ResponseWriter, r *http.Request) {
	var req dropboxImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}

	if req.RemotePath == "" {
		writeError(w, http.StatusBadRequest, "thiếu remotePath")
		return
	}

	projName := filepath.Base(req.RemotePath)
	targetDir := req.TargetDir
	if targetDir == "" {
		if s.opts.ProjectRoot != "" {
			targetDir = filepath.Join(s.opts.ProjectRoot, projName)
		} else {
			targetDir = filepath.Join(s.appSrv.Settings().General().RootDir, projName)
		}
	}

	abs, err := filepath.Abs(targetDir)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.checkProjectRoot(abs); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := os.MkdirAll(abs, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	client := s.appSrv.DropboxClient()
	entries, err := client.ListFolder(r.Context(), req.RemotePath, true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	downloadCount := 0
	for _, entry := range entries {
		if entry.Tag == "file" {
			rel := strings.TrimPrefix(entry.PathDisplay, req.RemotePath+"/")
			localFile := filepath.Join(abs, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(localFile), 0755); err != nil {
				continue
			}

			rc, _, err := client.DownloadFile(r.Context(), entry.PathDisplay)
			if err != nil {
				continue
			}
			f, err := os.Create(localFile)
			if err != nil {
				rc.Close()
				continue
			}
			_, _ = io.Copy(f, rc)
			f.Close()
			rc.Close()
			downloadCount++
		}
	}

	s.appSrv.SetProjectDir(abs)
	writeJSON(w, http.StatusOK, map[string]any{
		"path":       abs,
		"downloaded": downloadCount,
	})
}
