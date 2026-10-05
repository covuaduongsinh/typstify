package navpanel

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"path/filepath"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"looz.ws/typstify/ui/filechooser"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"

	// "golang.org/x/exp/shiny/materialdesign/icons"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/service"
	"looz.ws/typstify/service/bus"
	"looz.ws/typstify/ui/dialog"
	"looz.ws/typstify/ui/pkgmgmt"
	"looz.ws/typstify/ui/settings"
	"looz.ws/typstify/ui/statusbar"
	"looz.ws/typstify/ui/uitokens"
	wg "looz.ws/typstify/widgets"
	"looz.ws/typstify/widgets/icons"
)

var (
	openFolder     = icons.NewSvgIcon(icons.FolderOpen)
	newFolder      = icons.NewSvgIcon(icons.FolderPlus)
	historyIcon    = icons.NewSvgIcon(icons.History)
	pkgManagerIcon = icons.NewSvgIcon(icons.PackageOpen)
	settingsIcon   = icons.NewSvgIcon(icons.Cog)
	panelHideIcon  = icons.NewSvgIcon(icons.PanelLeftClose)
	panelShowIcon  = icons.NewSvgIcon(icons.PanelRightClose)
	cloudIcon      = icons.NewSvgIcon(icons.Cloud)
	cloudUpIcon    = icons.NewSvgIcon(icons.CloudUpload)
	cloudOkIcon    = icons.NewSvgIcon(icons.CloudCheck)
	cloudErrIcon   = icons.NewSvgIcon(icons.CloudAlert)
)

type MenuPanel struct {
	openDirBtn        widget.Clickable
	openDirTip        wg.TipArea
	openPkgManagerBtn widget.Clickable
	openPkgManagerTip wg.TipArea
	newProjectBtn     widget.Clickable
	newProjectTip     wg.TipArea
	openSettingBtn    widget.Clickable
	openSettingTip    wg.TipArea
	vpsSyncBtn        widget.Clickable
	vpsSyncTip        wg.TipArea
	hideDrawerBtn     widget.Clickable
	hideDrawerTip     wg.TipArea

	IsDrawerHidden bool
	vm             view.ViewManager
	srv            *service.ServiceFacade
}

func (cp *MenuPanel) Layout(gtx C, th *theme.Theme) D {
	cp.update(gtx)

	return layout.Inset{
		Left:   unit.Dp(8),
		Top:    unit.Dp(4),
		Bottom: unit.Dp(4),
	}.Layout(gtx, func(gtx C) D {
		return layout.Flex{
			Axis:    layout.Horizontal,
			Spacing: layout.SpaceEnd,
			Gap:     gtx.Dp(unit.Dp(16)),
		}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				btn := wg.TipIconButton(th, &cp.hideDrawerTip, i18n.Translate("Hide Explorer"))

				return btn.Layout(gtx, func(gtx C) D {
					icon := panelHideIcon
					if cp.IsDrawerHidden {
						icon = panelShowIcon
					}
					return cp.layoutBtn(gtx, th, &cp.hideDrawerBtn, icon)
				})
			}),
			layout.Rigid(func(gtx C) D {
				btn := wg.TipIconButton(th, &cp.openDirTip, i18n.Translate("Open Folder"))

				return btn.Layout(gtx, func(gtx C) D {
					return cp.layoutBtn(gtx, th, &cp.openDirBtn, openFolder)
				})
			}),

			layout.Rigid(func(gtx C) D {
				btn := wg.TipIconButton(th, &cp.newProjectTip, i18n.Translate("New Project"))

				return btn.Layout(gtx, func(gtx C) D {
					return cp.layoutBtn(gtx, th, &cp.newProjectBtn, newFolder)
				})
			}),

			layout.Rigid(func(gtx C) D {
				btn := wg.TipIconButton(th, &cp.openPkgManagerTip, i18n.Translate("Typst Package Center"))
				return btn.Layout(gtx, func(gtx C) D {
					return cp.layoutBtn(gtx, th, &cp.openPkgManagerBtn, pkgManagerIcon)
				})
			}),

			layout.Rigid(func(gtx C) D {
				btn := wg.TipIconButton(th, &cp.vpsSyncTip, cp.vpsSyncTooltip())
				return btn.Layout(gtx, func(gtx C) D {
					icon, col := cp.vpsSyncIcon(th)
					return cp.layoutBtnColor(gtx, &cp.vpsSyncBtn, icon, col)
				})
			}),

			layout.Rigid(func(gtx C) D {
				btn := wg.TipIconButton(th, &cp.openSettingTip, i18n.Translate("Settings"))
				return btn.Layout(gtx, func(gtx C) D {
					return cp.layoutBtn(gtx, th, &cp.openSettingBtn, settingsIcon)
				})
			}),
		)
	})
}

func (cp *MenuPanel) layoutBtn(gtx C, th *theme.Theme, btn *widget.Clickable, icon *icons.SvgIcon) D {
	return cp.layoutBtnColor(gtx, btn, icon, th.Fg)
}

func (cp *MenuPanel) layoutBtnColor(gtx C, btn *widget.Clickable, icon *icons.SvgIcon, col color.NRGBA) D {
	return btn.Layout(gtx, func(gtx C) D {
		return layout.UniformInset(unit.Dp(2)).Layout(gtx, func(gtx C) D {
			return icon.Layout(gtx, col, unit.Sp(16))
		})
	})
}

// vpsSyncConfigured báo đã nhập Server URL và bật đồng bộ VPS.
func (cp *MenuPanel) vpsSyncConfigured() bool {
	cfg := cp.srv.Settings().VPSSync()
	return cfg.Enabled && cfg.ServerURL != ""
}

// vpsSyncIcon chọn icon và màu theo trạng thái đồng bộ hiện tại.
func (cp *MenuPanel) vpsSyncIcon(th *theme.Theme) (*icons.SvgIcon, color.NRGBA) {
	if !cp.vpsSyncConfigured() {
		return cloudIcon, th.Fg
	}
	st := cp.srv.VPSSync().Status()
	switch {
	case st.Syncing:
		return cloudUpIcon, uitokens.InfoColor(th)
	case st.LastError != "":
		return cloudErrIcon, uitokens.ErrorColor(th)
	case !st.LastSyncTime.IsZero():
		return cloudOkIcon, uitokens.SuccessColor(th)
	default:
		return cloudIcon, th.Fg
	}
}

func (cp *MenuPanel) vpsSyncTooltip() string {
	if !cp.vpsSyncConfigured() {
		return i18n.Translate("VPS Sync: not configured")
	}
	st := cp.srv.VPSSync().Status()
	switch {
	case st.Syncing:
		return i18n.Translate("VPS Sync: syncing...")
	case st.LastError != "":
		return i18n.Translate("VPS Sync error: ") + st.LastError
	default:
		return i18n.Translate("VPS Sync: click to sync now")
	}
}

// startVPSSync chạy đồng bộ thủ công trong nền và làm mới cửa sổ khi xong.
func (cp *MenuPanel) startVPSSync() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		cp.srv.RefreshWindow()
		if _, err := cp.srv.VPSSync().PerformSync(ctx); err != nil {
			log.Println("vps sync failed: ", err)
			cp.srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{
				Content:  fmt.Sprintf(i18n.Translate("VPS Sync failed: %v"), err),
				Level:    2,
				Duration: 15 * time.Second,
			})
		}
		cp.srv.RefreshWindow()
	}()
}

func (cp *MenuPanel) update(gtx C) {
	cp.openDirTip.Direction = layout.E
	cp.newProjectTip.Direction = layout.E
	cp.openPkgManagerTip.Direction = layout.E
	cp.openSettingTip.Direction = layout.E
	cp.hideDrawerTip.Direction = layout.E

	if cp.openSettingBtn.Clicked(gtx) {
		cp.vm.RequestSwitch(view.Intent{
			Target:     settings.SettingViewID,
			RequireNew: true,
		})
	}

	if cp.newProjectBtn.Clicked(gtx) {
		cp.vm.RequestSwitch(view.Intent{
			Target:      dialog.CreateProjectDialogViewID,
			ShowAsModal: true,
		})
	}

	if cp.openDirBtn.Clicked(gtx) {
		go func() {
			projectDir, err := cp.srv.FileChooser().(*filechooser.FileChooser).ChooseFolder()
			if err != nil {
				log.Println("failed to choose folder: ", projectDir, err)
				cp.srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{
					Content:  fmt.Sprintf(i18n.Translate("Choose folder failed: %v"), err),
					Level:    1,
					Duration: 15 * time.Second,
				})
				return
			}
			if isFile(projectDir) {
				projectDir = filepath.Dir(projectDir)
			}

			log.Println("choosed folder: ", projectDir)
			cp.srv.EventBus().Emit(bus.TopicProjectSwitched, projectDir)
		}()
	}

	if cp.openPkgManagerBtn.Clicked(gtx) {
		cp.vm.RequestSwitch(view.Intent{
			Target:     pkgmgmt.PkgListViewID,
			RequireNew: true,
		})
	}

	if cp.vpsSyncBtn.Clicked(gtx) {
		if !cp.vpsSyncConfigured() {
			cp.vm.RequestSwitch(view.Intent{
				Target:     settings.SettingViewID,
				RequireNew: true,
				Params:     map[string]any{"tabIdx": settings.VPSSyncTabIdx},
			})
		} else if !cp.srv.VPSSync().Status().Syncing {
			cp.startVPSSync()
		}
	}

	if cp.hideDrawerBtn.Clicked(gtx) {
		cp.IsDrawerHidden = !cp.IsDrawerHidden
	}
}

func NewMenuPanel(vm view.ViewManager, srv *service.ServiceFacade) *MenuPanel {
	return &MenuPanel{
		vm:  vm,
		srv: srv,
	}
}
