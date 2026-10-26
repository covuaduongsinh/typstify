package service

import (
	"context"
	"errors"
	"io"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"looz.ws/typstify/agent"
	"looz.ws/typstify/lsp"
	"looz.ws/typstify/service/settings"
)

// ErrTooManyProjects is returned by Runtimes.Acquire when the limit of
// projects open at once is reached and every open one is in use.
var ErrTooManyProjects = errors.New("too many projects open at once")

// DefaultMaxOpenProjects bounds how many projects keep a tinymist (about
// 150-300 MB each) at once (docs/plans/plan_phase3b_per_user_projects.md).
const DefaultMaxOpenProjects = 3

// runtimeIdleTimeout: an unused runtime is closed after this long.
const runtimeIdleTimeout = 10 * time.Minute

// ProjectRuntime holds what one open project needs on the server: its own
// tinymist (LSP), tinymist preview and MCP server (the AI agent's built-in
// tools, bound to this project). All start on first use, so a project only
// browsed or edited through the REST API costs nothing.
type ProjectRuntime struct {
	Dir string
	st  *settings.Settings
	ws  *WorkspaceService
	log func() io.Writer

	mu      sync.Mutex
	client  *lsp.Client
	preview *lsp.PreviewService
	mcp     *agent.McpServer
	closed  bool

	// Guarded by Runtimes.mu.
	refs     int
	lastUsed time.Time
}

// LSP returns the project's tinymist client, starting it (again, after a
// crash) when needed; nil once the runtime is closed or when tinymist
// cannot start.
func (rt *ProjectRuntime) LSP() *lsp.Client {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.lspLocked()
}

func (rt *ProjectRuntime) lspLocked() *lsp.Client {
	if rt.closed {
		return nil
	}
	if rt.client != nil && rt.client.Running() {
		return rt.client
	}
	if rt.client != nil {
		rt.client.Stop()
		// The preview task lived in the dead tinymist.
		rt.client, rt.preview = nil, nil
	}
	c, err := lsp.NewClient(rt.Dir, rt.st)
	if err != nil {
		log.Printf("tinymist for %s failed to start: %v", rt.Dir, err)
		return nil
	}
	if rt.log != nil {
		c.SetServreLogStreamer(rt.log())
	} else {
		c.SetServreLogStreamer(io.Discard)
	}
	rt.client = c
	return c
}

// Preview returns the project's tinymist preview, nil until
// RestartPreview has started one.
func (rt *ProjectRuntime) Preview() *lsp.PreviewService {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.closed {
		return nil
	}
	return rt.preview
}

// RestartPreview (re)starts the tinymist preview on entryFile (absolute,
// or "" for tinymist's default) in the background; onFinish runs when it
// is up.
func (rt *ProjectRuntime) RestartPreview(entryFile string, onFinish func()) {
	rt.mu.Lock()
	client := rt.lspLocked()
	if client == nil {
		rt.mu.Unlock()
		return
	}
	if rt.preview == nil {
		rt.preview = lsp.NewPreviwService(client)
	}
	p := rt.preview
	rt.mu.Unlock()

	mode := lsp.PreviewMode(rt.ws.loadWorkspaceSettingsForPath(rt.Dir).PreviewMode)
	if mode == "" {
		mode = lsp.DocumentPreviewMode
	}
	go p.Start(context.Background(), lsp.PreviewOptions{
		Mode:          mode,
		ProjectRoot:   rt.Dir,
		EntryFile:     entryFile,
		InvertColor:   "never",
		PartialRender: rt.st.Lsp().EnablePartialRenderPreview,
	}, onFinish)
}

// mcpServer returns the runtime's MCP server, made by build on first use;
// nil once closed or when build fails (the agent then runs without the
// built-in tools).
func (rt *ProjectRuntime) mcpServer(build func() (*agent.McpServer, error)) *agent.McpServer {
	rt.mu.Lock()
	if rt.closed {
		rt.mu.Unlock()
		return nil
	}
	if rt.mcp != nil {
		m := rt.mcp
		rt.mu.Unlock()
		return m
	}
	rt.mu.Unlock()

	// Built unlocked: build starts tinymist through LSP(), which locks.
	m, err := build()
	if err != nil {
		log.Printf("MCP tools for %s: %v", rt.Dir, err)
		return nil
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.closed || rt.mcp != nil {
		// Closed meanwhile, or another caller won the race.
		_ = m.Shutdown(context.Background())
		if rt.closed {
			return nil
		}
		return rt.mcp
	}
	rt.mcp = m
	return m
}

func (rt *ProjectRuntime) close() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.closed {
		return
	}
	rt.closed = true
	if rt.mcp != nil {
		_ = rt.mcp.Shutdown(context.Background())
		rt.mcp = nil
	}
	if rt.preview != nil {
		_ = rt.preview.Destroy(context.Background())
		rt.preview = nil
	}
	if rt.client != nil {
		rt.client.Stop()
		rt.client = nil
	}
}

// Runtimes keeps one ProjectRuntime per open project, at most limit() of
// them: a new project evicts the least recently used idle one, and fails
// with ErrTooManyProjects when all are in use.
type Runtimes struct {
	st    *settings.Settings
	ws    *WorkspaceService
	log   func() io.Writer
	limit func() int
	now   func() time.Time

	mu   sync.Mutex
	open map[string]*ProjectRuntime
}

func newRuntimes(st *settings.Settings, ws *WorkspaceService, logSink func() io.Writer, limit func() int) *Runtimes {
	return &Runtimes{
		st:    st,
		ws:    ws,
		log:   logSink,
		limit: limit,
		now:   time.Now,
		open:  map[string]*ProjectRuntime{},
	}
}

func runtimeKey(dir string) string {
	k := filepath.Clean(dir)
	if runtime.GOOS == "windows" {
		k = strings.ToLower(k)
	}
	return k
}

func (r *Runtimes) max() int {
	if r.limit == nil {
		return DefaultMaxOpenProjects
	}
	if n := r.limit(); n > 0 {
		return n
	}
	return DefaultMaxOpenProjects
}

// Acquire returns dir's runtime, opening it if needed, and holds it open
// until release is called (release is safe to call more than once).
func (r *Runtimes) Acquire(dir string) (*ProjectRuntime, func(), error) {
	key := runtimeKey(dir)
	var victim *ProjectRuntime

	r.mu.Lock()
	rt := r.open[key]
	if rt == nil {
		if len(r.open) >= r.max() {
			victim = r.lruIdleLocked()
			if victim == nil {
				r.mu.Unlock()
				return nil, nil, ErrTooManyProjects
			}
			delete(r.open, runtimeKey(victim.Dir))
		}
		rt = &ProjectRuntime{Dir: filepath.Clean(dir), st: r.st, ws: r.ws, log: r.log}
		r.open[key] = rt
	}
	rt.refs++
	rt.lastUsed = r.now()
	r.mu.Unlock()

	if victim != nil {
		victim.close()
	}

	var once sync.Once
	release := func() {
		once.Do(func() {
			r.mu.Lock()
			rt.refs--
			rt.lastUsed = r.now()
			r.mu.Unlock()
		})
	}
	return rt, release, nil
}

// Touch marks dir's runtime, if open, as just used.
func (r *Runtimes) Touch(dir string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rt := r.open[runtimeKey(dir)]; rt != nil {
		rt.lastUsed = r.now()
	}
}

// Get returns dir's runtime if it is open, without holding it.
func (r *Runtimes) Get(dir string) *ProjectRuntime {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.open[runtimeKey(dir)]
}

// Open lists the directories of the open runtimes.
func (r *Runtimes) Open() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	dirs := make([]string, 0, len(r.open))
	for _, rt := range r.open {
		dirs = append(dirs, rt.Dir)
	}
	return dirs
}

func (r *Runtimes) lruIdleLocked() *ProjectRuntime {
	var oldest *ProjectRuntime
	for _, rt := range r.open {
		if rt.refs > 0 {
			continue
		}
		if oldest == nil || rt.lastUsed.Before(oldest.lastUsed) {
			oldest = rt
		}
	}
	return oldest
}

// Sweep closes the runtimes nobody holds that were last used more than
// idle ago, and returns how many it closed.
func (r *Runtimes) Sweep(idle time.Duration) int {
	cutoff := r.now().Add(-idle)
	var closing []*ProjectRuntime

	r.mu.Lock()
	for key, rt := range r.open {
		if rt.refs == 0 && rt.lastUsed.Before(cutoff) {
			closing = append(closing, rt)
			delete(r.open, key)
		}
	}
	r.mu.Unlock()

	for _, rt := range closing {
		rt.close()
	}
	return len(closing)
}

// Run sweeps idle runtimes every minute until ctx is done.
func (r *Runtimes) Run(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.Sweep(runtimeIdleTimeout)
		}
	}
}

// Close closes every runtime.
func (r *Runtimes) Close() {
	r.mu.Lock()
	all := make([]*ProjectRuntime, 0, len(r.open))
	for key, rt := range r.open {
		all = append(all, rt)
		delete(r.open, key)
	}
	r.mu.Unlock()
	for _, rt := range all {
		rt.close()
	}
}
