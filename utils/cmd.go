//go:build !windows
// +build !windows

package utils

import (
	"context"
	"os/exec"
	"runtime"
)

func BuildCmd(ctx context.Context, path string, args ...string) *exec.Cmd {
	resolved := LookupExecutable(path)
	return exec.CommandContext(ctx, resolved, args...)
}

func OpenInExternalApp(path string) error {
	switch runtime.GOOS {
	case "darwin", "ios":
		return runCmd("open", path)
	default:
		// linux, unix flavors.
		return runCmd("xdg-open", path)
	}
}

func runCmd(cmdName string, arg ...string) error {
	resolved := LookupExecutable(cmdName)
	cmd := exec.Command(resolved, arg...)
	return cmd.Run()
}
