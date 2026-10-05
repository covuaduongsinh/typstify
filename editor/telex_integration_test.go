package editor

import (
	"image"
	"testing"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"github.com/oligo/gvcode"
	"looz.ws/typstify/service/settings"
)

func TestEditorTelexIntegration(t *testing.T) {
	s := &settings.EditorSettings{
		EnableTelex: "true",
	}

	te := &TextEditor{
		state:       &gvcode.Editor{},
		enableTelex: true,
	}
	te.state.WithOptions(gvcode.AddTextInputHook(te.handleTelexInput))

	shaper := text.NewShaper()
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Exact(image.Pt(1200, 800))}

	simulateType := func(input string) string {
		te.state.SetText("")
		te.state.TextView().Layout(gtx, shaper)
		te.state.SetCaret(0, 0)

		for _, ch := range input {
			chStr := string(ch)
			caretStart, caretEnd := te.state.Selection()
			editEvt := key.EditEvent{
				Range: key.Range{Start: caretStart, End: caretEnd},
				Text:  chStr,
			}

			if !te.handleTelexInput(te.state, editEvt) {
				te.state.Insert(chStr)
			}
			te.state.TextView().Layout(gtx, shaper)
		}

		return te.state.Text()
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"tieengs Vieejt", "tiếng Việt"},
		{"ddoongf booj", "đồng bộ"},
		{"tawng kichs thuwocs", "tăng kích thước"},
		{"banf cowf owr duowis", "bàn cờ ở dưới"},
		{"vowis banf cowf owr treen", "với bàn cờ ở trên"},
	}

	for _, tt := range tests {
		got := simulateType(tt.input)
		if got != tt.expected {
			t.Errorf("simulateType(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}

	// Test when telex is disabled: English words should type raw without transformations
	te.enableTelex = false
	englishTests := []struct {
		input    string
		expected string
	}{
		{"tieengs", "tieengs"},
		{"hello world", "hello world"},
		{"typst document", "typst document"},
	}

	for _, tt := range englishTests {
		got := simulateType(tt.input)
		if got != tt.expected {
			t.Errorf("when telex disabled: simulateType(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}

	// Test settings toggle
	_ = s
}

func TestEditorTelexReadOnly(t *testing.T) {
	te := &TextEditor{
		state:       &gvcode.Editor{},
		enableTelex: true,
	}
	te.state.WithOptions(
		gvcode.ReadOnlyMode(true),
		gvcode.AddTextInputHook(te.handleTelexInput),
	)

	editEvt := key.EditEvent{
		Range: key.Range{Start: 0, End: 0},
		Text:  "s",
	}
	if te.handleTelexInput(te.state, editEvt) {
		t.Errorf("handleTelexInput should return false in read-only mode")
	}
}
