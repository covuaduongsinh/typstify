package ui

import (
	"context"
	"log"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/oligo/gioview/theme"
	"looz.ws/typstify/fonts"
	"looz.ws/typstify/service"
	"looz.ws/typstify/ui/palette"
)

// This file holds the Gio-dependent window-management helpers that used to
// live in service.WindowService. They were moved here (docs/plans/
// plan_web_version.md, Giai doan 4) so that package service -- and therefore
// cmd/typstify-server, the headless web-mode entrypoint -- has no gioui.org
// dependency and can cross-compile for Linux/Docker. Behavior is unchanged;
// these had no callers elsewhere in the repo at the time of the move.

// Window holds window state.
type Window struct {
	Service *service.WindowService
	*app.Window
}

type WindowView interface {
	// Run handles the window event loop.
	Run(ctx context.Context, w *Window) error
}

// WidgetView allows to use gioview Widget as a view.
type WidgetView func(gtx layout.Context, th *theme.Theme) layout.Dimensions

// Run displays the widget with default handling.
func (view WidgetView) Run(ctx context.Context, w *Window) error {
	var ops op.Ops
	th := LoadTheme(w.Service)

	go func() {
		select {
		case <-w.Service.Context.Done():
			w.Perform(system.ActionClose)
			log.Println("window is closed")
		case <-ctx.Done():
			w.Perform(system.ActionClose)
			log.Println("window is closed")
		}
	}()

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			view(gtx, th)
			e.Frame(gtx.Ops)
		}
	}
}

// NewWindow creates a new tracked window.
func NewWindow(ws *service.WindowService, ctx context.Context, title string, view WindowView, opts ...app.Option) {
	opts = append(opts, app.Title(title))
	ws.TrackWindow(func() {
		w := &Window{
			Service: ws,
			Window:  new(app.Window),
		}
		w.Window.Option(opts...)
		view.Run(ctx, w)
	})
}

func LoadTheme(ws *service.WindowService) *theme.Theme {
	th := theme.NewTheme("", fonts.Embedded, false)

	themeName := ws.Settings().General().Theme
	if themeName == "" {
		themeName = "Default Light"
	}

	cfg, err := palette.ThemeConfig(themeName)
	if err != nil {
		log.Println("Theme query failed: ", err)
		return th
	}

	th.TextSize = unit.Sp(ws.Settings().General().TextSize)
	th = th.WithPalette(cfg.Palette)
	return th
}
