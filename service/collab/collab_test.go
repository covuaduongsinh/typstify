package collab

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"looz.ws/typstify/service/projects"
)

// References printed by Yjs 13.6.33 itself:
//
//	d = new Y.Doc(); d.clientID = id; d.getText('content').insert(0, text)
//	Y.encodeStateAsUpdate(d)
func TestEncodeInsertTextMatchesYjs(t *testing.T) {
	cases := []struct {
		client uint64
		text   string
		want   string
	}{
		{12345, "abc", "0101b96000040107636f6e74656e740361626300"},
		{4294967295, "Xin chào 👍\n= Đề mục\n", "0101ffffffff0f00040107636f6e74656e741d58696e206368c3a06f20f09f918d0a3d20c490e1bb81206de1bba5630a00"},
		{7, "", "0000"},
	}
	for _, c := range cases {
		got := hex.EncodeToString(EncodeInsertText(c.client, RootText, c.text))
		if got != c.want {
			t.Errorf("client %d %q:\n got %s\nwant %s", c.client, c.text, got, c.want)
		}
	}
}

func TestLogAppendReplaceReload(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.ylog")
	l, _ := openLog(p)
	for _, s := range []string{"a", "bb", "ccc"} {
		if err := l.append([]byte(s)); err != nil {
			t.Fatal(err)
		}
	}
	again, _ := openLog(p)
	if len(again.records) != 3 || string(again.records[2]) != "ccc" || again.bytes != 6 {
		t.Fatalf("reload: %q", again.records)
	}
	if err := again.replace([][]byte{[]byte("snap")}); err != nil {
		t.Fatal(err)
	}
	// A torn trailing record (crash mid-append) is dropped, not fatal.
	f, _ := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644)
	_, _ = f.Write([]byte{10, 'x'})
	f.Close()
	last, _ := openLog(p)
	if len(last.records) != 1 || string(last.records[0]) != "snap" {
		t.Fatalf("after replace: %q", last.records)
	}
}

type testPeer struct {
	*Peer
	role projects.Role
}

func newTestPeer(user string, role projects.Role) *testPeer {
	tp := &testPeer{role: role}
	tp.Peer = NewPeer(user, user, func() (projects.Role, bool) { return tp.role, tp.role != "" })
	return tp
}

// join adds p and queues its replay in its outbox, so drain() sees every
// message the peer receives.
func join(t *testing.T, h *Hub, proj, abs string, p *Peer) *Room {
	t.Helper()
	r, replay, err := h.Join(proj, "main.typ", abs, p)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range replay {
		p.out <- m
	}
	return r
}

// saveMsg is the leader's save: the disk version it has seen, then text.
func saveMsg(seen, text string) []byte {
	return Frame(MsgSave, 0, append([]byte(seen), text...))
}

func drain(p *Peer) [][]byte {
	var out [][]byte
	for {
		select {
		case m := <-p.out:
			out = append(out, m)
		default:
			return out
		}
	}
}

func controls(msgs [][]byte) []map[string]any {
	var out []map[string]any
	for _, m := range msgs {
		if m[0] == MsgControl {
			var v map[string]any
			_ = json.Unmarshal(m[2:], &v)
			out = append(out, v)
		}
	}
	return out
}

func setup(t *testing.T, content string) (*Hub, string, string) {
	t.Helper()
	proj := t.TempDir()
	abs := filepath.Join(proj, "main.typ")
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	h := &Hub{Base: t.TempDir(), Save: func(abs string, data []byte) ([]byte, error) {
		return data, os.WriteFile(abs, data, 0o644)
	}}
	return h, proj, abs
}

func TestRoomSeedsFromDiskAndRelays(t *testing.T) {
	h, proj, abs := setup(t, "Hello")
	ed := newTestPeer("alice", projects.RoleEditor)
	room := join(t, h, proj, abs, ed.Peer)
	msgs := drain(ed.Peer)
	hello := controls(msgs)[0]
	if hello["type"] != "hello" || hello["leader"] != true || hello["epoch"] == "" {
		t.Fatalf("hello = %v", hello)
	}
	// hello, the seed update, synced
	if len(msgs) != 3 || msgs[1][0] != MsgUpdate || msgs[2][0] != MsgSynced {
		t.Fatalf("replay = %d messages", len(msgs))
	}
	// One client, one struct, ending with the text and an empty delete set.
	seed := hex.EncodeToString(msgs[1][2:])
	if seed[:4] != "0101" || !strings.HasSuffix(seed, hex.EncodeToString([]byte("Hello"))+"00") {
		t.Fatalf("seed update = %s", seed)
	}

	viewer := newTestPeer("bob", projects.RoleViewer)
	join(t, h, proj, abs, viewer.Peer)
	drain(viewer.Peer)
	drain(ed.Peer)

	room.Handle(ed.Peer, Frame(MsgUpdate, ChanText, []byte{1, 2, 3}))
	if got := drain(viewer.Peer); len(got) != 1 || got[0][0] != MsgUpdate {
		t.Fatalf("viewer should receive the editor's update, got %v", got)
	}
	if got := drain(ed.Peer); len(got) != 0 {
		t.Fatal("updates are not echoed to their sender")
	}

	// A viewer's text update is dropped; awareness is relayed.
	room.Handle(viewer.Peer, Frame(MsgUpdate, ChanText, []byte{9}))
	room.Handle(viewer.Peer, Frame(MsgUpdate, ChanComments, []byte{9}))
	if got := drain(ed.Peer); len(got) != 0 {
		t.Fatalf("viewer updates must be dropped, got %d", len(got))
	}
	room.Handle(viewer.Peer, Frame(MsgAwareness, 0, []byte{7}))
	if got := drain(ed.Peer); len(got) != 1 || got[0][0] != MsgAwareness {
		t.Fatal("awareness from a viewer is relayed")
	}
	if n := len(room.logs[ChanText].records); n != 2 {
		t.Fatalf("text log has %d records, want seed + 1", n)
	}
}

func TestCommenterMayOnlyComment(t *testing.T) {
	h, proj, abs := setup(t, "x")
	ed := newTestPeer("alice", projects.RoleEditor)
	room := join(t, h, proj, abs, ed.Peer)
	c := newTestPeer("carol", projects.RoleCommenter)
	join(t, h, proj, abs, c.Peer)
	drain(ed.Peer)
	room.Handle(c.Peer, Frame(MsgUpdate, ChanText, []byte{1}))
	room.Handle(c.Peer, Frame(MsgUpdate, ChanComments, []byte{2}))
	got := drain(ed.Peer)
	if len(got) != 1 || got[0][1] != ChanComments {
		t.Fatalf("only the comment update should pass, got %v", got)
	}
}

func TestLeaderHandoverSaveAndExternalChange(t *testing.T) {
	h, proj, abs := setup(t, "v1")
	a := newTestPeer("alice", projects.RoleEditor)
	room := join(t, h, proj, abs, a.Peer)
	v := newTestPeer("vic", projects.RoleViewer)
	join(t, h, proj, abs, v.Peer)
	b := newTestPeer("bob", projects.RoleEditor)
	join(t, h, proj, abs, b.Peer)
	drain(a.Peer)
	drain(b.Peer)

	// Only the leader saves.
	room.Handle(b.Peer, saveMsg(room.meta.DiskHash, "from bob"))
	if data, _ := os.ReadFile(abs); string(data) != "v1" {
		t.Fatal("a non-leader must not write the file")
	}
	room.Handle(a.Peer, saveMsg(room.meta.DiskHash, "v2"))
	if data, _ := os.ReadFile(abs); string(data) != "v2" {
		t.Fatal("leader save not written")
	}

	// The leader leaves: the next editor (bob, not the viewer) leads.
	h.Leave(room, a.Peer)
	got := controls(drain(b.Peer))
	if len(got) == 0 || got[0]["type"] != "leader" {
		t.Fatalf("bob should become leader, got %v", got)
	}

	// An outside edit is reported to the new leader, once.
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(abs, []byte("edited by the agent"), 0o644); err != nil {
		t.Fatal(err)
	}
	h.CheckDisk()
	h.CheckDisk()
	ext := controls(drain(b.Peer))
	if len(ext) != 1 || ext[0]["type"] != "external" || ext[0]["text"] != "edited by the agent" {
		t.Fatalf("external change = %v", ext)
	}
	// Once the leader saves that text, it is no longer "external".
	room.Handle(b.Peer, saveMsg(hashBytes([]byte("edited by the agent")), "edited by the agent"))
	h.CheckDisk()
	if got := controls(drain(b.Peer)); len(got) != 0 {
		t.Fatalf("no further notice expected, got %v", got)
	}
}

func TestRevokedPeerIsKickedAndRoomReopensFromLog(t *testing.T) {
	h, proj, abs := setup(t, "start")
	a := newTestPeer("alice", projects.RoleEditor)
	room := join(t, h, proj, abs, a.Peer)
	room.Handle(a.Peer, Frame(MsgUpdate, ChanText, []byte{4, 5}))
	a.role = ""
	room.Handle(a.Peer, Frame(MsgUpdate, ChanText, []byte{6}))
	select {
	case <-a.Done():
	default:
		t.Fatal("revoked peer must be disconnected")
	}
	h.Leave(room, a.Peer)

	// The room was closed; reopening replays seed + update, same epoch.
	b := newTestPeer("bob", projects.RoleEditor)
	join(t, h, proj, abs, b.Peer)
	msgs := drain(b.Peer)
	updates := 0
	for _, m := range msgs {
		if m[0] == MsgUpdate {
			updates++
		}
	}
	if updates != 2 {
		t.Fatalf("replayed %d updates, want 2", updates)
	}
}

func TestCompactionReplacesLog(t *testing.T) {
	h, proj, abs := setup(t, "x")
	a := newTestPeer("alice", projects.RoleEditor)
	room := join(t, h, proj, abs, a.Peer)
	drain(a.Peer)
	for i := 0; i <= compactRecords; i++ {
		room.Handle(a.Peer, Frame(MsgUpdate, ChanText, []byte{byte(i)}))
	}
	asked := controls(drain(a.Peer))
	if len(asked) != 1 || asked[0]["type"] != "compact" {
		t.Fatalf("expected one compact request, got %v", asked)
	}
	room.Handle(a.Peer, Frame(MsgUpdate, ChanText, []byte{0xee})) // arrives after the request
	room.Handle(a.Peer, Frame(MsgCompact, ChanText, []byte("SNAPSHOT")))
	// The request went out when the log passed compactRecords (seed + 256
	// updates); everything logged after it is kept behind the snapshot.
	recs := room.logs[ChanText].records
	if len(recs) != 3 || string(recs[0]) != "SNAPSHOT" || recs[1][0] != byte(compactRecords%256) || recs[2][0] != 0xee {
		t.Fatalf("compacted log = %q", recs)
	}
}

// Regression: after the leader folded in an outside edit (its save equals
// the disk), a server restart must not report that edit again; doing so
// once made a reconnecting leader delete its offline changes.
func TestFoldedExternalEditSurvivesRestart(t *testing.T) {
	h, proj, abs := setup(t, "v1")
	a := newTestPeer("alice", projects.RoleEditor)
	room := join(t, h, proj, abs, a.Peer)
	drain(a.Peer)

	time.Sleep(10 * time.Millisecond)
	_ = os.WriteFile(abs, []byte("v1 + agent"), 0o644)
	h.CheckDisk()
	ext := controls(drain(a.Peer))
	if len(ext) != 1 || ext[0]["base"] != "v1" || ext[0]["text"] != "v1 + agent" {
		t.Fatalf("external = %v", ext)
	}
	room.Handle(a.Peer, saveMsg(hashBytes([]byte("v1 + agent")), "v1 + agent"))
	h.Leave(room, a.Peer)

	restarted := &Hub{Base: h.Base, Save: h.Save}
	b := newTestPeer("alice", projects.RoleEditor)
	join(t, restarted, proj, abs, b.Peer)
	for _, c := range controls(drain(b.Peer)) {
		if c["type"] == "external" {
			t.Fatalf("stale external after restart: %v", c)
		}
	}
}

// Review finding: an autosave arriving before the disk check must not
// overwrite an outside edit the leader has not seen.
func TestSaveNeverOverwritesUnseenOutsideEdit(t *testing.T) {
	h, proj, abs := setup(t, "v1")
	a := newTestPeer("alice", projects.RoleEditor)
	room := join(t, h, proj, abs, a.Peer)
	drain(a.Peer)
	before := room.meta.DiskHash

	_ = os.WriteFile(abs, []byte("v1 + agent"), 0o644)
	room.Handle(a.Peer, saveMsg(before, "v1 + typing"))
	if data, _ := os.ReadFile(abs); string(data) != "v1 + agent" {
		t.Fatalf("outside edit overwritten: %q", data)
	}
	ext := controls(drain(a.Peer))
	if len(ext) != 1 || ext[0]["type"] != "external" || ext[0]["hash"] != hashBytes([]byte("v1 + agent")) {
		t.Fatalf("leader should be told instead, got %v", ext)
	}
	// Having folded it in, the leader saves against the version it saw.
	room.Handle(a.Peer, saveMsg(hashBytes([]byte("v1 + agent")), "v1 + agent + typing"))
	if data, _ := os.ReadFile(abs); string(data) != "v1 + agent + typing" {
		t.Fatalf("merged save not written: %q", data)
	}
}

func TestDemotedLeaderStepsDown(t *testing.T) {
	h, proj, abs := setup(t, "x")
	a := newTestPeer("alice", projects.RoleEditor)
	room := join(t, h, proj, abs, a.Peer)
	b := newTestPeer("bob", projects.RoleViewer)
	join(t, h, proj, abs, b.Peer)
	drain(a.Peer)
	drain(b.Peer)

	a.role = projects.RoleViewer
	b.role = projects.RoleEditor
	room.Handle(b.Peer, Frame(MsgAwareness, 0, []byte{1}))
	if room.leader != b.Peer {
		t.Fatal("the promoted editor should lead")
	}
	got := controls(drain(a.Peer))
	if len(got) == 0 || got[0]["type"] != "leader" || got[0]["leader"] != false {
		t.Fatalf("demoted leader should be told, got %v", got)
	}
}

func TestLongLogReplayDoesNotUseOutbox(t *testing.T) {
	h, proj, abs := setup(t, "x")
	a := newTestPeer("alice", projects.RoleEditor)
	room := join(t, h, proj, abs, a.Peer)
	drain(a.Peer)
	// Without a compacting leader answer, the log grows past the outbox.
	for i := 0; i < outboxSize+10; i++ {
		_ = room.logs[ChanComments].append([]byte{byte(i)})
	}
	b := newTestPeer("bob", projects.RoleViewer)
	_, replay, err := h.Join(proj, "main.typ", abs, b.Peer)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-b.Done():
		t.Fatal("joining a long room must not kick the peer")
	default:
	}
	if len(replay) < outboxSize+10 {
		t.Fatalf("replay has %d messages", len(replay))
	}
}

func TestCommentsCompactWithoutLeader(t *testing.T) {
	h, proj, abs := setup(t, "x")
	c := newTestPeer("carol", projects.RoleCommenter)
	room := join(t, h, proj, abs, c.Peer)
	drain(c.Peer)
	for i := 0; i <= compactRecords; i++ {
		room.Handle(c.Peer, Frame(MsgUpdate, ChanComments, []byte{byte(i)}))
	}
	asked := controls(drain(c.Peer))
	if len(asked) != 1 || asked[0]["type"] != "compact" {
		t.Fatalf("a commenter should be asked to compact comments, got %v", asked)
	}
	room.Handle(c.Peer, Frame(MsgCompact, ChanComments, []byte("SNAP")))
	if string(room.logs[ChanComments].records[0]) != "SNAP" {
		t.Fatal("comments log not compacted")
	}
}

func TestTornLogIsTruncatedBeforeAppending(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.ylog")
	l, _ := openLog(p)
	_ = l.append([]byte("ok"))
	f, _ := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644)
	_, _ = f.Write([]byte{20, 'x', 'y'}) // record claims 20 bytes, has 2
	f.Close()
	l, _ = openLog(p)
	_ = l.append([]byte("after"))
	again, _ := openLog(p)
	if len(again.records) != 2 || string(again.records[1]) != "after" {
		t.Fatalf("records after a torn write: %q", again.records)
	}
}
