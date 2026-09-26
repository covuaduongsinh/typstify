package server

import (
	"strings"
	"testing"
)

func TestRepairTypstContent_HoistImport(t *testing.T) {
	// Import at bottom should be moved to line 1
	input := `#puzzle-card(
  "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6",
  number: 1,
  title: "Đòn đánh đôi",
  turn: "w",
  difficulty: 2,
  hint: "Gỡ ghim và phản công",
  solution: "1. dxe4 Bxc3+ 2. bxc3 Trắng hơn quân."
)

#import "@local/chessbook:0.1.0": *`

	got := string(repairTypstContent([]byte(input)))
	if !strings.HasPrefix(got, `#import "@local/chessbook:0.1.0": *`) {
		t.Fatalf("expected #import at start of file, got:\n%s", got)
	}
	if strings.Count(got, `@local/chessbook:0.1.0`) != 1 {
		t.Fatalf("expected exactly 1 import line, got:\n%s", got)
	}
	if strings.HasSuffix(strings.TrimSpace(got), `#import "@local/chessbook:0.1.0": *`) {
		t.Fatalf("import should have been removed from bottom of file, got:\n%s", got)
	}
}

func TestRepairTypstContent_StripMocksAndHoist(t *testing.T) {
	input := `#puzzle-card("fen", number: 1)

#let turn-box(turn) = {
  box()
}

#let lesson-header(lesson-num: 1) = {
  block()[]
}

#import "@local/chessbook:0.1.0": *`

	got := string(repairTypstContent([]byte(input)))
	if !strings.HasPrefix(got, `#import "@local/chessbook:0.1.0": *`) {
		t.Fatalf("expected #import at start of file, got:\n%s", got)
	}
	if strings.Contains(got, "#let turn-box") {
		t.Fatalf("expected mock #let turn-box to be stripped, got:\n%s", got)
	}
	if strings.Contains(got, "#let lesson-header") {
		t.Fatalf("expected mock #let lesson-header to be stripped, got:\n%s", got)
	}
	if !strings.Contains(got, `#puzzle-card("fen", number: 1)`) {
		t.Fatalf("expected puzzle-card call to be preserved, got:\n%s", got)
	}
}

func TestRepairTypstContent_NonChessDoc(t *testing.T) {
	input := `= Normal Document

This is just a regular text document without any chess items.`

	got := string(repairTypstContent([]byte(input)))
	if got != input {
		t.Fatalf("expected non-chess document to remain untouched, got:\n%s", got)
	}
}
