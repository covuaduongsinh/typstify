package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"looz.ws/typstify/service/wasmassets"
)

func TestCreateProjectFromTemplate(t *testing.T) {
	root := tempStorage(t)
	s := New(testService(), Options{ProjectRoot: root, StorageDir: tempStorage(t)})
	// A local package root stands in for the registry download.
	pkgs := tempStorage(t)
	s.wasmPackages = &wasmassets.Packages{SearchDirs: func() []string { return []string{pkgs} }}
	demo := filepath.Join(pkgs, "preview", "demo", "0.1.0")
	plain := filepath.Join(pkgs, "preview", "plain", "1.0.0")
	for path, content := range map[string]string{
		filepath.Join(demo, "typst.toml"):                  "[package]\nname = \"demo\"\nversion = \"0.1.0\"\n\n[template]\npath = \"template\"\nentrypoint = \"doc/main.typ\"\n",
		filepath.Join(demo, "template", "doc", "main.typ"): `#import "@preview/demo:0.1.0": *`,
		filepath.Join(demo, "template", "refs.bib"):        "@book{x}",
		filepath.Join(demo, "README.md"):                   "# Demo",
		filepath.Join(plain, "typst.toml"):                 "[package]\nname = \"plain\"\nversion = \"1.0.0\"\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	c := &apiClient{t: t, h: s.Handler()}
	mustStatus(t, c.do("POST", "/api/auth/register", `{"username":"owner","password":"pw1234"}`), 200, "register")

	w := c.do("POST", "/api/projects", `{"name":"Luận văn","template":{"name":"demo","version":"0.1.0"}}`)
	mustStatus(t, w, 200, "create from template")
	var created struct {
		ID   string `json:"id"`
		Main string `json:"main"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	p, _ := s.projects.Get(created.ID)
	if created.Main != "doc/main.typ" {
		t.Fatalf("main = %q", created.Main)
	}
	for _, f := range []string{"doc/main.typ", "refs.bib"} {
		if _, err := os.Stat(filepath.Join(p.Dir, filepath.FromSlash(f))); err != nil {
			t.Errorf("template file %s missing", f)
		}
	}
	if _, err := os.Stat(filepath.Join(p.Dir, "main.typ")); err == nil {
		t.Error("default main.typ written next to the template")
	}

	// A package without [template] is refused and leaves no folder behind.
	before, _ := os.ReadDir(root)
	mustStatus(t, c.do("POST", "/api/projects", `{"name":"Hỏng","template":{"name":"plain","version":"1.0.0"}}`), 400, "no template")
	mustStatus(t, c.do("POST", "/api/projects", `{"name":"Sai","template":{"name":"../x","version":"1.0.0"}}`), 400, "bad name")
	if after, _ := os.ReadDir(root); len(after) != len(before) {
		t.Fatalf("failed creates left folders: %d -> %d", len(before), len(after))
	}

	// Built-in templates send main.typ's text.
	w = c.do("POST", "/api/projects", `{"name":"Cờ","mainContent":"= Bài tập\n"}`)
	mustStatus(t, w, 200, "create with content")
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	p, _ = s.projects.Get(created.ID)
	if data, _ := os.ReadFile(filepath.Join(p.Dir, "main.typ")); string(data) != "= Bài tập\n" || created.Main != "main.typ" {
		t.Fatalf("main.typ = %q, main = %q", data, created.Main)
	}

	// README of a package.
	w = c.do("GET", "/api/universe/readme/demo/0.1.0", "")
	mustStatus(t, w, 200, "readme")
	if !json.Valid(w.Body.Bytes()) || string(w.Body.Bytes()) == "" {
		t.Fatalf("readme body %s", w.Body.String())
	}
	mustStatus(t, c.do("GET", "/api/universe/readme/Demo/0.1", ""), 400, "bad spec")
}
