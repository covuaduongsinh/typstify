package server

import (
	"encoding/json"
	"os"
	"testing"
)

func TestHistoryAPI(t *testing.T) {
	root, err := os.MkdirTemp("", "typstify-history-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	s := New(testService(), Options{ProjectRoot: root, StorageDir: tempStorage(t)})
	h := s.Handler()

	owner := &apiClient{t: t, h: h}
	mustStatus(t, owner.do("POST", "/api/auth/register", `{"username":"owner","password":"pw1234"}`), 200, "register owner")
	ownerCookie := owner.cookie
	mustStatus(t, owner.do("POST", "/api/auth/register", `{"username":"guest","password":"pw1234"}`), 200, "register guest")
	owner.cookie = ownerCookie

	w := owner.do("POST", "/api/projects", `{"name":"Lịch sử"}`)
	mustStatus(t, w, 200, "create project")
	var created projectView
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	mustStatus(t, owner.do("PUT", "/api/projects/"+created.ID+"/members/guest", `{"role":"viewer"}`), 200, "add viewer")
	mustStatus(t, owner.do("POST", "/api/projects/"+created.ID+"/open", ""), 200, "open")
	mustStatus(t, owner.do("PUT", "/api/workspace/file?path=main.typ", "= Bản một"), 200, "write v1")

	mustStatus(t, owner.do("POST", "/api/history", `{"label":""}`), 400, "empty label")
	w = owner.do("POST", "/api/history", `{"label":"Bản gửi nhà in"}`)
	mustStatus(t, w, 200, "create version")
	var v1 struct{ ID, Label, Kind string }
	_ = json.Unmarshal(w.Body.Bytes(), &v1)
	if v1.Kind != "manual" || v1.Label != "Bản gửi nhà in" {
		t.Fatalf("version %+v", v1)
	}

	mustStatus(t, owner.do("PUT", "/api/workspace/file?path=main.typ", "= Bản hai"), 200, "write v2")

	guest := &apiClient{t: t, h: h}
	mustStatus(t, guest.do("POST", "/api/auth/login", `{"username":"guest","password":"pw1234"}`), 200, "guest login")
	mustStatus(t, guest.do("POST", "/api/projects/"+created.ID+"/open", ""), 200, "guest opens")
	w = guest.do("GET", "/api/history/"+v1.ID, "")
	mustStatus(t, w, 200, "viewer reads version")
	var detail struct {
		Changes []struct{ Path, Status string }
	}
	_ = json.Unmarshal(w.Body.Bytes(), &detail)
	if len(detail.Changes) != 1 || detail.Changes[0].Path != "main.typ" || detail.Changes[0].Status != "modified" {
		t.Fatalf("changes %+v", detail.Changes)
	}
	if w := guest.do("GET", "/api/history/"+v1.ID+"/file?path=main.typ", ""); w.Body.String() != "= Bản một" {
		t.Fatalf("old content %q", w.Body.String())
	}
	mustStatus(t, guest.do("POST", "/api/history/"+v1.ID+"/restore", `{}`), 403, "viewer restores")
	mustStatus(t, guest.do("POST", "/api/history", `{"label":"x"}`), 403, "viewer creates")
	mustStatus(t, guest.do("GET", "/api/history/nope", ""), 404, "unknown version")

	// Restoring a file the version did not have removes it; a path that
	// climbs out of the project is resolved inside it.
	mustStatus(t, owner.do("PUT", "/api/workspace/file?path=added.typ", "new"), 200, "write added")
	mustStatus(t, owner.do("POST", "/api/history/"+v1.ID+"/restore", `{"paths":["added.typ","../escape.typ"]}`), 200, "restore removal")
	mustStatus(t, owner.do("GET", "/api/workspace/file?path=added.typ", ""), 404, "added file removed")
	if _, err := os.Stat(root + "/../escape.typ"); err == nil {
		t.Fatal("nothing may be written outside the project")
	}
	mustStatus(t, owner.do("POST", "/api/history/"+v1.ID+"/restore", `{"paths":["main.typ"]}`), 200, "restore file")
	if w := owner.do("GET", "/api/workspace/file?path=main.typ", ""); w.Body.String() != "= Bản một" {
		t.Fatalf("restored content %q", w.Body.String())
	}
	w = owner.do("GET", "/api/history", "")
	var list []struct{ Kind string }
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 3 || list[0].Kind != "restore" {
		t.Fatalf("history after restore %+v", list)
	}
}
