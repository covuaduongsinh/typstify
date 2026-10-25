package history

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func osWriter(dir string) Writer {
	return Writer{
		Write: func(rel string, data []byte) error {
			p := filepath.Join(dir, filepath.FromSlash(rel))
			_ = os.MkdirAll(filepath.Dir(p), 0o755)
			return os.WriteFile(p, data, 0o644)
		},
		Remove: func(rel string) error { return os.Remove(filepath.Join(dir, filepath.FromSlash(rel))) },
	}
}

func countObjects(s *Store) int {
	n := 0
	_ = filepath.WalkDir(filepath.Join(s.dir, "objects"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			n++
		}
		return nil
	})
	return n
}

func TestTakeDedupesAndSkipsUnchanged(t *testing.T) {
	proj := t.TempDir()
	write(t, proj, "main.typ", "= A")
	write(t, proj, "img/logo.png", "PNG")
	write(t, proj, ".git/HEAD", "ref")
	s := newStore(t.TempDir())

	first, created, err := s.Take(proj, "alice", "", KindAuto, false)
	if err != nil || !created || first.Files != 2 || first.Added != 2 {
		t.Fatalf("first = %+v created=%v err=%v", first, created, err)
	}
	if _, created, _ := s.Take(proj, "alice", "", KindAuto, false); created {
		t.Fatal("an unchanged project must not get a new automatic version")
	}

	time.Sleep(20 * time.Millisecond)
	write(t, proj, "main.typ", "= B")
	write(t, proj, "new.typ", "x")
	second, created, _ := s.Take(proj, "bob", "Bản 2", KindManual, false)
	if !created || second.Modified != 1 || second.Added != 1 || second.Label != "Bản 2" {
		t.Fatalf("second = %+v", second)
	}
	// logo.png is stored once for both versions.
	if n := countObjects(s); n != 4 {
		t.Fatalf("objects = %d, want 4 (A, B, PNG, x)", n)
	}
	list, _ := s.List()
	if len(list) != 2 || list[0].ID != second.ID {
		t.Fatalf("list not newest first: %+v", list)
	}
	old, err := s.Read(first.ID, "main.typ")
	if err != nil || string(old) != "= A" {
		t.Fatalf("read = %q, %v", old, err)
	}
	if _, err := s.Read(first.ID, "new.typ"); !errors.Is(err, ErrNoFile) {
		t.Fatalf("want ErrNoFile, got %v", err)
	}
}

func TestCompareAndRestore(t *testing.T) {
	proj := t.TempDir()
	write(t, proj, "main.typ", "v1")
	write(t, proj, "keep.typ", "same")
	write(t, proj, "gone.typ", "old")
	s := newStore(t.TempDir())
	v1, _, _ := s.Take(proj, "alice", "v1", KindManual, false)

	time.Sleep(20 * time.Millisecond)
	write(t, proj, "main.typ", "v2")
	_ = os.Remove(filepath.Join(proj, "gone.typ"))
	write(t, proj, "added.typ", "new")

	changes, err := s.Compare(v1.ID, proj)
	if err != nil {
		t.Fatal(err)
	}
	status := map[string]string{}
	for _, c := range changes {
		status[c.Path] = c.Status
	}
	want := map[string]string{"main.typ": "modified", "keep.typ": "unchanged", "gone.typ": "deleted", "added.typ": "added"}
	for p, st := range want {
		if status[p] != st {
			t.Fatalf("%s: %s, want %s (all: %v)", p, status[p], st, status)
		}
	}

	// One file.
	safety, err := s.Restore(v1.ID, proj, "alice", []string{"main.typ"}, false, osWriter(proj))
	if err != nil || safety.Kind != KindRestore {
		t.Fatalf("restore: %+v %v", safety, err)
	}
	if b, _ := os.ReadFile(filepath.Join(proj, "main.typ")); string(b) != "v1" {
		t.Fatalf("main.typ = %q", b)
	}
	// The safety version still has what was overwritten.
	if b, _ := s.Read(safety.ID, "main.typ"); string(b) != "v2" {
		t.Fatalf("safety copy = %q", b)
	}

	// Whole version, removing newer files.
	if _, err := s.Restore(v1.ID, proj, "alice", nil, true, osWriter(proj)); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(proj, "gone.typ")); string(b) != "old" {
		t.Fatal("deleted file not restored")
	}
	if _, err := os.Stat(filepath.Join(proj, "added.typ")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("newer file should be removed when asked")
	}
}

func TestPruneThinsAutomaticVersionsOnly(t *testing.T) {
	proj := t.TempDir()
	s := newStore(t.TempDir())
	now := time.Now()
	var snaps []Snapshot
	add := func(at time.Time, kind, content string) {
		write(t, proj, "main.typ", content)
		snap, _, err := s.Take(proj, "x", "", kind, true)
		if err != nil {
			t.Fatal(err)
		}
		snap.At = at
		snaps = append(snaps, snap)
	}
	// Three automatic versions in the same hour ten days ago, one named.
	old := now.Add(-10 * 24 * time.Hour).Truncate(time.Hour)
	add(old, KindAuto, "a")
	add(old.Add(10*time.Minute), KindAuto, "b")
	add(old.Add(20*time.Minute), KindManual, "c")
	add(old.Add(30*time.Minute), KindAuto, "d")
	add(now, KindAuto, "e")
	// Backdate them in the index, then prune.
	if err := s.writeIndex(snaps); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.pruneLocked(now)
	s.mu.Unlock()

	list, _ := s.List()
	var kept []string
	for _, snap := range list {
		b, _ := s.Read(snap.ID, "main.typ")
		kept = append(kept, string(b))
	}
	// Newest first: e (recent), d (latest of that day), c (named).
	if len(kept) != 3 || kept[0] != "e" || kept[1] != "d" || kept[2] != "c" {
		t.Fatalf("kept %v", kept)
	}
	if n := countObjects(s); n != 3 {
		t.Fatalf("objects = %d, want 3 after sweeping a and b", n)
	}
}

func TestQuotaDropsOldestAutomaticVersions(t *testing.T) {
	proj := t.TempDir()
	s := newStore(t.TempDir())
	write(t, proj, "main.typ", "named")
	named, _, _ := s.Take(proj, "x", "giữ", KindManual, true)
	for i := 0; i < 4; i++ {
		time.Sleep(15 * time.Millisecond)
		write(t, proj, "main.typ", string(rune('a'+i))+"-big-enough-content")
		if _, _, err := s.Take(proj, "x", "", KindAuto, false); err != nil {
			t.Fatal(err)
		}
	}
	s.mu.Lock()
	s.quota = 1 // forces the drop of every automatic version but the newest
	s.pruneLocked(time.Now())
	s.mu.Unlock()
	list, _ := s.List()
	if len(list) != 2 || list[1].ID != named.ID || list[0].Kind != KindAuto {
		t.Fatalf("kept %+v", list)
	}
	if b, _ := s.Read(list[0].ID, "main.typ"); string(b) != "d-big-enough-content" {
		t.Fatalf("newest version lost: %q", b)
	}
}

func TestReadRejectsForeignIDs(t *testing.T) {
	s := newStore(t.TempDir())
	for _, id := range []string{"../../x", "..%2F..", "20261025T000000-zzzzzzzz", ""} {
		if _, err := s.Read(id, "main.typ"); !errors.Is(err, ErrNotFound) {
			t.Errorf("%q: want ErrNotFound, got %v", id, err)
		}
	}
}

func TestCompareWritesNothing(t *testing.T) {
	proj := t.TempDir()
	write(t, proj, "main.typ", "v1")
	s := newStore(t.TempDir())
	v1, _, _ := s.Take(proj, "x", "v1", KindManual, false)
	before := countObjects(s)
	write(t, proj, "main.typ", "changed")
	if _, err := s.Compare(v1.ID, proj); err != nil {
		t.Fatal(err)
	}
	if countObjects(s) != before {
		t.Fatal("Compare must not store contents")
	}
}
