package utils

import (
	"context"
	"os"
	"os/exec"
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
		merged = append(merged, k+"="+v)
	}
	return merged
}
