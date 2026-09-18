package utils

import (
	"context"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func BuildCmd(ctx context.Context, path string, args ...string) *exec.Cmd {
	resolved := LookupExecutable(path)
	cmd := exec.CommandContext(ctx, resolved, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	return cmd
}

// explorer command returns non-zero even if it is successful.
// So we migrate to the shell API here.
func OpenInExternalApp(path string) error {
	verbPtr, _ := windows.UTF16PtrFromString("open")
	pathPtr, _ := windows.UTF16PtrFromString(path)

	return windows.ShellExecute(0, verbPtr, pathPtr, nil, nil, windows.SW_SHOWNORMAL)
}
