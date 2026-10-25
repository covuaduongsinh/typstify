package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"looz.ws/typstify/service/collab"
)

func dialCollab(t *testing.T, srv *httptest.Server, c *apiClient, project, path string) *websocket.Conn {
	t.Helper()
	h := http.Header{}
	h.Add("Cookie", c.cookie.String())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/collab?path="+path+"&project="+url.QueryEscape(project), &websocket.DialOptions{HTTPHeader: h})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	conn.SetReadLimit(collab.MaxFileSize + 1<<20)
	return conn
}

// readUntilSynced returns the hello message and the replayed updates.
func readUntilSynced(t *testing.T, conn *websocket.Conn) (map[string]any, int) {
	t.Helper()
	var hello map[string]any
	updates := 0
	for {
		msg := readFrame(t, conn, 5*time.Second)
		if msg == nil {
			t.Fatal("no synced frame")
		}
		switch msg[0] {
		case collab.MsgControl:
			var v map[string]any
			_ = json.Unmarshal(msg[2:], &v)
			if v["type"] == "hello" {
				hello = v
			}
		case collab.MsgUpdate:
			updates++
		case collab.MsgSynced:
			return hello, updates
		}
	}
}

// readFrame returns the next non-control frame, or nil on timeout/close.
func readFrame(t *testing.T, conn *websocket.Conn, wait time.Duration) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	_, data, err := conn.Read(ctx)
	if err != nil {
		return nil
	}
	return data
}

func nextData(t *testing.T, conn *websocket.Conn, wait time.Duration) []byte {
	t.Helper()
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		msg := readFrame(t, conn, time.Until(deadline))
		if msg == nil {
			return nil
		}
		if msg[0] != collab.MsgControl {
			return msg
		}
	}
	return nil
}

func TestCollabWebSocketRoles(t *testing.T) {
	root, err := os.MkdirTemp("", "typstify-collab-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	s := New(testService(), Options{ProjectRoot: root, StorageDir: tempStorage(t)})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	owner := &apiClient{t: t, h: s.Handler()}
	mustStatus(t, owner.do("POST", "/api/auth/register", `{"username":"owner","password":"pw1234"}`), 200, "register owner")
	ownerCookie := owner.cookie
	mustStatus(t, owner.do("POST", "/api/auth/register", `{"username":"guest","password":"pw1234"}`), 200, "register guest")
	owner.cookie = ownerCookie

	w := owner.do("POST", "/api/projects", `{"name":"Cộng tác"}`)
	mustStatus(t, w, 200, "create project")
	var created projectView
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	w = owner.do("POST", "/api/projects/"+created.ID+"/shares", `{"role":"viewer"}`)
	var share struct{ Token string }
	_ = json.Unmarshal(w.Body.Bytes(), &share)

	guest := &apiClient{t: t, h: s.Handler()}
	mustStatus(t, guest.do("POST", "/api/auth/login", `{"username":"guest","password":"pw1234"}`), 200, "guest login")
	mustStatus(t, guest.do("POST", "/api/projects/join", `{"token":"`+share.Token+`"}`), 200, "join")
	mustStatus(t, owner.do("POST", "/api/projects/"+created.ID+"/open", ""), 200, "open")
	mustStatus(t, owner.do("PUT", "/api/workspace/file?path=main.typ", "= Bắt đầu"), 200, "write file")

	var cur struct{ Path string }
	_ = json.Unmarshal(owner.do("GET", "/api/workspace/current", "").Body.Bytes(), &cur)
	oc := dialCollab(t, srv, owner, cur.Path, "main.typ")
	defer oc.CloseNow()
	hello, seeded := readUntilSynced(t, oc)
	if hello["leader"] != true || hello["role"] != "owner" || seeded != 1 {
		t.Fatalf("owner hello %v, %d updates", hello, seeded)
	}

	gc := dialCollab(t, srv, guest, cur.Path, "main.typ")
	defer gc.CloseNow()
	hello, seeded = readUntilSynced(t, gc)
	if hello["leader"] != false || hello["role"] != "viewer" || seeded != 1 {
		t.Fatalf("guest hello %v, %d updates", hello, seeded)
	}

	// While the room is open, a plain PUT would overwrite the shared text.
	mustStatus(t, owner.do("PUT", "/api/workspace/file?path=main.typ", "stale copy"), 409, "PUT on a co-edited file")

	ctx := context.Background()
	update := collab.Frame(collab.MsgUpdate, collab.ChanText, []byte{0, 0})
	if err := oc.Write(ctx, websocket.MessageBinary, update); err != nil {
		t.Fatal(err)
	}
	if got := nextData(t, gc, 3*time.Second); got == nil || got[0] != collab.MsgUpdate {
		t.Fatalf("guest should receive the owner's update, got %v", got)
	}

	// The viewer's text and comment updates go nowhere.
	_ = gc.Write(ctx, websocket.MessageBinary, collab.Frame(collab.MsgUpdate, collab.ChanText, []byte{0, 0}))
	_ = gc.Write(ctx, websocket.MessageBinary, collab.Frame(collab.MsgUpdate, collab.ChanComments, []byte{0, 0}))
	if got := nextData(t, oc, 500*time.Millisecond); got != nil {
		t.Fatalf("viewer update leaked: %v", got)
	}

	// Removing the member cuts the open socket at its next message.
	mustStatus(t, owner.do("DELETE", "/api/projects/"+created.ID+"/members/guest", ""), 200, "remove guest")
	_ = gc.Write(ctx, websocket.MessageBinary, collab.Frame(collab.MsgAwareness, 0, []byte{1}))
	deadline := time.Now().Add(3 * time.Second)
	closed := false
	for time.Now().Before(deadline) {
		rctx, cancel := context.WithTimeout(ctx, time.Until(deadline))
		_, _, err := gc.Read(rctx)
		cancel()
		if err != nil {
			closed = websocket.CloseStatus(err) == websocket.StatusPolicyViolation
			break
		}
	}
	if !closed {
		t.Fatal("removed member's socket should be closed")
	}

	// A client that still has another project open is refused: the same
	// relative path would be a different file here.
	h := http.Header{}
	h.Add("Cookie", owner.cookie.String())
	wrong := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/collab?path=main.typ&project=" + url.QueryEscape(root+"/elsewhere")
	if _, _, err := websocket.Dial(ctx, wrong, &websocket.DialOptions{HTTPHeader: h}); err == nil {
		t.Fatal("dial for another project must fail")
	}

	// Anonymous clients cannot join at all.
	if _, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/collab?path=main.typ", nil); err == nil {
		t.Fatal("anonymous dial must fail")
	}
}
