package server

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// defaultBodyLimit caps request bodies for every endpoint except the
	// explicit file save below. JSON payloads here are tiny; the cap exists
	// so an unauthenticated client can't make the server buffer an
	// arbitrarily large login/i18n body.
	defaultBodyLimit = 2 << 20
	// fileBodyLimit matches handleFilePut's historical 64 MiB ceiling.
	fileBodyLimit = 64 << 20
)

// trustedPeer reports whether the direct TCP peer is a reverse proxy we
// can take X-Forwarded-* headers from. The Docker/Dokploy deployments put
// Caddy or Traefik in front of the server on a private network, so the
// peer is loopback or a private address; a client connecting from the
// public internet can never be one, so it can't spoof those headers.
func trustedPeer(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate())
}

// clientIP is the address used for login rate limiting: the first
// X-Forwarded-For hop when the peer is a trusted proxy, else the peer.
func clientIP(r *http.Request) string {
	if trustedPeer(r) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			first, _, _ := strings.Cut(xff, ",")
			if ip := strings.TrimSpace(first); ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// isHTTPS reports whether the browser reached us over TLS, either directly
// or through a TLS-terminating proxy (Caddy/Traefik) that says so.
func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return trustedPeer(r) && strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// sameOrigin reports whether a browser request's Origin header names this
// server. Requests without an Origin (non-browser clients) pass; the auth
// cookie check still applies to them.
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := r.Host
	if trustedPeer(r) {
		if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
			host = fwd
		}
	}
	return strings.EqualFold(u.Host, host)
}

// withBodyLimit caps every request body (see defaultBodyLimit). Reads past
// the cap fail with *http.MaxBytesError, which handlers turn into 413.
func withBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			limit := int64(defaultBodyLimit)
			if r.Method == http.MethodPut && r.URL.Path == "/api/workspace/file" {
				limit = fileBodyLimit
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}

// loginLimiter throttles password guessing per client IP: after
// maxFailures failed attempts inside failureWindow, that IP is locked out
// for lockoutDuration.
type loginLimiter struct {
	mu      sync.Mutex
	entries map[string]*loginAttempts
	now     func() time.Time
}

type loginAttempts struct {
	failures    int
	windowStart time.Time
	lockedUntil time.Time
}

const (
	maxLoginFailures = 5
	failureWindow    = 15 * time.Minute
	lockoutDuration  = 15 * time.Minute
)

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{entries: make(map[string]*loginAttempts), now: time.Now}
}

// allowed reports whether ip may attempt a login now, and if not, how long
// until it may.
func (l *loginLimiter) allowed(ip string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.entries[ip]
	if e == nil {
		return true, 0
	}
	if wait := e.lockedUntil.Sub(l.now()); wait > 0 {
		return false, wait
	}
	return true, 0
}

func (l *loginLimiter) recordFailure(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	e := l.entries[ip]
	if e == nil || now.Sub(e.windowStart) > failureWindow {
		e = &loginAttempts{windowStart: now}
		l.entries[ip] = e
	}
	e.failures++
	if e.failures >= maxLoginFailures {
		e.lockedUntil = now.Add(lockoutDuration)
		e.failures = 0
		e.windowStart = now
	}
}

func (l *loginLimiter) recordSuccess(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, ip)
}

// sweep drops stale entries so the map can't grow without bound. Caller
// holds l.mu.
func (l *loginLimiter) sweep(now time.Time) {
	for ip, e := range l.entries {
		if now.After(e.lockedUntil) && now.Sub(e.windowStart) > failureWindow {
			delete(l.entries, ip)
		}
	}
}
