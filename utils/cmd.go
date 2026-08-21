//go:build !windows
// +build !windows

package utils

import (
	"context"
	"os/exec"
	"runtime"
)

func BuildCmd(ctx context.Context, path string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, path, args...)
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
	cmd := exec.Command(cmdName, arg...)
	return cmd.Run()
}
