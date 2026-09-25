package utils

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	loginEnvOnce sync.Once
	loginEnvMap  map[string]string
)

// LoginShellEnv captures the environment of the user's login shell and
// returns it as a map. It is intended for child processes that expect a
// full user environment (e.g. ACP agents), because GUI launchers like
// Finder provide only a minimal launchd environment.
//
// The capture happens once on first call; subsequent calls return the
// cached result. If the probe fails, an empty map is returned.
func LoginShellEnv() map[string]string {
	loginEnvOnce.Do(func() {
		loginEnvMap = captureLoginShellEnv()
	})
	return loginEnvMap
}

func captureLoginShellEnv() map[string]string {
	env := make(map[string]string)

	// Windows has no POSIX login shell to probe. Worse, if $SHELL happens to
	// be set anyway (e.g. Git Bash/MSYS2, WSL-adjacent tooling installed
	// alongside a normal Windows dev setup), running it here would merge its
	// Unix-style PATH (/c/Program Files/... , ':'-separated) over the real
	// Windows PATH, which breaks npx.cmd's own internal path handling --
	// observed live 2026-09-18: it makes every npx-based ACP agent
	// (including the default Claude Code one) fail to start with
	// `'"node"' is not recognized`. app.go's init() already skips the
	// equivalent PATH-capture step on Windows for the same reason; mirror
	// that guard here so every caller (not just app.go) is covered.
	if runtime.GOOS == "windows" {
		return env
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	out, err := exec.CommandContext(ctx, shell, "-l", "-c", "env").Output()
	if err != nil {
		return env
	}

	for _, line := range strings.Split(string(out), "\n") {
		if k, v, ok := strings.Cut(line, "="); ok {
			env[k] = v
		}
	}

	return env
}

// ParseEnv converts a list of "KEY=VALUE" strings into a map. Malformed
// entries without '=' are skipped.
func ParseEnv(list []string) map[string]string {
	env := make(map[string]string, len(list))
	for _, kv := range list {
		if k, v, ok := strings.Cut(kv, "="); ok {
			env[k] = v
		}
	}
	return env
}

// MergeEnv builds an environment list from base, overlaid by each overlay
// map in order. Keys from later overlays replace earlier values, and keys
// present only in the base are preserved.
func MergeEnv(base []string, overlays ...map[string]string) []string {
	env := ParseEnv(base)
	for _, overlay := range overlays {
		for k, v := range overlay {
			env[k] = v
		}
	}

	merged := make([]string, 0, len(env))
	for k, v := range env {
		if isSecretEnv(k) {
			continue
		}
		merged = append(merged, k+"="+v)
	}
	return merged
}

// secretEnvKeys are variables that configure this server itself and must
// never reach child processes (AI agents, the shell commands they run).
// The login-shell snapshot is taken before main() can unset them, so they
// are filtered here as well.
var secretEnvKeys = map[string]bool{
	"TYPSTIFY_SERVER_PASSWORD": true,
}

func isSecretEnv(key string) bool { return secretEnvKeys[key] }

// ScrubSecretEnv returns env ("KEY=value" entries) without secretEnvKeys.
func ScrubSecretEnv(env []string) []string {
	out := env[:0:0]
	for _, kv := range env {
		k, _, _ := strings.Cut(kv, "=")
		if !isSecretEnv(k) {
			out = append(out, kv)
		}
	}
	return out
}
