package settings

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/dustin/go-humanize"
	"github.com/oligo/gioview/explorer"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/service"
	"looz.ws/typstify/service/bus"
	"looz.ws/typstify/service/fonts"
	"looz.ws/typstify/ui/statusbar"
)

// FontsSettingsView lets the user list, add, and remove extra font files
// (.ttf/.otf/.ttc) used by the Typst compiler/LSP/preview. It is a desktop
// UI for the same managed directory (service/fonts.Dir, keyed off
// TypstSettings.ExtraFontPath) that the web server's font API
// (server/fonts_api.go) already exposes over HTTP.
type FontsSettingsView struct {
	srv *service.ServiceFacade

	list       widget.List
	fontsList  atomic.Pointer[[]fonts.Info]
	loading    atomic.Bool
	lastErr    error
	deleteBtns []widget.Clickable
	addBtn     widget.Clickable
	busy       atomic.Bool
}

func NewFontsSettingsView(srv *service.ServiceFacade) *FontsSettingsView {
	return &FontsSettingsView{
		srv:  srv,
		list: widget.List{List: layout.List{Axis: layout.Vertical}},
	}
}

func (f *FontsSettingsView) Title() string { return i18n.Translate("Fonts") }

func (f *FontsSettingsView) Layout(gtx C, th *theme.Theme) D {
	f.ensureLoaded()
	f.update(gtx)

	list := f.fontsList.Load()

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					label := material.Subtitle1(th.Theme, i18n.Translate("Fonts"))
					return label.Layout(gtx)
				}),
				layout.Flexed(1, func(gtx C) D { return D{} }),
				layout.Rigid(func(gtx C) D {
					btn := material.Button(th.Theme, &f.addBtn, i18n.Translate("Add font..."))
					btn.Inset = layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8)}
					return btn.Layout(gtx)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx C) D {
			if f.lastErr != nil {
				label := material.Label(th.Theme, th.TextSize, f.lastErr.Error())
				label.Color = misc.WithAlpha(th.Fg, 0x90)
				return label.Layout(gtx)
			}
			return D{}
		}),
		layout.Rigid(func(gtx C) D {
			if list == nil || len(*list) == 0 {
				label := material.Label(th.Theme, th.TextSize, i18n.Translate("No extra fonts installed."))
				label.Color = misc.WithAlpha(th.Fg, 0x60)
				return label.Layout(gtx)
			}

			items := *list
			for len(f.deleteBtns) < len(items) {
				f.deleteBtns = append(f.deleteBtns, widget.Clickable{})
			}

			gtx.Constraints.Max.Y = min(gtx.Constraints.Max.Y, gtx.Dp(unit.Dp(280)))
			l := material.List(th.Theme, &f.list)
			return l.Layout(gtx, len(items), func(gtx C, index int) D {
				return f.layoutRow(gtx, th, items[index], index)
			})
		}),
	)
}

func (f *FontsSettingsView) layoutRow(gtx C, th *theme.Theme, item fonts.Info, index int) D {
	return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx C) D {
				label := material.Label(th.Theme, th.TextSize, item.Name)
				return label.Layout(gtx)
			}),
			layout.Rigid(func(gtx C) D {
				label := material.Label(th.Theme, th.TextSize*0.85, humanize.Bytes(uint64(item.Size)))
				label.Color = misc.WithAlpha(th.Fg, 0xb0)
				return label.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
			layout.Rigid(func(gtx C) D {
				btn := material.Button(th.Theme, &f.deleteBtns[index], i18n.Translate("Remove"))
				btn.Inset = layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}
				return btn.Layout(gtx)
			}),
		)
	})
}

func (f *FontsSettingsView) update(gtx C) {
	if f.addBtn.Clicked(gtx) {
		f.addFonts()
	}
	for i := range f.deleteBtns {
		if f.deleteBtns[i].Clicked(gtx) {
			if list := f.fontsList.Load(); list != nil && i < len(*list) {
				f.deleteFont((*list)[i].Name)
			}
		}
	}
}

func (f *FontsSettingsView) ensureLoaded() {
	if f.fontsList.Load() != nil {
		return
	}
	if f.loading.CompareAndSwap(false, true) {
		go f.refresh()
	}
}

func (f *FontsSettingsView) refresh() {
	list, err := fonts.List(f.srv.Settings())
	f.lastErr = err
	f.fontsList.Store(&list)
	f.srv.RefreshWindow()
}

func (f *FontsSettingsView) addFonts() {
	if !f.busy.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer f.busy.Store(false)

		chooser, ok := f.srv.FileChooser().(*explorer.FileChooser)
		if !ok {
			log.Println("fonts settings: FileChooser is not available")
			f.srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{
				Content:  i18n.Translate("File chooser is not available"),
				Level:    2,
				Duration: 15 * time.Second,
			})
			return
		}
		readers, err := chooser.ChooseFiles(".ttf", ".otf", ".ttc")
		if err != nil {
			log.Println("fonts settings: choose files failed:", err)
			f.srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{
				Content:  fmt.Sprintf(i18n.Translate("Choose font files failed: %v"), err),
				Level:    1,
				Duration: 15 * time.Second,
			})
			return
		}

		for _, rc := range readers {
			name := ""
			if osFile, ok := rc.(*os.File); ok {
				name = filepath.Base(osFile.Name())
			}
			if name == "" {
				log.Println("fonts settings: could not determine chosen file name, skipping")
				rc.Close()
				continue
			}
			if err := fonts.Upload(f.srv.Settings(), name, rc); err != nil {
				log.Println("fonts settings: upload failed:", err)
				f.srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{
					Content:  fmt.Sprintf(i18n.Translate("Upload font %s failed: %v"), name, err),
					Level:    2,
					Duration: 15 * time.Second,
				})
			}
			rc.Close()
		}

		f.refresh()
	}()
}

func (f *FontsSettingsView) deleteFont(name string) {
	go func() {
		if err := fonts.Delete(f.srv.Settings(), name); err != nil {
			log.Println("fonts settings: delete failed:", err)
			f.srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{
				Content:  fmt.Sprintf(i18n.Translate("Delete font %s failed: %v"), name, err),
				Level:    2,
				Duration: 15 * time.Second,
			})
		}
		f.refresh()
	}()
}
