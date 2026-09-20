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

// TestModelSave_PreservesIntentionallyEmptyField reproduces a real bug
// (found live 2026-09-19 debugging why Google Antigravity always launched
// with Claude Code's npx args tacked on): mergeModel used to treat "field
// equals its zero value" as "field was never saved", so an
// intentionally-empty saved field (Args="" for a binary agent that takes
// no arguments) got silently reset back to the section's default
// (Args="-y @agentclientprotocol/claude-agent-acp@...", Claude Code's own
// default args) on every subsequent load -- not just the first one.
func TestModelSave_PreservesIntentionallyEmptyField(t *testing.T) {
	root := t.TempDir()
	db := newSettings(root, nil)

	agent := db.AcpAgent()
	agent.AgentID = "antigravity-acp"
	agent.AgentName = "Google Antigravity"
	agent.Cmd = "agy_acp_server.exe"
	agent.Args = "" // intentionally empty: this agent takes no args
	if err := agent.Save(); err != nil {
		t.Fatal(err)
	}

	// Reload through a fresh Settings instance, the same way a new HTTP
	// request or a restarted session manager would.
	reloaded := newSettings(root, nil)
	t.Cleanup(reloaded.Close)

	got := reloaded.AcpAgent()
	if got.Args != "" {
		t.Fatalf("Args = %q, want empty string (must not fall back to the default agent's args)", got.Args)
	}
	if got.Cmd != "agy_acp_server.exe" {
		t.Fatalf("Cmd = %q, want %q", got.Cmd, "agy_acp_server.exe")
	}

	// Loading a THIRD time (as would happen on every subsequent settings
	// read) must keep giving the same answer -- the original bug only
	// showed up from the second load onward.
	again := newSettings(root, nil)
	t.Cleanup(again.Close)
	if got := again.AcpAgent().Args; got != "" {
		t.Fatalf("Args on third load = %q, want empty string", got)
	}
}

// TestModelSave_DefaultsFillOnlyTrulyMissingFields is the complementary
// case: a field that was never part of the persisted section (e.g. added
// to the struct after the settings file was last written) must still fall
// back to the default -- the fix for the bug above must not disable
// migration of new fields for existing settings files.
func TestModelSave_DefaultsFillOnlyTrulyMissingFields(t *testing.T) {
	root := t.TempDir()

	// Simulate an old settings.json that predates the "env" field: write
	// the acpAgent section by hand without it.
	doc := map[string]json.RawMessage{
		"acpAgent": json.RawMessage(`{"agentId":"claude-acp","agentName":"Claude Code","cmd":"npx","args":"-y @agentclientprotocol/claude-agent-acp@0.35.0"}`),
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "settings.json"), data, 0600); err != nil {
		t.Fatal(err)
	}

	db := newSettings(root, nil)
	t.Cleanup(db.Close)

	agent := db.AcpAgent()
	if agent.UseStaticMcpPort != defaultAcpAgentSettings.UseStaticMcpPort {
		t.Fatalf("UseStaticMcpPort = %d, want default %d (field absent from the persisted JSON must still merge in)",
			agent.UseStaticMcpPort, defaultAcpAgentSettings.UseStaticMcpPort)
	}
	if agent.Args != "-y @agentclientprotocol/claude-agent-acp@0.35.0" {
		t.Fatalf("Args = %q, want the persisted value unchanged", agent.Args)
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
