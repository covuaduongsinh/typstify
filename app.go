package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gioui.org/app"
	//"github.com/pkg/profile"

	"looz.ws/typstify/logger"
	"looz.ws/typstify/service"
	"looz.ws/typstify/ui"
	"looz.ws/typstify/utils"
)

func main() {
	// log.SetFlags(log.Default().Flags() | log.Llongfile)
	//defer profile.Start(profile.CPUProfile).Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := service.NewService(ctx)
	// init logger
	logger.InitLogger(filepath.Join(srv.Settings().General().RootDir, "application.log"))
	defer logger.AppLogger.Close()

	ui := ui.NewUI(srv, false)

	go func() {
		err := ui.Loop(ctx)
		if err != nil {
			log.Println(err)
		}
		srv.Close(ctx)
		os.Exit(0)
	}()

	app.Main()
}

func init() {
	if runtime.GOOS == "windows" {
		return
	}

	// In MacOS, GUI launchers like Finder provide a minimal launchd environment.
	// Capture the user's login-shell environment once, so it can be applied
	// to child processes that need it (see agent.SessionManager.Start).
	// Globally, only set PATH, so binaries like npx are locatable via
	// exec.LookPath without exposing the full shell env to other children.
	env := utils.LoginShellEnv()
	if path := env["PATH"]; path != "" {
		os.Setenv("PATH", path)
	} else {
		os.Setenv("PATH", fallbackPath(os.Getenv("PATH")))
	}
}

func fallbackPath(currentPath string) string {
	home, _ := os.UserHomeDir()
	fallbacks := []string{
		"/opt/homebrew/bin", // for macOS
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
		filepath.Join(home, ".local/bin"), // standard Linux user-bin
		currentPath,
	}
	return strings.Join(fallbacks, ":")
}
