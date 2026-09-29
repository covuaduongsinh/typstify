package settings

// RemoteSettings holds this machine's connection to another self-hosted
// Typstify instance (desktop -> web server, typically), used by the
// settings-sync feature (Giai đoạn A) and later by remote-agent (Giai
// đoạn B) and remote-project (Giai đoạn C) modes. It is intentionally
// never itself synced -- Token is this machine's own credential to the
// remote, syncing it would leak it to (and let it be overwritten by) the
// other side.
type RemoteSettings struct {
	baseModel

	ServerURL string `key:"serverUrl" json:"serverUrl"`
	// Token is a long-lived Bearer token obtained via remote.Client.Login,
	// stored in plaintext -- the same convention DropboxSettings.AccessToken
	// already uses in this same settings.json file, not a new lower
	// security bar.
	Token   string `key:"token" json:"token"`
	Enabled bool   `key:"enabled" json:"enabled"`
	// RemoteAgentEnabled switches AI Agent chat (ui/assistant) from
	// spawning a local agent CLI process to connecting to the agent
	// session already running on ServerURL instead -- see Giai đoạn B.
	RemoteAgentEnabled bool `key:"remoteAgentEnabled" json:"remoteAgentEnabled"`
}

var _ Model = (*RemoteSettings)(nil)

func (r *RemoteSettings) Save() error     { return r.baseModel.save(r) }
func (r *RemoteSettings) Load() error     { return r.baseModel.load(r, &RemoteSettings{}) }
func (r *RemoteSettings) Validate() error { return nil }
