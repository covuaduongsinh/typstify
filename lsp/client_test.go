package lsp

import (
	"path/filepath"
	"sync"
	"testing"

	"looz.ws/typstify/lsp/protocol"
)

func TestUpdateDiagnostics(t *testing.T) {
	client := &Client{
		diagnostics: []*DocDiagnostics{},
	}

	pathEmpty, _ := filepath.Abs("empty.typ")
	uriEmpty := protocol.URIFromPath(pathEmpty)

	// 1. Receiving empty diagnostics for a new file should not panic and shouldn't add anything.
	client.updateDiagnostics(DocDiagnostics{
		URI:         uriEmpty,
		Diagnostics: []protocol.Diagnostic{},
	})

	if len(client.diagnostics) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(client.diagnostics))
	}
	if d := client.Diagnostics(pathEmpty); d != nil {
		t.Fatalf("expected nil diagnostics for %s, got %v", pathEmpty, d)
	}

	pathDoc, _ := filepath.Abs("doc.typ")
	uriDoc := protocol.URIFromPath(pathDoc)

	// 2. Receiving non-empty diagnostics for a new file should append it.
	client.updateDiagnostics(DocDiagnostics{
		URI: uriDoc,
		Diagnostics: []protocol.Diagnostic{
			{Message: "syntax error"},
		},
	})

	if len(client.diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic entry, got %d", len(client.diagnostics))
	}
	if d := client.Diagnostics(pathDoc); d == nil || len(d.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic for %s, got %v", pathDoc, d)
	}

	// 3. Receiving updated diagnostics for an existing file should update it.
	client.updateDiagnostics(DocDiagnostics{
		URI: uriDoc,
		Diagnostics: []protocol.Diagnostic{
			{Message: "syntax error 1"},
			{Message: "syntax error 2"},
		},
	})

	if len(client.diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic entry, got %d", len(client.diagnostics))
	}
	if d := client.Diagnostics(pathDoc); d == nil || len(d.Diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics for %s, got %v", pathDoc, d)
	}

	// 4. Receiving empty diagnostics for an existing file should remove it.
	client.updateDiagnostics(DocDiagnostics{
		URI:         uriDoc,
		Diagnostics: []protocol.Diagnostic{},
	})

	if len(client.diagnostics) != 0 {
		t.Fatalf("expected 0 diagnostic entries after clearing, got %d", len(client.diagnostics))
	}
	if d := client.Diagnostics(pathDoc); d != nil {
		t.Fatalf("expected nil diagnostics after clearing, got %v", d)
	}
}

func TestUpdateDiagnosticsConcurrent(t *testing.T) {
	client := &Client{
		diagnostics: []*DocDiagnostics{},
	}

	pathConcurrent, _ := filepath.Abs("concurrent.typ")
	uriConcurrent := protocol.URIFromPath(pathConcurrent)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(id int) {
			defer wg.Done()
			client.updateDiagnostics(DocDiagnostics{
				URI: uriConcurrent,
				Diagnostics: []protocol.Diagnostic{
					{Message: "error"},
				},
			})
		}(i)

		go func(id int) {
			defer wg.Done()
			_ = client.Diagnostics(pathConcurrent)
		}(i)
	}

	wg.Wait()
}
