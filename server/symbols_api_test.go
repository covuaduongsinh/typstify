package server

import (
	"encoding/json"
	"testing"
)

func TestParseSymbols(t *testing.T) {
	var res any
	_ = json.Unmarshal([]byte(`{"symbols":[
		{"category":"arrow","glyph":"<svg/>","id":"sym.arrow.r","value":"→"},
		{"category":"greek","glyph":"<svg/>","id":"sym.Alpha","value":"Α"},
		{"id":"emoji.cat","value":"🐈"},
		{"id":"sym.empty","value":""}
	]}`), &res)
	got := parseSymbols(res)
	if len(got) != 2 || got[0].Name != "Alpha" || got[1].Name != "arrow.r" || got[1].Char != "→" || got[1].Category != "arrow" {
		t.Fatalf("parsed %+v", got)
	}
	if parseSymbols(nil) == nil || len(parseSymbols(map[string]any{"symbols": "x"})) != 0 {
		t.Fatal("bad input must give an empty list")
	}
}
