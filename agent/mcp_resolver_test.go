package agent

import (
	"testing"

	"github.com/coder/acp-go-sdk"
)

func httpMcp(url string) acp.McpServer {
	return acp.McpServer{Http: &acp.McpServerHttpInline{Name: ServerName, Type: "http", Url: url, Headers: []acp.HttpHeader{}}}
}

func TestSessionManagerMcpPerWorkingDirectory(t *testing.T) {
	conn := &AgentConn{}
	conn.AgentCapabilities.McpCapabilities.Http = true

	sm := NewSessionManager([]acp.McpServer{httpMcp("http://fixed")})
	if got := sm.serversFor(conn, "/x"); len(got) != 1 || got[0].Http.Url != "http://fixed" {
		t.Fatalf("without a resolver: %+v", got)
	}

	sm.SetMcpResolver(func(cwd string) []acp.McpServer {
		return []acp.McpServer{httpMcp("http://" + cwd[1:])}
	})
	for _, cwd := range []string{"/x", "/y"} {
		got := sm.serversFor(conn, cwd)
		if len(got) != 1 || got[0].Http.Url != "http://"+cwd[1:] {
			t.Fatalf("%s: %+v", cwd, got)
		}
	}

	// An agent without HTTP MCP still gets nothing it cannot use.
	conn.AgentCapabilities.McpCapabilities.Http = false
	if got := sm.serversFor(conn, "/x"); len(got) != 0 {
		t.Fatalf("http MCP offered to an agent without it: %+v", got)
	}
}
