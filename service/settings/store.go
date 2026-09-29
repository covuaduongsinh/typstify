package settings

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"

	"looz.ws/typstify/utils"
)

const (
	settingsFileName    = "settings.json"
	legacySettingsDB    = "settings.db"
	settingsFileVersion = 1
	settingsVersionKey  = "version"
	settingsMetaKey     = "__meta__"
)

type settingsStore struct {
	mu     sync.Mutex
	root   string
	path   string
	loaded bool
	data   map[string]json.RawMessage
	// meta tracks, per section name, when it was last written -- either by
	// a local Save() (stamped "now") or by applyRemote (stamped with the
	// remote's own timestamp, so pulling a value doesn't make this side
	// look newer than the side it was just pulled from). Used by desktop's
	// settings-sync feature to decide which side of a general<->typst<->
	// editor<->lsp section is newer.
	meta map[string]time.Time
}

func newSettingsStore(root string) *settingsStore {
	return &settingsStore{
		root: root,
		path: filepath.Join(root, settingsFileName),
	}
}

// load unmarshals the persisted section named name into model. The
// returned json.RawMessage is the exact bytes that were persisted (nil if
// nothing was ever saved under name) -- callers use it to tell "this field
// was saved as its zero value on purpose" apart from "this field was never
// saved at all", which a zero-value check alone can't distinguish. See
// mergeModel.
func (s *settingsStore) load(name string, model Model) (json.RawMessage, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureLoadedLocked(); err != nil {
		return nil, false, err
	}

	raw, ok := s.data[name]
	if !ok {
		return nil, false, nil
	}

	if err := json.Unmarshal(raw, model); err != nil {
		return nil, false, err
	}
	return raw, true, nil
}

func (s *settingsStore) save(name string, model Model) error {
	return s.saveWithTimestamp(name, model, time.Now())
}

// saveWithTimestamp persists model under name and stamps its "last updated"
// meta entry with ts. A normal local edit (save, via Save()) always stamps
// "now". Applying a value pulled from a remote settings-sync peer
// (applyRemote) stamps the remote's own timestamp instead, so this side
// doesn't look newer than the peer it just copied -- without that, two
// machines syncing back and forth would each think their own copy (just
// received the OTHER side's data) is now the newest and re-push it forever.
func (s *settingsStore) saveWithTimestamp(name string, model Model, ts time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureLoadedLocked(); err != nil {
		return err
	}

	raw, err := json.Marshal(model)
	if err != nil {
		return err
	}
	s.data[name] = raw
	s.meta[name] = ts

	return s.writeLocked()
}

// metaSnapshot returns a copy of the per-section last-updated timestamps.
func (s *settingsStore) metaSnapshot() map[string]time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureLoadedLocked(); err != nil {
		return map[string]time.Time{}
	}

	out := make(map[string]time.Time, len(s.meta))
	for k, v := range s.meta {
		out[k] = v
	}
	return out
}

func (s *settingsStore) ensureLoadedLocked() error {
	if s.loaded {
		return nil
	}

	s.data = make(map[string]json.RawMessage)
	s.meta = make(map[string]time.Time)

	data, err := os.ReadFile(s.path)
	if err == nil {
		if len(bytes.TrimSpace(data)) > 0 {
			if err := json.Unmarshal(data, &s.data); err != nil {
				return err
			}
			delete(s.data, settingsVersionKey)
			if rawMeta, ok := s.data[settingsMetaKey]; ok {
				_ = json.Unmarshal(rawMeta, &s.meta)
				delete(s.data, settingsMetaKey)
			}
		}
		s.loaded = true
		return nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	legacyData, err := loadLegacySettings(filepath.Join(s.root, legacySettingsDB))
	if err != nil {
		log.Printf("migrate legacy settings failed: %v", err)
	}
	for name, raw := range legacyData {
		s.data[name] = raw
	}

	s.loaded = true
	if len(s.data) > 0 {
		return s.writeLocked()
	}
	return nil
}

func (s *settingsStore) writeLocked() error {
	if err := os.MkdirAll(s.root, 0700); err != nil {
		return err
	}

	doc := make(map[string]json.RawMessage, len(s.data)+1)
	for key, val := range s.data {
		doc[key] = val
	}
	version, err := json.Marshal(settingsFileVersion)
	if err != nil {
		return err
	}
	doc[settingsVersionKey] = version

	metaJSON, err := json.Marshal(s.meta)
	if err != nil {
		return err
	}
	doc[settingsMetaKey] = metaJSON

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(s.root, ".settings-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, s.path)
}

func loadLegacySettings(dbFile string) (map[string]json.RawMessage, error) {
	if _, err := os.Stat(dbFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	sections := []struct {
		name  string
		model Model
	}{
		{name: "general", model: &GeneralSettings{}},
		{name: "editor", model: &EditorSettings{}},
		{name: "typst", model: &TypstSettings{}},
		{name: "tpix", model: &TpixSettings{}},
		{name: "acpAgent", model: &AcpAgentSettings{}},
	}

	result := make(map[string]json.RawMessage)
	for _, section := range sections {
		if loadLegacyModel(db, section.name, section.model) {
			raw, err := json.Marshal(section.model)
			if err != nil {
				return nil, err
			}
			result[section.name] = raw
		}
	}

	return result, nil
}

func loadLegacyModel(db *bolt.DB, name string, model Model) bool {
	values, ok := readLegacyValues(db, name, model)
	if !ok {
		return false
	}

	loaded := loadLegacyValuesIndividually(model, values)
	if loadLegacyValuesAsStream(model, values) {
		loaded = true
	}
	return loaded
}

func readLegacyValues(db *bolt.DB, name string, model Model) ([][]byte, bool) {
	t := reflect.ValueOf(model).Elem()
	values := make([][]byte, t.NumField())
	found := false

	err := db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(name))
		if bkt == nil {
			return nil
		}

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.CanSet() {
				continue
			}

			keyName := t.Type().Field(i).Tag.Get("key")
			if keyName == "" {
				continue
			}

			raw := bkt.Get([]byte(keyName))
			if raw == nil {
				continue
			}

			values[i] = append([]byte(nil), raw...)
			found = true
		}
		return nil
	})
	if err != nil {
		log.Printf("read legacy settings bucket %s failed: %v", name, err)
		return nil, false
	}

	return values, found
}

func loadLegacyValuesIndividually(model Model, values [][]byte) bool {
	t := reflect.ValueOf(model).Elem()
	loaded := false

	for i, raw := range values {
		if raw == nil {
			continue
		}

		var val any
		if err := (&utils.BinaryEncoder[any]{}).Decode(raw, &val); err != nil {
			continue
		}

		if err := setFieldValue(t.Field(i), val); err != nil {
			continue
		}
		loaded = true
	}

	return loaded
}

func loadLegacyValuesAsStream(model Model, values [][]byte) bool {
	var stream bytes.Buffer
	for _, raw := range values {
		if raw == nil {
			continue
		}
		stream.Write(raw)
	}
	if stream.Len() == 0 {
		return false
	}

	decoder := gob.NewDecoder(&stream)
	t := reflect.ValueOf(model).Elem()
	loaded := false

	for i, raw := range values {
		if raw == nil {
			continue
		}

		var val any
		if err := decoder.Decode(&val); err != nil {
			return loaded
		}

		if err := setFieldValue(t.Field(i), val); err != nil {
			continue
		}
		loaded = true
	}

	return loaded
}
