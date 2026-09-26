package agent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"looz.ws/typstify/version"
)

const (
	ServerName = "Typstify"
)

type McpServer struct {
	mcpServer  *mcpsdk.Server
	httpServer *http.Server
	serverAddr string
	port       int
	started    atomic.Bool
	running    atomic.Bool
}

type McpToolProvider interface {
	RegisterTools(s *McpServer) error
}

type McpResourceProvider interface {
	RegisterResources(s *McpServer) error
}

func NewMcpServer(port int) *McpServer {
	return &McpServer{
		port: port,
		mcpServer: mcpsdk.NewServer(
			&mcpsdk.Implementation{
				Name:    "typstify-mcp-server",
				Title:   ServerName,
				Version: version.BinVersion,
			},
			&mcpsdk.ServerOptions{
				Instructions: `Typstify MCP Server – Chess Document Editor Environment.

CRITICAL RULES for Typst chess documents:
1. NEVER manually declare #let definitions for any chessbook function (e.g. lesson-header, chess-quote, instructor-note, practice-question, puzzle-card, game-header, eco-header, concept-box, teaching-diagram, column-diagram, opening-diagram-box, difficulty-stars, turn-indicator, chess-board, turn-box, nag, note-num, upside-down-solutions, render-puzzle-solutions).
2. ALL these functions are provided by the @local/chessbook:0.1.0 package. If a document uses any of them, ensure #import "@local/chessbook:0.1.0": * is at the top.
3. If you see "unknown variable" errors for any of these functions, ADD THE IMPORT LINE — do NOT create inline #let definitions.
4. In Typst content blocks [...], the # character starts a code expression. Literal # must be escaped as \# (e.g. table.header([*\#*], [*White*], [*Black*])).
5. puzzle-card accepts both turn: and to-move: parameters.
6. chess-quote, instructor-note, concept-box accept content blocks: #chess-quote(author: "Name")[Content here].
7. Show preview result before editing the typst files if possible.`,
				Logger:       slog.Default(),
				PageSize:     mcpsdk.DefaultPageSize,
				GetSessionID: func() string {
					return uuid.NewString()
				},
				Capabilities: &mcpsdk.ServerCapabilities{
					Logging:   &mcpsdk.LoggingCapabilities{},
					Prompts:   &mcpsdk.PromptCapabilities{},
					Resources: &mcpsdk.ResourceCapabilities{},
					Tools:     &mcpsdk.ToolCapabilities{},
				},
			},
		),
	}
}

func (s *McpServer) Run() error {
	if s.mcpServer == nil {
		return errors.New("mcp server is not initialized")
	}

	sseHandler := mcpsdk.NewStreamableHTTPHandler(func(request *http.Request) *mcpsdk.Server {
		return s.mcpServer
	}, &mcpsdk.StreamableHTTPOptions{})

	serveMux := http.NewServeMux()
	serveMux.Handle("/", recoveryMiddleware(sseHandler))

	if s.started.CompareAndSwap(false, true) {
		s.serverAddr = "127.0.0.1"

		address := "127.0.0.1:0" // use random port.
		if s.port > 0 {
			address = fmt.Sprintf("127.0.0.1:%d", s.port)
		}
		listener, err := net.Listen("tcp4", address)
		if err != nil {
			// e.g. the static MCP port is already taken. Run without the
			// built-in tools rather than crashing the whole app/server.
			s.started.Store(false)
			return fmt.Errorf("mcp server: listen on %s: %w", address, err)
		}

		if s.port <= 0 {
			netAddr := listener.Addr().(*net.TCPAddr)
			s.port = netAddr.Port
		}

		log.Printf("mcp server is running at %s:%d", s.serverAddr, s.port)

		s.httpServer = &http.Server{
			Addr:         address,
			WriteTimeout: time.Second * 10,
			Handler:      serveMux,
		}

		go func() {
			if err := s.httpServer.Serve(listener); err != nil {
				s.started.Store(false)
				listener.Close()
				log.Println("mcp server down: ", err)
			}

			log.Println("mcp server is down.")
		}()
	}

	return nil
}

func (s *McpServer) Addr() (string, int) {
	return s.serverAddr, s.port
}

func (s *McpServer) Shutdown(ctx context.Context) error {
	if s.httpServer == nil { // never started (e.g. Run failed to listen)
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *McpServer) RegisterToolProvider(provider McpToolProvider) {
	provider.RegisterTools(s)
}

func (s *McpServer) RegisterResourceProvider(provider McpResourceProvider) {
	provider.RegisterResources(s)
}

func (s *McpServer) AddResource(resource *mcpsdk.Resource, handler mcpsdk.ResourceHandler) error {
	if s.mcpServer == nil {
		return errors.New("mcp server is not initialized")
	}

	s.mcpServer.AddResource(resource, handler)
	return nil
}

func (s *McpServer) RemoveTools(names ...string) error {
	if s.mcpServer == nil {
		return errors.New("mcp server is not initialized")
	}

	s.mcpServer.RemoveTools(names...)
	return nil
}

func (s *McpServer) RemoveResources(uris ...string) error {
	if s.mcpServer == nil {
		return errors.New("mcp server is not initialized")
	}

	s.mcpServer.RemoveResources(uris...)
	return nil
}

func AddMcpTool[In, Out any](s *McpServer, tool *mcpsdk.Tool, handler mcpsdk.ToolHandlerFor[In, Out]) error {
	if s.mcpServer == nil {
		return errors.New("mcp server is not initialized")
	}

	mcpsdk.AddTool(s.mcpServer, tool, handler)
	return nil
}

func recoveryMiddleware(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Println("recovering mcp server, error:", err)

				w.WriteHeader(http.StatusInternalServerError)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write([]byte("Server Panics"))
				return
			}
		}()

		h.ServeHTTP(w, r)
	}
}
