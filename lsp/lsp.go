package lsp

import (
	"context"
	"log"
	"runtime"
	"strings"
	"sync"

	"looz.ws/typstify/service/settings"
	"looz.ws/typstify/utils"
)

var (
	// The default client: the desktop app's (and a single-project
	// server's) workspace. Other workspaces get their own client from
	// NewClient. defaultMu guards it: HTTP handlers call GetLspClient
	// concurrently.
	defaultMu  sync.Mutex
	lspClient  *Client
	version    string
	cmdBuilder utils.CmdBuilder
)

// use init function to setup PATH.
func SetupCmdBuilder(externalExe string) {
	exists, isDir := utils.CheckFileExists(externalExe)
	if exists && !isDir {
		cmdBuilder.Path = externalExe
	} else {
		var exeName = "tinymist"
		if runtime.GOOS == "windows" {
			exeName = "tinymist.exe"
		}
		cmdBuilder.Path = exeName
	}

	cmdBuilder.DefaultArgs = []string{}
	cmdBuilder.Check()
}

// NewClient starts a tinymist instance for workspace. Each runs as its own
// process over stdio, so several workspaces can be served at once; the
// caller owns the client and must Stop it.
func NewClient(workspace string, setting *settings.Settings) (*Client, error) {
	c := newClient(newServer(workspace))
	if err := c.Start(context.Background(), setting); err != nil {
		return nil, err
	}
	return c, nil
}

// GetLspClient returns the default client, switching it to workspace (and
// stopping the previous one) when needed.
func GetLspClient(workspace string, setting *settings.Settings) *Client {
	defaultMu.Lock()
	defer defaultMu.Unlock()

	if lspClient != nil && lspClient.server.Workspace() == workspace {
		return lspClient
	}

	if lspClient != nil {
		lspClient.Stop()
		lspClient = nil
	}

	c, err := NewClient(workspace, setting)
	if err != nil {
		log.Println("LSP client setup failed: ", err)
		return nil
	}
	lspClient = c

	log.Println("LSP client is setup")
	return lspClient
}

func StopLsp() {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if lspClient != nil {
		lspClient.Stop()
	}
}

// Version returns the LSP server version.
func Version() string {
	if version == "" {
		cmd := cmdBuilder.Build(context.Background(), "-V")
		out, _ := cmd.Output()
		version = strings.TrimSpace(string(out))
	}

	return version
}
