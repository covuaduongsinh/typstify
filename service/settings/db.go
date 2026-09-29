package settings

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"looz.ws/typstify/service/bus"
)

type Settings struct {
	store    *settingsStore
	eventbus *bus.EventBus

	// mu guards the lazy-init + reload-on-every-call accessors below
	// (General, Editor, Typst, Lsp, Tpix, AcpAgent). Each one mutates a
	// cached model shared across every caller and reloads it from disk on
	// every call -- without a lock, two goroutines calling the same
	// accessor concurrently (e.g. two /ws/agent connections both calling
	// AcpAgent() from ServiceFacade.StartACPSession) race on that shared
	// model's fields. Confirmed with `go run -race`.
	mu sync.Mutex

	general  *GeneralSettings
	editor   *EditorSettings
	typst    *TypstSettings
	lsp      *LspSettings
	tpix     *TpixSettings
	acpAgent *AcpAgentSettings
	dropbox  *DropboxSettings
	remote   *RemoteSettings
}

func configRoot() string {
	path, err := os.UserConfigDir()
	if err != nil {
		log.Println("Cannot determine system config dir, use user home instead.", err)
		if home, err2 := os.UserHomeDir(); err2 == nil {
			path = home
		}
	}

	configPath := filepath.Join(path, "/typstify")

	err = os.MkdirAll(configPath, 0700)
	if err != nil {
		log.Fatalln("init failed: ", err)
	}

	return configPath
}

func NewSettings(bus *bus.EventBus) *Settings {
	return newSettings(configRoot(), bus)
}

func newSettings(rootDir string, bus *bus.EventBus) *Settings {
	return &Settings{
		store:    newSettingsStore(rootDir),
		eventbus: bus,
	}
}

func (s *Settings) Close() {
}

func (s *Settings) General() *GeneralSettings {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.general == nil {
		s.general = &GeneralSettings{
			baseModel: s.initModel("general"),
		}
	}

	s.general.Load()
	return s.general
}

func (s *Settings) Editor() *EditorSettings {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.editor == nil {
		s.editor = &EditorSettings{
			baseModel: s.initModel("editor"),
		}
	}

	s.editor.Load()

	return s.editor
}

func (s *Settings) Typst() *TypstSettings {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.typst == nil {
		s.typst = &TypstSettings{
			baseModel: s.initModel("typst"),
		}
	}

	s.typst.Load()
	return s.typst
}

func (s *Settings) Lsp() *LspSettings {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lsp == nil {
		s.lsp = &LspSettings{
			baseModel: s.initModel("lsp"),
		}
	}

	s.lsp.Load()
	return s.lsp
}

func (s *Settings) Tpix() *TpixSettings {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tpix == nil {
		s.tpix = &TpixSettings{
			baseModel: s.initModel("tpix"),
		}
	}

	s.tpix.Load()
	return s.tpix
}

func (s *Settings) AcpAgent() *AcpAgentSettings {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.acpAgent == nil {
		s.acpAgent = &AcpAgentSettings{
			baseModel: s.initModel("acpAgent"),
		}
	}

	s.acpAgent.Load()
	return s.acpAgent
}

func (s *Settings) Dropbox() *DropboxSettings {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dropbox == nil {
		s.dropbox = &DropboxSettings{
			baseModel: s.initModel("dropbox"),
		}
	}

	s.dropbox.Load()
	return s.dropbox
}

// Meta returns, for every settings section, when it was last written --
// either by a local edit or by applying a value pulled from a remote
// Typstify instance. Used by the settings-sync feature (desktop<->web) to
// decide which side of each section is newer.
func (s *Settings) Meta() map[string]time.Time {
	return s.store.metaSnapshot()
}

func (s *Settings) Remote() *RemoteSettings {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.remote == nil {
		s.remote = &RemoteSettings{
			baseModel: s.initModel("remote"),
		}
	}

	s.remote.Load()
	return s.remote
}

func (s *Settings) initModel(name string) baseModel {
	return baseModel{name: name, store: s.store, onSave: func(model Model) {
		if s.eventbus != nil {
			s.eventbus.Emit(bus.TopicSettingsUpdated, model)
		}
	}}
}
