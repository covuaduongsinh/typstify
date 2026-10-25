package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/coder/websocket"

	"looz.ws/typstify/service/collab"
	"looz.ws/typstify/service/projects"
)

// Transport for real-time co-editing (service/collab, web/src/lib/collab):
// one WebSocket per open file, carrying binary collab frames.

const (
	collabDiskCheckInterval = 2 * time.Second
	// A compacted state can be several times the file (tombstones).
	collabReadLimit = 8 * collab.MaxFileSize
	// Close code for transient drops (the client fell behind): the client
	// reconnects. 1008 (policy violation) means "do not come back".
	collabStatusRetry websocket.StatusCode = 4000
)

func (s *Server) newCollabHub() *collab.Hub {
	return &collab.Hub{
		Base: filepath.Join(s.storageDir, "collab"),
		Save: func(abs string, data []byte) ([]byte, error) {
			// Same rules as PUT /api/workspace/file.
			if strings.HasSuffix(abs, ".typ") {
				data = repairTypstContent(data)
				s.SetActiveFile(abs, 0)
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return nil, err
			}
			return data, writeFileAtomic(abs, data)
		},
	}
}

// runCollabDiskWatch reports outside edits (agent, Dropbox, desktop) of
// co-edited files to their rooms' leaders.
func (s *Server) runCollabDiskWatch(ctx context.Context) {
	t := time.NewTicker(collabDiskCheckInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.collab.CheckDisk()
		}
	}
}

// collabRole is the caller's current role for an open room: none once the
// session ended or the server switched to another project.
func (s *Server) collabRole(r *http.Request, projectDir string) (projects.Role, bool) {
	if s.appSrv.CurrentProjectDir() != projectDir {
		return "", false
	}
	if s.auth.enabled() {
		if _, ok := s.auth.getSession(r); !ok {
			if _, ok := s.auth.authenticatedUser(r); !ok {
				return "", false
			}
		}
	}
	_, role, registered := s.currentProjectRole(r)
	if !registered {
		// Unregistered projects keep the old "any signed-in user edits".
		return projects.RoleEditor, true
	}
	return role, role.AtLeast(projects.RoleViewer)
}

func sameDir(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// handleCollabWS joins the co-editing room of ?path= in the open project.
func (s *Server) handleCollabWS(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	// The client names the project it has open. After a restart or a
	// project switch the server's current project may be another one,
	// where the same relative path is a different file.
	if want := r.URL.Query().Get("project"); !sameDir(want, root) {
		writeError(w, http.StatusConflict, "Máy chủ đang mở một dự án khác")
		return
	}
	abs, err := resolveInRoot(root, r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rel = filepath.ToSlash(rel)

	user, _ := s.requestUser(r)
	name := s.auth.displayNameOf(user)
	peer := collab.NewPeer(user, name, func() (projects.Role, bool) { return s.collabRole(r, root) })

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: s.wsOriginPatterns(r)})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(collabReadLimit)

	room, replay, err := s.collab.Join(root, rel, abs, peer)
	if err != nil {
		msg := "không mở được phòng cộng tác"
		if errors.Is(err, collab.ErrTooLarge) {
			msg = err.Error()
		}
		_ = conn.Close(websocket.StatusPolicyViolation, msg)
		return
	}
	defer s.collab.Leave(room, peer)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	for _, msg := range replay {
		if err := conn.Write(ctx, websocket.MessageBinary, msg); err != nil {
			return
		}
	}
	go func() {
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			case <-peer.Done():
				code := collabStatusRetry
				if peer.Fatal() {
					code = websocket.StatusPolicyViolation
				}
				_ = conn.Close(code, peer.Reason())
				return
			case msg := <-peer.Outbox():
				if err := conn.Write(ctx, websocket.MessageBinary, msg); err != nil {
					return
				}
			}
		}
	}()

	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if typ == websocket.MessageBinary {
			room.Handle(peer, data)
		}
	}
}
