package settings

import (
	"encoding/json"
	"sync/atomic"
	"time"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	gvwidget "github.com/oligo/gioview/widget"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/service"
	"looz.ws/typstify/service/remote"
	"looz.ws/typstify/service/settings"
)

// syncableSection is implemented by every settings section that
// participates in desktop<->web settings sync (Giai đoạn A) -- see
// settings.RemoteApplier for why this is a separate interface from
// settings.Model, and service/settings/remote.go for why RemoteSettings
// itself is deliberately excluded.
type syncableSection interface {
	settings.Model
	settings.RemoteApplier
}

// SyncView lets the user connect this desktop install to a self-hosted
// Typstify web server and merge settings between the two, per-section,
// using whichever side wrote that section more recently (see
// service/settings.Settings.Meta / RemoteSettings).
type SyncView struct {
	srv *service.ServiceFacade

	serverURLInput    gvwidget.TextField
	passwordInput     gvwidget.TextField
	connectBtn        widget.Clickable
	disconnectBtn     widget.Clickable
	syncBtn           widget.Clickable
	remoteAgentToggle widget.Bool

	connecting            atomic.Bool
	syncing               atomic.Bool
	status                string
	lastErr               error
	remoteAgentToggleInit bool
}

func NewSyncView(srv *service.ServiceFacade) *SyncView {
	return &SyncView{srv: srv}
}

func (sv *SyncView) Title() string { return i18n.Translate("Đồng bộ") }

func (sv *SyncView) Layout(gtx C, th *theme.Theme) D {
	sv.update(gtx)

	remoteSettings := sv.srv.Settings().Remote()

	rows := []layout.FlexChild{
		layout.Rigid(func(gtx C) D {
			label := material.Label(th.Theme, th.TextSize,
				i18n.Translate("Kết nối tới một máy chủ Typstify tự host khác (ví dụ bản web trên VPS của bạn) để đồng bộ cấu hình (giao diện, editor, Typst, LSP) giữa desktop và web."))
			label.LineHeightScale = 1.5
			return label.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
	}

	if sv.lastErr != nil {
		rows = append(rows,
			layout.Rigid(func(gtx C) D { return misc.LayoutErrorLabel(gtx, th, sv.lastErr) }),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		)
	}

	if remoteSettings.Enabled && remoteSettings.Token != "" {
		rows = append(rows, sv.layoutConnected(th, remoteSettings)...)
	} else {
		rows = append(rows, sv.layoutDisconnected(th)...)
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

func (sv *SyncView) layoutDisconnected(th *theme.Theme) []layout.FlexChild {
	return []layout.FlexChild{
		layout.Rigid(func(gtx C) D {
			sv.serverURLInput.SingleLine = true
			sv.serverURLInput.Alignment = text.Start
			return sv.serverURLInput.Layout(gtx, th, i18n.Translate("Địa chỉ máy chủ, vd: https://typstify.example.com"))
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx C) D {
			sv.passwordInput.SingleLine = true
			sv.passwordInput.Alignment = text.Start
			sv.passwordInput.Mask = '*'
			return sv.passwordInput.Layout(gtx, th, i18n.Translate("Mật khẩu"))
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx C) D {
			btn := material.Button(th.Theme, &sv.connectBtn, i18n.Translate("Kết nối"))
			if sv.connecting.Load() {
				btn.Text = i18n.Translate("Đang kết nối...")
			}
			return btn.Layout(gtx)
		}),
	}
}

func (sv *SyncView) layoutConnected(th *theme.Theme, rs *settings.RemoteSettings) []layout.FlexChild {
	return []layout.FlexChild{
		layout.Rigid(func(gtx C) D {
			label := material.Label(th.Theme, th.TextSize, i18n.Translate("Đã kết nối tới ")+rs.ServerURL)
			return label.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
		layout.Rigid(func(gtx C) D {
			if sv.status == "" {
				return D{}
			}
			label := material.Label(th.Theme, th.TextSize*0.9, sv.status)
			label.Color = misc.WithAlpha(th.Fg, 0xb0)
			return label.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					btn := material.Button(th.Theme, &sv.syncBtn, i18n.Translate("Đồng bộ ngay"))
					if sv.syncing.Load() {
						btn.Text = i18n.Translate("Đang đồng bộ...")
					}
					return btn.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(func(gtx C) D {
					return material.Button(th.Theme, &sv.disconnectBtn, i18n.Translate("Ngắt kết nối")).Layout(gtx)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
		layout.Rigid(func(gtx C) D {
			if !sv.remoteAgentToggleInit {
				sv.remoteAgentToggle.Value = rs.RemoteAgentEnabled
				sv.remoteAgentToggleInit = true
			}
			return material.Switch(th.Theme, &sv.remoteAgentToggle,
				i18n.Translate("Dùng AI Agent trên máy chủ này thay vì chạy cục bộ")).Layout(gtx)
		}),
	}
}

func (sv *SyncView) update(gtx C) {
	if sv.connectBtn.Clicked(gtx) {
		sv.connect()
	}
	if sv.disconnectBtn.Clicked(gtx) {
		sv.disconnect()
	}
	if sv.syncBtn.Clicked(gtx) {
		sv.syncNow()
	}
	if sv.remoteAgentToggleInit && sv.remoteAgentToggle.Update(gtx) {
		rs := sv.srv.Settings().Remote()
		rs.RemoteAgentEnabled = sv.remoteAgentToggle.Value
		sv.lastErr = rs.Save()
	}
}

func (sv *SyncView) connect() {
	if !sv.connecting.CompareAndSwap(false, true) {
		return
	}
	serverURL := sv.serverURLInput.Text()
	password := sv.passwordInput.Text()

	go func() {
		defer sv.connecting.Store(false)

		client := remote.NewClient(serverURL, "")
		token, err := client.Login("", password, "typstify-desktop")
		if err != nil {
			sv.lastErr = err
			sv.srv.RefreshWindow()
			return
		}

		rs := sv.srv.Settings().Remote()
		rs.ServerURL = serverURL
		rs.Token = token
		rs.Enabled = true
		if err := rs.Save(); err != nil {
			sv.lastErr = err
			sv.srv.RefreshWindow()
			return
		}

		sv.lastErr = nil
		sv.passwordInput.SetText("")
		sv.status = i18n.Translate("Kết nối thành công.")
		sv.srv.RefreshWindow()
	}()
}

func (sv *SyncView) disconnect() {
	rs := sv.srv.Settings().Remote()
	rs.Token = ""
	rs.Enabled = false
	rs.RemoteAgentEnabled = false
	sv.lastErr = rs.Save()
	sv.status = ""
	sv.remoteAgentToggleInit = false
}

func (sv *SyncView) syncNow() {
	if !sv.syncing.CompareAndSwap(false, true) {
		return
	}

	go func() {
		defer sv.syncing.Store(false)

		rs := sv.srv.Settings().Remote()
		client := remote.NewClient(rs.ServerURL, rs.Token)

		remoteMeta, err := client.GetSettingsMeta()
		if err != nil {
			sv.lastErr = err
			sv.srv.RefreshWindow()
			return
		}
		localMeta := sv.srv.Settings().Meta()

		errs := []error{}
		errs = append(errs, syncSection(client, "general", sv.srv.Settings().General, localMeta["general"], remoteMeta["general"]))
		errs = append(errs, syncSection(client, "editor", sv.srv.Settings().Editor, localMeta["editor"], remoteMeta["editor"]))
		errs = append(errs, syncSection(client, "typst", sv.srv.Settings().Typst, localMeta["typst"], remoteMeta["typst"]))
		errs = append(errs, syncSection(client, "lsp", sv.srv.Settings().Lsp, localMeta["lsp"], remoteMeta["lsp"]))

		sv.lastErr = nil
		for _, err := range errs {
			if err != nil {
				sv.lastErr = err
				break
			}
		}
		if sv.lastErr == nil {
			sv.status = i18n.Translate("Đồng bộ xong lúc ") + time.Now().Format("15:04:05")
		}
		sv.srv.RefreshWindow()
	}()
}

// syncSection reconciles one settings section against its remote
// counterpart: whichever side wrote it more recently wins. get is the
// section's Settings accessor (e.g. Settings.General), reloading current
// local state on every call.
func syncSection[T syncableSection](client *remote.Client, name string, get func() T, localTS, remoteTS time.Time) error {
	if remoteTS.After(localTS) {
		var raw json.RawMessage
		if err := client.GetSettings(name, &raw); err != nil {
			return err
		}
		return get().ApplyRemote(raw, remoteTS)
	}
	if localTS.After(remoteTS) {
		return client.PutSettings(name, get())
	}
	return nil
}
