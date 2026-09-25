package server

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const loginConsole = `Please visit https://accounts.google.com/o/oauth2/v2/auth?client_id=x&redirect_uri=http%3A%2F%2F127.0.0.1%3A44479&response_type=code&state=abc to log in`

func TestAllowedCallbackTarget(t *testing.T) {
	ok, err := allowedCallbackTarget(loginConsole, "http://127.0.0.1:44479/?state=abc&code=4/0AX&iss=https://accounts.google.com")
	if err != nil || ok.Host != "127.0.0.1:44479" || ok.Query().Get("code") != "4/0AX" {
		t.Fatalf("valid callback rejected: %v %v", ok, err)
	}
	if u, err := allowedCallbackTarget(loginConsole, "http://localhost:44479/?code=1"); err != nil || u.Host != "127.0.0.1:44479" {
		t.Fatalf("localhost callback: %v %v", u, err)
	}
	// An agent listening on IPv6 loopback is reached there, not on 127.0.0.1.
	v6 := "visit https://accounts.google.com/auth?redirect_uri=http%3A%2F%2F%5B%3A%3A1%5D%3A5555&x=1"
	if u, err := allowedCallbackTarget(v6, "http://[::1]:5555/?code=1"); err != nil || u.Host != "[::1]:5555" {
		t.Fatalf("IPv6 callback: %v %v", u, err)
	}

	cases := []struct {
		name, console, raw string
		want               error
	}{
		{"other port", loginConsole, "http://127.0.0.1:8080/?code=1", errCallbackPort},
		{"non-loopback host", loginConsole, "http://evil.example:44479/?code=1", errCallbackFormat},
		{"https", loginConsole, "https://127.0.0.1:44479/?code=1", errCallbackFormat},
		{"no port", loginConsole, "http://127.0.0.1/?code=1", errCallbackFormat},
		{"no code", loginConsole, "http://127.0.0.1:44479/?state=abc", errCallbackNoCode},
		{"no login link", "nothing here", "http://127.0.0.1:44479/?code=1", errCallbackNoLogin},
		{"garbage", loginConsole, "not a url", errCallbackFormat},
	}
	for _, c := range cases {
		if _, err := allowedCallbackTarget(c.console, c.raw); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}
}

func callbackRequest(t *testing.T, s *Server, raw string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(agentAuthCallbackRequest{URL: raw})
	w := httptest.NewRecorder()
	s.handleAgentAuthCallback(w, httptest.NewRequest("POST", "/api/agent/auth/callback", strings.NewReader(string(body))))
	return w
}

func TestAgentAuthCallbackRequiresPendingLogin(t *testing.T) {
	s := &Server{consoleTextFn: func() string { return loginConsole }}
	if w := callbackRequest(t, s, "http://127.0.0.1:44479/?code=1"); w.Code != http.StatusConflict {
		t.Fatalf("no login pending: status %d, want 409", w.Code)
	}
}

func TestAgentAuthCallbackRelaysToAgent(t *testing.T) {
	var gotQuery url.Values
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte("<html><body><h1>Signed in!</h1> You can close this tab.</body></html>"))
	}))
	defer agent.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(agent.URL, "http://"))

	console := "open https://accounts.google.com/auth?redirect_uri=http%3A%2F%2F127.0.0.1%3A" + port + "&x=1"
	s := &Server{consoleTextFn: func() string { return console }}
	s.authInProgress.Add(1)

	w := callbackRequest(t, s, "http://127.0.0.1:"+port+"/?state=abc&code=4/0AX")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if gotQuery.Get("code") != "4/0AX" || gotQuery.Get("state") != "abc" {
		t.Fatalf("agent got query %v", gotQuery)
	}
	if !strings.Contains(w.Body.String(), "Signed in! You can close this tab.") {
		t.Fatalf("response %s", w.Body.String())
	}
}

func TestAgentAuthRoutesRegister(t *testing.T) {
	// The literal /callback route must coexist with /{methodId}.
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/agent/auth/callback", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("POST /api/agent/auth/{methodId}", func(http.ResponseWriter, *http.Request) {})
	_, pattern := mux.Handler(httptest.NewRequest("POST", "/api/agent/auth/callback", nil))
	if pattern != "POST /api/agent/auth/callback" {
		t.Fatalf("callback routed to %q", pattern)
	}
}
