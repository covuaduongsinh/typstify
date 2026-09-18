package service

import (
	"context"
	"sync"

	"looz.ws/typstify/service/settings"
)

// WindowService tracks application shutdown so that windows -- owned and
// created by the Gio desktop UI layer (see ui/windowview.go) -- can
// coordinate on it. It intentionally has no Gio dependency of its own, so
// package service (and therefore cmd/typstify-server, the headless web-mode
// entrypoint) can be built without pulling in gioui.org.
type WindowService struct {
	settings *settings.Settings
	// Context is used to broadcast application shutdown.
	Context context.Context
	// Shutdown shuts down all windows.
	Shutdown func()
	// active keeps track the open windows, such that application
	// can shut down, when all of them are closed.
	active sync.WaitGroup
}

func NewWindowService(ctx context.Context, settings *settings.Settings) *WindowService {
	ctx, cancel := context.WithCancel(ctx)
	return &WindowService{
		Context:  ctx,
		Shutdown: cancel,
		settings: settings,
	}
}

// Wait waits for all windows to close.
func (w *WindowService) Wait() {
	w.active.Wait()
}

// TrackWindow runs run (a window's event loop) in a tracked goroutine, so
// Wait blocks until every tracked window has returned.
func (w *WindowService) TrackWindow(run func()) {
	w.active.Add(1)
	go func() {
		defer w.active.Done()
		run()
	}()
}

func (w *WindowService) Settings() *settings.Settings {
	return w.settings
}
