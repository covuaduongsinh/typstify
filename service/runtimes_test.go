package service

import (
	"errors"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }
func newTestRuntimes(limit int) (*Runtimes, *fakeClock) {
	clock := &fakeClock{t: time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)}
	r := newRuntimes(nil, nil, nil, func() int { return limit })
	r.now = clock.now
	return r, clock
}

func dir(name string) string { return filepath.Join(string(filepath.Separator)+"projects", name) }

func TestRuntimesShareOneRuntimePerProject(t *testing.T) {
	r, _ := newTestRuntimes(3)
	a1, rel1, err := r.Acquire(dir("a"))
	if err != nil {
		t.Fatal(err)
	}
	a2, rel2, err := r.Acquire(dir("a") + string(filepath.Separator))
	if err != nil {
		t.Fatal(err)
	}
	if a1 != a2 {
		t.Fatal("two acquires of the same project got different runtimes")
	}
	if got := len(r.Open()); got != 1 {
		t.Fatalf("open = %d, want 1", got)
	}
	rel1()
	rel1() // idempotent
	rel2()
	if a1.refs != 0 {
		t.Fatalf("refs = %d after releasing both, want 0", a1.refs)
	}
}

func TestRuntimesCaseInsensitiveOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("paths are case-sensitive here")
	}
	r, _ := newTestRuntimes(3)
	a, _, _ := r.Acquire(`C:\Projects\Book`)
	b, _, _ := r.Acquire(`c:\projects\book`)
	if a != b {
		t.Fatal("same directory in another case opened a second runtime")
	}
}

func TestRuntimesEvictLeastRecentlyUsedIdle(t *testing.T) {
	r, clock := newTestRuntimes(2)
	_, relA, _ := r.Acquire(dir("a"))
	clock.advance(time.Minute)
	_, relB, _ := r.Acquire(dir("b"))
	clock.advance(time.Minute)
	relB() // b idle, used later than a
	clock.advance(time.Minute)
	relA() // a idle now, most recently released

	c, relC, err := r.Acquire(dir("c"))
	if err != nil {
		t.Fatalf("acquire over the limit with idle runtimes: %v", err)
	}
	defer relC()
	open := r.Open()
	slices.Sort(open)
	want := []string{dir("a"), dir("c")}
	if !slices.Equal(open, want) {
		t.Fatalf("open = %v, want %v (b was the least recently used idle one)", open, want)
	}
	if c.closed {
		t.Fatal("the new runtime is closed")
	}
}

func TestRuntimesRefuseWhenAllInUse(t *testing.T) {
	r, _ := newTestRuntimes(2)
	a, _, _ := r.Acquire(dir("a"))
	_, _, _ = r.Acquire(dir("b"))
	_, _, err := r.Acquire(dir("c"))
	if !errors.Is(err, ErrTooManyProjects) {
		t.Fatalf("err = %v, want ErrTooManyProjects", err)
	}
	if a.closed {
		t.Fatal("a runtime in use was closed")
	}
	// An open project can still be joined at the limit.
	if _, rel, err := r.Acquire(dir("a")); err != nil {
		t.Fatalf("joining an open project at the limit: %v", err)
	} else {
		rel()
	}
}

func TestRuntimesSweepClosesOnlyIdleAndOld(t *testing.T) {
	r, clock := newTestRuntimes(3)
	_, relOld, _ := r.Acquire(dir("old"))
	relOld()
	held, relHeld, _ := r.Acquire(dir("held"))
	defer relHeld()
	clock.advance(11 * time.Minute)
	_, relFresh, _ := r.Acquire(dir("fresh"))
	relFresh()

	if n := r.Sweep(10 * time.Minute); n != 1 {
		t.Fatalf("swept %d, want 1", n)
	}
	open := strings.Join(r.Open(), ",")
	if strings.Contains(open, "old") || !strings.Contains(open, "held") || !strings.Contains(open, "fresh") {
		t.Fatalf("open after sweep = %s", open)
	}
	if held.closed {
		t.Fatal("a held runtime was closed")
	}
}

func TestRuntimesTouchKeepsIdleRuntimeWarm(t *testing.T) {
	r, clock := newTestRuntimes(3)
	_, rel, _ := r.Acquire(dir("a"))
	rel()
	clock.advance(9 * time.Minute)
	r.Touch(dir("a"))
	clock.advance(9 * time.Minute)
	if n := r.Sweep(10 * time.Minute); n != 0 {
		t.Fatalf("swept %d runtimes used 9 minutes ago", n)
	}
}

func TestRuntimesClosedRuntimeHasNoLSP(t *testing.T) {
	r, _ := newTestRuntimes(3)
	a, rel, _ := r.Acquire(dir("a"))
	rel()
	r.Close()
	if !a.closed {
		t.Fatal("Close left a runtime open")
	}
	if a.LSP() != nil || a.Preview() != nil {
		t.Fatal("a closed runtime still hands out tinymist")
	}
	if len(r.Open()) != 0 {
		t.Fatal("Close left runtimes registered")
	}
}
