package settings

import "time"

type DropboxSettings struct {
	baseModel

	AppKey           string    `key:"appKey" json:"appKey"`
	AppSecret        string    `key:"appSecret" json:"appSecret"`
	AccessToken      string    `key:"accessToken" json:"accessToken"`
	RefreshToken     string    `key:"refreshToken" json:"refreshToken"`
	TokenExpiry      time.Time `key:"tokenExpiry" json:"tokenExpiry"`
	AccountID        string    `key:"accountId" json:"accountId"`
	AccountName      string    `key:"accountName" json:"accountName"`
	AccountEmail     string    `key:"accountEmail" json:"accountEmail"`
	SyncFolder       string    `key:"syncFolder" json:"syncFolder"` // Remote root folder, e.g. "/Typstify"
	AutoSync         bool      `key:"autoSync" json:"autoSync"`
	AutoSyncInterval int       `key:"autoSyncInterval" json:"autoSyncInterval"` // In minutes (default 10)
	SyncOnSave       bool      `key:"syncOnSave" json:"syncOnSave"`
	LastSyncTime     time.Time `key:"lastSyncTime" json:"lastSyncTime"`
}

var defaultDropboxSettings = &DropboxSettings{
	SyncFolder:       "/Typstify",
	AutoSync:         false,
	AutoSyncInterval: 10,
	SyncOnSave:       false,
}

func (s *DropboxSettings) Validate() error {
	if s.SyncFolder == "" {
		s.SyncFolder = "/Typstify"
	}
	if s.AutoSyncInterval <= 0 {
		s.AutoSyncInterval = 10
	}
	return nil
}

func (s *DropboxSettings) Save() error {
	if err := s.Validate(); err != nil {
		return err
	}
	return s.baseModel.save(s)
}

func (s *DropboxSettings) Load() error {
	return s.baseModel.load(s, defaultDropboxSettings)
}

func (s *DropboxSettings) IsConnected() bool {
	return s.AccessToken != "" || s.RefreshToken != ""
}

func (s *DropboxSettings) Clear() {
	s.AccessToken = ""
	s.RefreshToken = ""
	s.TokenExpiry = time.Time{}
	s.AccountID = ""
	s.AccountName = ""
	s.AccountEmail = ""
	_ = s.Save()
}
