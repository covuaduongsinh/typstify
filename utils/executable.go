package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// LookupExecutable looks up the executable path from the executable dir,
// bin subdirectories, the current process root dir, and existing PATH,
// returning an absolute path to avoid Go 1.19+ exec.ErrDot issues.
func LookupExecutable(exeName string) string {
	if filepath.IsAbs(exeName) {
		return exeName
	}

	binDir := ""
	currentExePath, err := os.Executable()
	if err == nil {
		binDir = filepath.Dir(currentExePath)
	}

	// 1. Check in same directory as current executable
	if binDir != "" {
		exePath := filepath.Join(binDir, exeName)
		if exists, isDir := CheckFileExists(exePath); exists && !isDir {
			if abs, err := filepath.Abs(exePath); err == nil {
				return abs
			}
			return exePath
		}
		pBin := filepath.Join(binDir, "bin", exeName)
		if exists, isDir := CheckFileExists(pBin); exists && !isDir {
			if abs, err := filepath.Abs(pBin); err == nil {
				return abs
			}
			return pBin
		}
	}

	// 2. Check in current working directory
	cwd, err := os.Getwd()
	if err == nil {
		exePath := filepath.Join(cwd, exeName)
		if exists, isDir := CheckFileExists(exePath); exists && !isDir {
			if abs, err := filepath.Abs(exePath); err == nil {
				return abs
			}
			return exePath
		}
		pBin := filepath.Join(cwd, "bin", exeName)
		if exists, isDir := CheckFileExists(pBin); exists && !isDir {
			if abs, err := filepath.Abs(pBin); err == nil {
				return abs
			}
			return pBin
		}
	}

	// 3. Fallback to LookPath
	absPath, err := exec.LookPath(exeName)
	if err == nil {
		if abs, err := filepath.Abs(absPath); err == nil {
			return abs
		}
		return absPath
	}

	log.Printf("No %s found after searching PATH: %s", exeName, os.Getenv("PATH"))
	return exeName
}

func lookupExecutable(exeName string) string {
	return LookupExecutable(exeName)
}

type CmdBuilder struct {
	DefaultArgs []string
	Path        string
}

func (b *CmdBuilder) Check() (string, error) {
	path := b.Path
	if filepath.Base(path) == path {
		path = LookupExecutable(path)
	} else {
		exists, isDir := CheckFileExists(path)
		if !exists || isDir {
			return "", fmt.Errorf("executable not exists: %s", path)
		}
	}

	return path, nil
}

func (b *CmdBuilder) Build(ctx context.Context, args ...string) *exec.Cmd {
	path, err := b.Check()
	if err != nil {
		return nil
	}

	args = append(b.DefaultArgs, args...)
	return BuildCmd(ctx, path, args...)
}
