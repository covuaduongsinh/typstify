package settings

import "strings"

// VPSSyncSettings cấu hình đồng bộ file một chiều từ máy local lên VPS
// (typstify-server). Token là credential của riêng máy này nên section này
// không tham gia đồng bộ desktop<->web (không implement RemoteApplier).
type VPSSyncSettings struct {
	baseModel

	Enabled        bool     `key:"enabled" json:"enabled"`
	ServerURL      string   `key:"serverUrl" json:"server_url"`
	Username       string   `key:"username" json:"username"`
	Token          string   `key:"token" json:"token"`
	AutoSync       bool     `key:"autoSync" json:"auto_sync"`
	IntervalSec    int      `key:"intervalSec" json:"interval_sec"`
	IgnorePatterns []string `key:"ignorePatterns" json:"ignore_patterns"`
}

var defaultVPSSyncSettings = &VPSSyncSettings{
	IntervalSec:    300,
	IgnorePatterns: []string{},
}

var _ Model = (*VPSSyncSettings)(nil)

func (s *VPSSyncSettings) Validate() error {
	s.ServerURL = strings.TrimRight(strings.TrimSpace(s.ServerURL), "/")
	if s.IntervalSec <= 0 {
		s.IntervalSec = 300
	}
	return nil
}

func (s *VPSSyncSettings) Save() error {
	if err := s.Validate(); err != nil {
		return err
	}
	return s.baseModel.save(s)
}

func (s *VPSSyncSettings) Load() error {
	return s.baseModel.load(s, defaultVPSSyncSettings)
}
