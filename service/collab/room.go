package collab

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"looz.ws/typstify/service/projects"
)

// Wire format: [type][channel][payload]. Mirrored in
// web/src/lib/collab/protocol.ts.
const (
	MsgUpdate    byte = 0 // Yjs update; logged and relayed
	MsgAwareness byte = 1 // y-protocols awareness; relayed only
	MsgControl   byte = 2 // server -> client JSON
	MsgSynced    byte = 3 // server -> client: log replay finished
	MsgCompact   byte = 4 // leader -> server: full state replacing the log
	MsgSave      byte = 5 // leader -> server: disk hash it has seen (64 hex) + text
)

const (
	ChanText     byte = 0
	ChanComments byte = 1
	numChannels       = 2
)

// A log longer than this asks the leader for a compacted state.
const (
	compactRecords = 256
	compactBytes   = 1 << 20
)

// maxAwareness bounds one awareness message (a cursor and a name).
const maxAwareness = 64 << 10

// Frame builds a wire message.
func Frame(typ, ch byte, payload []byte) []byte {
	return append([]byte{typ, ch}, payload...)
}

// outboxSize bounds the messages queued for one slow client; a client
// that falls this far behind is disconnected rather than stalling the
// room (it rebuilds its state from the log when it reconnects). It must
// also hold a full log replay on join: both channels just under the
// compaction threshold, plus what arrives meanwhile.
const outboxSize = 4096

// Peer is one connected browser.
type Peer struct {
	ID   uint64
	User string
	Name string
	// Role returns the user's current role in the project; false means the
	// access was revoked and the peer is disconnected.
	Role func() (projects.Role, bool)

	out       chan []byte
	done      chan struct{}
	closeOnce sync.Once
	reason    string
	fatal     bool
}

func NewPeer(user, name string, role func() (projects.Role, bool)) *Peer {
	return &Peer{User: user, Name: name, Role: role, out: make(chan []byte, outboxSize), done: make(chan struct{})}
}

// Outbox delivers the messages to write to this peer's socket.
func (p *Peer) Outbox() <-chan []byte { return p.out }

// Done is closed when the room drops the peer; Reason says why, and Fatal
// whether reconnecting is pointless (access revoked) rather than a
// transient condition (the client fell behind).
func (p *Peer) Done() <-chan struct{} { return p.done }
func (p *Peer) Reason() string        { return p.reason }
func (p *Peer) Fatal() bool           { return p.fatal }

func (p *Peer) send(msg []byte) {
	select {
	case p.out <- msg:
	case <-p.done:
	default:
		p.kick("client too slow", false)
	}
}

func (p *Peer) kick(reason string, fatal bool) {
	p.closeOnce.Do(func() {
		p.reason = reason
		p.fatal = fatal
		close(p.done)
	})
}

type roomMeta struct {
	Path     string `json:"path"`
	Epoch    string `json:"epoch"`
	DiskHash string `json:"diskHash"`
}

// Room is the collaboration state of one file.
type Room struct {
	hub  *Hub
	key  string
	abs  string
	dir  string
	meta roomMeta

	// base is the file content the shared text last matched on disk. An
	// outside edit is reported as base -> new disk content, so the leader
	// can replay just that change on top of the current text (3-way)
	// instead of overwriting edits that never reached the disk.
	base []byte

	mu           sync.Mutex
	peers        []*Peer // join order: the earliest editor leads
	leader       *Peer
	logs         [numChannels]*updateLog
	compactMark  [numChannels]int   // log length when compaction was asked, -1 if not
	compactPeer  [numChannels]*Peer // who was asked
	notifiedHash string             // disk version already sent to the leader
	statSize     int64
	statMod      time.Time
}

func newEpoch() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func randomClientID() uint64 {
	var b [4]byte
	_, _ = rand.Read(b[:])
	// Yjs client ids are uint32; 0 is avoided only for readability.
	return uint64(binary.LittleEndian.Uint32(b[:])) | 1
}

func openRoom(h *Hub, key, rel, abs, dir string) (*Room, error) {
	r := &Room{hub: h, key: key, abs: abs, dir: dir, compactMark: [numChannels]int{-1, -1}}
	for ch, name := range []string{"text.ylog", "comments.ylog"} {
		l, err := openLog(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		r.logs[ch] = l
	}
	if data, err := os.ReadFile(filepath.Join(dir, "meta.json")); err == nil {
		_ = json.Unmarshal(data, &r.meta)
	}
	r.meta.Path = rel

	disk, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	r.rememberStat()
	if b, err := os.ReadFile(filepath.Join(dir, "base.txt")); err == nil && hashBytes(b) == r.meta.DiskHash {
		r.base = b
	} else {
		r.base = disk
	}
	if len(r.logs[ChanText].records) == 0 || r.meta.Epoch == "" {
		// A new history: comments anchored in an old one cannot survive.
		if err := r.logs[ChanComments].replace(nil); err != nil {
			return nil, err
		}
		if err := r.logs[ChanText].replace([][]byte{EncodeInsertText(randomClientID(), RootText, string(disk))}); err != nil {
			return nil, err
		}
		r.meta.Epoch = newEpoch()
		if err := r.setBase(disk); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// setBase records content as what the shared text matches on disk.
func (r *Room) setBase(content []byte) error {
	r.base = append([]byte(nil), content...)
	r.meta.DiskHash = hashBytes(content)
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(r.dir, "base.txt.tmp")
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(r.dir, "base.txt")); err != nil {
		return err
	}
	return r.saveMeta()
}

func (r *Room) saveMeta() error {
	data, err := json.Marshal(r.meta)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(r.dir, "meta.json.tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(r.dir, "meta.json"))
}

func (r *Room) rememberStat() {
	if info, err := os.Stat(r.abs); err == nil {
		r.statSize, r.statMod = info.Size(), info.ModTime()
	}
}

type peerView struct {
	ID     uint64        `json:"id"`
	User   string        `json:"user"`
	Name   string        `json:"name"`
	Role   projects.Role `json:"role"`
	Leader bool          `json:"leader"`
}

func (r *Room) peerViews() []peerView {
	out := make([]peerView, 0, len(r.peers))
	for _, p := range r.peers {
		role, _ := p.Role()
		out = append(out, peerView{p.ID, p.User, p.Name, role, p == r.leader})
	}
	return out
}

func control(v any) []byte {
	data, _ := json.Marshal(v)
	return Frame(MsgControl, 0, data)
}

func (r *Room) broadcast(msg []byte, except *Peer) {
	for _, p := range r.peers {
		if p != except {
			p.send(msg)
		}
	}
}

func (r *Room) broadcastPeers() {
	r.broadcast(control(map[string]any{"type": "peers", "peers": r.peerViews()}), nil)
}

func canEdit(p *Peer) bool {
	role, ok := p.Role()
	return ok && role.AtLeast(projects.RoleEditor)
}

// join adds p and returns the messages that bring it up to date (hello,
// the logs, synced). They are written to the socket directly, ahead of the
// outbox: a long log must not overflow the outbox before the writer runs.
// Later messages queue in the outbox, so ordering is kept.
func (r *Room) join(p *Peer) [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.peers = append(r.peers, p)
	if r.leader == nil && canEdit(p) {
		r.setLeader(p)
	}
	role, _ := p.Role()
	replay := [][]byte{control(map[string]any{
		"type":     "hello",
		"you":      p.ID,
		"role":     role,
		"epoch":    r.meta.Epoch,
		"diskHash": r.meta.DiskHash,
		"leader":   p == r.leader,
		"peers":    r.peerViews(),
	})}
	for ch, l := range r.logs {
		for _, rec := range l.records {
			replay = append(replay, Frame(MsgUpdate, byte(ch), rec))
		}
	}
	replay = append(replay, Frame(MsgSynced, 0, nil))
	r.broadcast(control(map[string]any{"type": "peers", "peers": r.peerViews()}), p)
	if p == r.leader {
		r.checkDiskLocked(true)
	}
	return replay
}

func (r *Room) setLeader(p *Peer) {
	r.leader = p
	r.compactMark = [numChannels]int{-1, -1}
	r.compactPeer = [numChannels]*Peer{}
	r.notifiedHash = ""
}

// electLocked keeps the room led by an editor: a leader whose role was
// lowered steps down, and a room without one picks its earliest editor
// (who may have just been promoted).
func (r *Room) electLocked() {
	if r.leader != nil && !canEdit(r.leader) {
		r.leader.send(control(map[string]any{"type": "leader", "leader": false}))
		r.leader = nil
	}
	if r.leader != nil {
		return
	}
	for _, q := range r.peers {
		if canEdit(q) {
			r.setLeader(q)
			q.send(control(map[string]any{"type": "leader", "leader": true}))
			r.checkDiskLocked(true)
			r.broadcastPeers()
			return
		}
	}
}

// leave removes p and hands leadership to the next editor. It reports
// whether the room is now empty.
func (r *Room) leave(p *Peer) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, q := range r.peers {
		if q == p {
			r.peers = append(r.peers[:i], r.peers[i+1:]...)
			break
		}
	}
	for ch, q := range r.compactPeer {
		if q == p {
			r.compactPeer[ch] = nil
			r.compactMark[ch] = -1
		}
	}
	if r.leader == p {
		r.leader = nil
	}
	r.electLocked()
	r.broadcastPeers()
	return len(r.peers) == 0
}

// handle applies one message from p, dropping what p's role does not
// allow. A peer whose access was revoked is disconnected.
func (r *Room) handle(p *Peer, msg []byte) {
	if len(msg) < 2 {
		return
	}
	typ, ch, payload := msg[0], msg[1], msg[2:]
	role, ok := p.Role()
	if !ok {
		p.kick("access revoked", true)
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.electLocked()
	switch typ {
	case MsgUpdate:
		if int(ch) >= numChannels || len(payload) == 0 {
			return
		}
		min := projects.RoleEditor
		if ch == ChanComments {
			min = projects.RoleCommenter
		}
		if !role.AtLeast(min) {
			return
		}
		if err := r.logs[ch].append(payload); err != nil {
			// Not relayed either, so nobody holds an edit the log lacks;
			// the sender resends it after reconnecting.
			p.kick("không lưu được thay đổi", false)
			return
		}
		r.broadcast(msg, p)
		r.maybeCompactLocked(ch)
	case MsgAwareness:
		if len(payload) <= maxAwareness {
			r.broadcast(msg, p)
		}
	case MsgCompact:
		if int(ch) >= numChannels || p != r.compactPeer[ch] || r.compactMark[ch] < 0 {
			return
		}
		if min := channelRole(ch); !role.AtLeast(min) {
			return
		}
		// Updates logged after the request may not be in the snapshot;
		// keeping them is harmless (Yjs updates are idempotent).
		tail := r.logs[ch].records[r.compactMark[ch]:]
		records := append([][]byte{append([]byte(nil), payload...)}, tail...)
		if err := r.logs[ch].replace(records); err == nil {
			r.compactMark[ch] = -1
			r.compactPeer[ch] = nil
		}
	case MsgSave:
		if p != r.leader || !role.AtLeast(projects.RoleEditor) || len(payload) < 64 {
			return
		}
		r.saveLocked(string(payload[:64]), payload[64:])
	}
}

func channelRole(ch byte) projects.Role {
	if ch == ChanComments {
		return projects.RoleCommenter
	}
	return projects.RoleEditor
}

// maybeCompactLocked asks for a compacted state when a log grows long:
// the leader, or for comments any peer allowed to write them (a room of
// commenters has no leader but still adds comments).
func (r *Room) maybeCompactLocked(ch byte) {
	l := r.logs[ch]
	if r.compactMark[ch] >= 0 || (len(l.records) <= compactRecords && l.bytes <= compactBytes) {
		return
	}
	target := r.leader
	if target == nil && ch == ChanComments {
		for _, q := range r.peers {
			if role, ok := q.Role(); ok && role.AtLeast(projects.RoleCommenter) {
				target = q
				break
			}
		}
	}
	if target == nil {
		return
	}
	r.compactMark[ch] = len(l.records)
	r.compactPeer[ch] = target
	target.send(control(map[string]any{"type": "compact", "channel": ch}))
}

// saveLocked writes the leader's text. seen is the hash of the disk
// version the leader's text already includes: if the file changed since
// (an outside edit not folded in yet, or kept because of a conflict), it
// is not overwritten; the leader is told about it instead.
func (r *Room) saveLocked(seen string, text []byte) {
	current, err := os.ReadFile(r.abs)
	if err == nil {
		h := hashBytes(current)
		if h != r.meta.DiskHash && h != seen {
			r.notifiedHash = ""
			r.checkDiskLocked(true)
			return
		}
		if bytes.Equal(current, text) {
			// Already on disk (e.g. the leader just folded in an outside
			// edit): the text and the file agree again.
			if h != r.meta.DiskHash {
				_ = r.setBase(current)
			}
			r.notifiedHash = ""
			return
		}
	}
	written, err := r.hub.Save(r.abs, text)
	if err != nil {
		r.leader.send(control(map[string]any{"type": "error", "message": "không ghi được tệp"}))
		return
	}
	_ = r.setBase(written)
	r.notifiedHash = ""
	r.rememberStat()
	// Saving may rewrite the text (e.g. repairing chessbook imports): hand
	// the change back so the shared document matches the file.
	if !bytes.Equal(written, text) {
		r.notifiedHash = r.meta.DiskHash
		r.leader.send(control(map[string]any{"type": "external", "base": string(text), "text": string(written), "hash": r.meta.DiskHash}))
	}
}

// checkDiskLocked tells the leader when the file changed outside the room
// (agent edits, Dropbox, the desktop app). force skips the cheap
// size/mtime check.
func (r *Room) checkDiskLocked(force bool) {
	info, err := os.Stat(r.abs)
	if err != nil {
		return
	}
	if !force && info.Size() == r.statSize && info.ModTime().Equal(r.statMod) {
		return
	}
	r.statSize, r.statMod = info.Size(), info.ModTime()
	data, err := os.ReadFile(r.abs)
	if err != nil {
		return
	}
	h := hashBytes(data)
	if h == r.meta.DiskHash || h == r.notifiedHash || r.leader == nil {
		return
	}
	r.notifiedHash = h
	r.leader.send(control(map[string]any{"type": "external", "base": string(r.base), "text": string(data), "hash": h}))
}

// Hub owns the open rooms.
type Hub struct {
	// Base is the storage directory (<storage>/collab).
	Base string
	// Save writes a file the way the HTTP API does and returns the bytes
	// actually written.
	Save func(abs string, data []byte) ([]byte, error)

	mu     sync.Mutex
	rooms  map[string]*Room
	nextID uint64
}

var ErrTooLarge = errors.New("tệp quá lớn để cộng tác")

// MaxFileSize bounds files opened for co-editing.
const MaxFileSize = 4 << 20

func roomDir(base, projectDir, rel string) string {
	a := sha256.Sum256([]byte(projectDir))
	b := sha256.Sum256([]byte(rel))
	return filepath.Join(base, hex.EncodeToString(a[:8]), hex.EncodeToString(b[:8]))
}

// roomKey identifies a file; on Windows paths are case-insensitive, so
// "Main.typ" and "main.typ" share a room.
func roomKey(projectDir, rel string) (string, string, string) {
	if runtime.GOOS == "windows" {
		projectDir, rel = strings.ToLower(projectDir), strings.ToLower(rel)
	}
	return projectDir + "\x00" + rel, projectDir, rel
}

// IsOpen reports whether rel in projectDir is being co-edited.
func (h *Hub) IsOpen(projectDir, rel string) bool {
	key, _, _ := roomKey(projectDir, rel)
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.rooms[key]
	return ok
}

// Join opens (or reuses) the room of rel in projectDir, adds p to it and
// returns the messages to send p first (see Room.join).
func (h *Hub) Join(projectDir, rel, abs string, p *Peer) (*Room, [][]byte, error) {
	if info, err := os.Stat(abs); err != nil {
		return nil, nil, err
	} else if info.Size() > MaxFileSize {
		return nil, nil, ErrTooLarge
	}
	key, dirKey, relKey := roomKey(projectDir, rel)
	h.mu.Lock()
	if h.rooms == nil {
		h.rooms = map[string]*Room{}
	}
	h.nextID++
	p.ID = h.nextID
	r, ok := h.rooms[key]
	if !ok {
		var err error
		r, err = openRoom(h, key, rel, abs, roomDir(h.Base, dirKey, relKey))
		if err != nil {
			h.mu.Unlock()
			return nil, nil, err
		}
		h.rooms[key] = r
	}
	// Joined under h.mu (lock order: hub, then room), so a concurrent Leave
	// of the last peer cannot drop the room between lookup and join and
	// leave this peer alone in a room nobody else will find.
	replay := r.join(p)
	h.mu.Unlock()
	return r, replay, nil
}

// Handle applies a message from p.
func (r *Room) Handle(p *Peer, msg []byte) { r.handle(p, msg) }

// Leave removes p; the last peer closes the room (its logs stay on disk).
func (h *Hub) Leave(r *Room, p *Peer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r.leave(p) && h.rooms[r.key] == r {
		delete(h.rooms, r.key)
	}
	p.kick("left", false)
}

// CheckDisk looks for outside changes to the files of open rooms.
func (h *Hub) CheckDisk() {
	h.mu.Lock()
	rooms := make([]*Room, 0, len(h.rooms))
	for _, r := range h.rooms {
		rooms = append(rooms, r)
	}
	h.mu.Unlock()
	for _, r := range rooms {
		r.mu.Lock()
		r.checkDiskLocked(false)
		r.mu.Unlock()
	}
}
