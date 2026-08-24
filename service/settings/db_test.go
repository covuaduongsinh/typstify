package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestModelSave(t *testing.T) {
	root := t.TempDir()
	db := newSettings(root, nil)
	general := db.General()

	t.Log("general: ", general)

	general.Language = "cn/zh"

	err := general.Save()
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	db.Close()

	data, err := os.ReadFile(filepath.Join(root, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}

	var persisted GeneralSettings
	if err := json.Unmarshal(doc["general"], &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Language != "cn/zh" {
		t.Fatalf("Language = %q, want %q", persisted.Language, "cn/zh")
	}

	reloaded := newSettings(root, nil)
	if got := reloaded.General().Language; got != "cn/zh" {
		t.Fatalf("reloaded Language = %q, want %q", got, "cn/zh")
	}

}

func TestModelGetDefault(t *testing.T) {
	db := newSettings(t.TempDir(), nil)
	general := db.General()
	typst := db.Typst()

	t.Log("general: ", general)
	t.Log("typst: ", typst)
	t.Log("editor: ", db.Editor())

	t.Cleanup(func() {
		db.Close()
	})

}
