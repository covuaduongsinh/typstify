// Command typstify-server is the headless entrypoint for Typstify's self-hosted
// web mode. It boots the same backend (service.ServiceFacade, LSP client, typst
// compiler wrapper, ACP agent session manager) used by the desktop app, without
// importing gioui.org/app or any other Gio UI package, and serves the HTTP/
// WebSocket API from package server (docs/plans/plan_web_version.md, Giai
// doan 1) plus the built web frontend, if present.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"looz.ws/typstify/logger"
	"looz.ws/typstify/server"
	"looz.ws/typstify/service"
	"looz.ws/typstify/utils"
	"looz.ws/typstify/version"
)

func main() {
	projectDirFlag := flag.String("project", "", "Path to the Typst project directory to open (defaults to $TYPSTIFY_PROJECT_DIR, then the current working directory)")
	addrFlag := flag.String("addr", envOr("TYPSTIFY_SERVER_ADDR", ":8080"), "Address to listen on")
	passwordFlag := flag.String("password", os.Getenv("TYPSTIFY_SERVER_PASSWORD"), "Password required to use the server. Empty disables auth (loopback/dev use only)")
	staticDirFlag := flag.String("static-dir", envOr("TYPSTIFY_STATIC_DIR", "web/dist"), "Directory containing the built web frontend to serve; ignored if it doesn't exist")
	flag.Parse()

	projectDir, err := resolveProjectDir(*projectDirFlag)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	appSrv := service.NewService(ctx)

	logger.InitLogger(filepath.Join(appSrv.Settings().General().RootDir, "application.log"))
	defer logger.AppLogger.Close()

	appSrv.SetProjectDir(projectDir)

	if *passwordFlag == "" {
		log.Println("WARNING: no password configured (TYPSTIFY_SERVER_PASSWORD/-password) -- the server is running WITHOUT authentication. Only safe for loopback/dev use, never expose this on a network like this.")
	}

	staticDir := *staticDirFlag
	if _, statErr := os.Stat(filepath.Join(staticDir, "index.html")); statErr != nil {
		log.Printf("web frontend not found at %s, serving API only (see docs/plans/plan_web_version.md Giai doan 2)", staticDir)
		staticDir = ""
	}

	httpSrv := &http.Server{
		Addr: *addrFlag,
		Handler: server.New(appSrv, server.Options{
			Password:  *passwordFlag,
			StaticDir: staticDir,
		}).Handler(),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		log.Println("shutting down...")
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	log.Printf("Typstify web server starting (version %s)", version.BinVersion)
	log.Printf("project directory: %s", projectDir)
	log.Printf("listening on %s", *addrFlag)

	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	appSrv.Close(closeCtx)
	log.Println("service down")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// resolveProjectDir picks the project directory from, in order of priority,
// the -project flag, the TYPSTIFY_PROJECT_DIR env var, or the current working
// directory, then validates it exists and is a directory.
func resolveProjectDir(flagValue string) (string, error) {
	dir := flagValue
	if dir == "" {
		dir = os.Getenv("TYPSTIFY_PROJECT_DIR")
	}
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = cwd
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", os.ErrInvalid
	}

	return absDir, nil
}

func init() {
	if runtime.GOOS == "windows" {
		return
	}

	// Mirrors app.go's init(): GUI/service launchers (systemd, Docker ENTRYPOINT,
	// macOS Finder/launchd) often provide a minimal environment, so capture the
	// user's login-shell PATH once and apply it to child processes (typst,
	// tinymist, npx-based AI agents) that need it.
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
