package view

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"

	"looz.ws/typstify/agent"
)

func createTestPng() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestInputBoxImageAttachments(t *testing.T) {
	pngData := createTestPng()

	ib := &InputBox{}
	if ib.HasAttachments() {
		t.Fatalf("expected no attachments initially")
	}

	ok := ib.AddImageAttachment(pngData)
	if !ok {
		t.Fatalf("failed to add image attachment")
	}

	if !ib.HasAttachments() {
		t.Fatalf("expected attachments after adding")
	}

	attachments := ib.Attachments()
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}

	if attachments[0].MimeType != "image/png" {
		t.Fatalf("expected mime type image/png, got %s", attachments[0].MimeType)
	}

	blocks := ib.Blocks()
	if len(blocks) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(blocks))
	}

	if blocks[0].Image == nil {
		t.Fatalf("expected Image content block")
	}

	expectedB64 := base64.StdEncoding.EncodeToString(pngData)
	if blocks[0].Image.Data != expectedB64 {
		t.Fatalf("base64 data mismatch")
	}

	ib.ClearAttachments()
	if ib.HasAttachments() {
		t.Fatalf("expected no attachments after clear")
	}
}

func TestInputBoxTextAndClear(t *testing.T) {
	ib := newInputBox(&agent.ACPSession{})
	sample := "tăng kích thước 2 bàn cờ ở dưới bằng với bàn cờ ở trên cho đồng bộ"
	ib.Editor.SetText(sample)

	if got := ib.Editor.Text(); got != sample {
		t.Errorf("InputBox text = %q; want %q", got, sample)
	}

	blocks := ib.Blocks()
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	ib.Clear()
	if ib.Editor.Text() != "" {
		t.Fatalf("expected empty text after clear")
	}
}


