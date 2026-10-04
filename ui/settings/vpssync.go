package settings

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/theme"
	gvwidget "github.com/oligo/gioview/widget"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/service"
	"looz.ws/typstify/service/vpssync"
)

// VPSSyncTabIdx là chỉ số tab VPS Sync trong NewSettingsView (đặt sau Agent).
const VPSSyncTabIdx = 8

// VPSSyncView cấu hình đồng bộ file lên typstify-server (VPS). Trạng thái
// và nút Sync Now nằm ở thanh menu (ui/navpanel).
type VPSSyncView struct {
	srv         *service.ServiceFacade
	serverInput gvwidget.TextField
	tokenInput  gvwidget.TextField
	enabled     widget.Bool
	autoSync    widget.Bool
	testBtn     widget.Clickable
	saveBtn     widget.Clickable
	testing     atomic.Bool
	once        sync.Once

	mu     sync.Mutex
	msg    string
	msgErr bool
}

func NewVPSSyncView(srv *service.ServiceFacade) *VPSSyncView {
	return &VPSSyncView{srv: srv}
}

func (v *VPSSyncView) Title() string {
	return i18n.Translate("VPS Sync")
}

func (v *VPSSyncView) setMsg(msg string, isErr bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.msg = msg
	v.msgErr = isErr
}

func (v *VPSSyncView) message() (string, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.msg, v.msgErr
}

// loadFromSettings nạp giá trị đã lưu vào các ô nhập, chỉ làm một lần khi mở view.
func (v *VPSSyncView) loadFromSettings() {
	cfg := v.srv.Settings().VPSSync()
	v.serverInput.SetText(cfg.ServerURL)
	v.tokenInput.SetText(cfg.Token)
	v.enabled.Value = cfg.Enabled
	v.autoSync.Value = cfg.AutoSync
}

// save ghi các ô nhập vào settings. Test Connection cũng gọi hàm này trước,
// vì engine đọc cấu hình đã lưu chứ không đọc ô nhập.
func (v *VPSSyncView) save() error {
	cfg := v.srv.Settings().VPSSync()
	cfg.ServerURL = strings.TrimSpace(v.serverInput.Text())
	cfg.Token = strings.TrimSpace(v.tokenInput.Text())
	cfg.Enabled = v.enabled.Value
	cfg.AutoSync = v.autoSync.Value
	return cfg.Save()
}

func (v *VPSSyncView) update(gtx C) {
	v.once.Do(v.loadFromSettings)

	if v.saveBtn.Clicked(gtx) {
		if err := v.save(); err != nil {
			v.setMsg(err.Error(), true)
		} else {
			v.setMsg(i18n.Translate("Saved"), false)
		}
	}

	if v.testBtn.Clicked(gtx) && v.testing.CompareAndSwap(false, true) {
		if err := v.save(); err != nil {
			v.setMsg(err.Error(), true)
			v.testing.Store(false)
			return
		}
		v.setMsg(i18n.Translate("Testing connection..."), false)
		go func() {
			defer v.testing.Store(false)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			err := v.srv.VPSSync().TestConnection(ctx)
			if err != nil {
				// Thử tự động login nếu tokenInput là password
				serverURL := strings.TrimSpace(v.serverInput.Text())
				passOrToken := strings.TrimSpace(v.tokenInput.Text())
				if passOrToken != "" && serverURL != "" {
					if tok, loginErr := vpssync.LoginWithPassword(ctx, serverURL, "", passOrToken); loginErr == nil && tok != "" {
						v.tokenInput.SetText(tok)
						_ = v.save()
						if testErr := v.srv.VPSSync().TestConnection(ctx); testErr == nil {
							v.setMsg(i18n.Translate("Login successful & Connection OK"), false)
							v.srv.RefreshWindow()
							return
						}
					}
				}
				v.setMsg(fmt.Sprintf("%s: %v", i18n.Translate("Connection failed"), err), true)
			} else {
				v.setMsg(i18n.Translate("Connection OK"), false)
			}
			v.srv.RefreshWindow()
		}()
	}
}

func (v *VPSSyncView) Layout(gtx C, th *theme.Theme) D {
	v.update(gtx)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			label := material.Label(th.Theme, th.TextSize, i18n.Translate("Upload project files from this computer to a typstify server (VPS). Token is the Bearer token issued by the server."))
			label.LineHeightScale = 1.5
			return label.Layout(gtx)
		}),

		layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),

		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx C) D {
					v.serverInput.Alignment = text.Start
					v.serverInput.SingleLine = true
					return v.serverInput.Layout(gtx, th, "Server URL, e.g. https://typst.example.com")
				}),
			)
		}),

		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),

		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx C) D {
					v.tokenInput.Alignment = text.Start
					v.tokenInput.SingleLine = true
					return v.tokenInput.Layout(gtx, th, "Token")
				}),
			)
		}),

		layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),

		layout.Rigid(material.CheckBox(th.Theme, &v.enabled, i18n.Translate("Enable VPS sync")).Layout),

		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),

		layout.Rigid(material.CheckBox(th.Theme, &v.autoSync, i18n.Translate("Auto-sync on save")).Layout),

		layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),

		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceStart}.Layout(gtx,
				layout.Rigid(material.Button(th.Theme, &v.testBtn, i18n.Translate("Test Connection")).Layout),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(material.Button(th.Theme, &v.saveBtn, i18n.Translate("Save")).Layout),
			)
		}),

		layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),

		layout.Rigid(func(gtx C) D {
			msg, isErr := v.message()
			if msg == "" {
				return layout.Dimensions{}
			}
			label := material.Label(th.Theme, th.TextSize, msg)
			if isErr {
				label.Color = th.ContrastBg
			}
			return label.Layout(gtx)
		}),
	)
}
