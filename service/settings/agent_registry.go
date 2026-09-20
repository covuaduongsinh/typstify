package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const registryURL = "https://cdn.agentclientprotocol.com/registry/v1/latest/registry.json"

type AgentRegistry struct {
	Version string       `json:"version"`
	Agents  []AgentEntry `json:"agents"`
}

type AgentEntry struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Version      string             `json:"version"`
	Description  string             `json:"description"`
	Repository   string             `json:"repository,omitempty"`
	Website      string             `json:"website,omitempty"`
	Authors      []string           `json:"authors"`
	License      string             `json:"license"`
	Icon         string             `json:"icon"`
	Distribution AgentDistribution `json:"distribution"`
}

// DistKind returns "npx", "uvx", "binary", or "unknown".
func (e AgentEntry) DistKind() string {
	if e.Distribution.Npx != nil {
		return "npx"
	}
	if e.Distribution.Uvx != nil {
		return "uvx"
	}
	if len(e.Distribution.Binary) > 0 {
		return "binary"
	}
	return "unknown"
}

// AgentDistribution is a one-of: exactly one of Npx, Uvx, or Binary is set.
type AgentDistribution struct {
	Npx    *PackageDistro            `json:"npx,omitempty"`
	Uvx    *PackageDistro            `json:"uvx,omitempty"`
	Binary map[string]*BinaryDistro  `json:"binary,omitempty"`
}

// PackageDistro holds the fields for npx and uvx distributions.
type PackageDistro struct {
	Package string            `json:"package"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// BinaryDistro holds a single platform binary distribution.
type BinaryDistro struct {
	Archive string            `json:"archive"`
	Cmd     string            `json:"cmd"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

var (
	cachedRegistry *AgentRegistry
	registryMu     sync.Mutex
)

// FetchAgentRegistry fetches the ACP agent registry from the CDN.
// Results are cached in memory after the first successful fetch.
func FetchAgentRegistry(ctx context.Context) (*AgentRegistry, error) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if cachedRegistry != nil {
		return cachedRegistry, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, registryURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var reg AgentRegistry
	if err := json.NewDecoder(resp.Body).Decode(&reg); err != nil {
		return nil, err
	}

	cachedRegistry = &reg
	return cachedRegistry, nil
}

// ResolvedAgentCommand is what an AgentEntry resolves to for the current
// platform: the (Cmd, Args, Env) to store in AcpAgentSettings.
type ResolvedAgentCommand struct {
	Cmd  string
	Args string
	Env  string
}

// ResolveAgentCommand converts a registry entry into a concrete command to
// run, for the current OS/arch. Shared by the desktop UI
// (ui/settings/agent.go) and the web backend (server/agent_api.go) so both
// pick agents identically -- see docs/plans (AI subscription agents plan).
func ResolveAgentCommand(entry *AgentEntry) ResolvedAgentCommand {
	switch entry.DistKind() {
	case "npx":
		args := "-y " + entry.Distribution.Npx.Package
		if len(entry.Distribution.Npx.Args) > 0 {
			args += " " + strings.Join(entry.Distribution.Npx.Args, " ")
		}
		return ResolvedAgentCommand{Cmd: "npx", Args: args}

	case "uvx":
		args := entry.Distribution.Uvx.Package
		if len(entry.Distribution.Uvx.Args) > 0 {
			args += " " + strings.Join(entry.Distribution.Uvx.Args, " ")
		}
		return ResolvedAgentCommand{Cmd: "uvx", Args: args}

	default:
		platform := runtime.GOOS + "-" + runtime.GOARCH
		if runtime.GOARCH == "amd64" {
			platform = runtime.GOOS + "-x86_64"
		}

		var bin *BinaryDistro
		if b, ok := entry.Distribution.Binary[platform]; ok {
			bin = b
		} else {
			for _, b := range entry.Distribution.Binary {
				bin = b
				break
			}
		}
		if bin == nil {
			return ResolvedAgentCommand{}
		}

		// Registry binary Cmd values are given as "./name" (relative to
		// wherever the archive was extracted); Typstify's own
		// utils.LookupExecutable already searches the app dir, its bin/
		// subdir, cwd, and PATH for a bare name, so strip the "./"/".\"
		// prefix rather than pass it through literally.
		result := ResolvedAgentCommand{
			Cmd:  strings.TrimPrefix(strings.TrimPrefix(bin.Cmd, "./"), ".\\"),
			Args: strings.Join(bin.Args, " "),
		}

		if entry.ID == "antigravity-acp" {
			// agy_acp_server has been observed to hang without a writable
			// TEMP/TMP; point it at a dedicated subdirectory of the OS temp
			// dir rather than leaving it unset. Verified live 2026-09-18.
			dir := filepath.Join(os.TempDir(), "typstify-agy")
			result.Env = "TEMP=" + dir + " TMP=" + dir
		}
		return result
	}
}

// LookupAgent finds a registry entry by ID. Returns nil if not found.
func LookupAgent(id string) *AgentEntry {
	registryMu.Lock()
	defer registryMu.Unlock()

	if cachedRegistry == nil {
		return nil
	}
	for i := range cachedRegistry.Agents {
		if cachedRegistry.Agents[i].ID == id {
			return &cachedRegistry.Agents[i]
		}
	}
	return nil
}
