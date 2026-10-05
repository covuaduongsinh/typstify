// Package remoteproject is the minimal "Remote Project" desktop view for
// Giai đoạn C.1: browsing and editing a project's files on a self-hosted
// Typstify server over the network, with no local copy. Deliberately
// small -- plain-text editing only, no LSP/compile/preview (see
// service/projectstore for the ProjectStore abstraction this is built on,
// and the plan's Giai đoạn C.2+ for where those would plug in later).
package remoteproject

import (
	"context"
	"errors"
	"image/color"
	"log"
	"path"
	"strings"
	"sync/atomic"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"
	gvwidget "github.com/oligo/gioview/widget"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/service"
	"looz.ws/typstify/service/projectstore"
	"looz.ws/typstify/service/remote"
	"looz.ws/typstify/widgets"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

var RemoteProjectViewID = view.NewViewID("RemoteProjectView")

var errNotConnected = errors.New("not connected to any server -- go to Settings → Sync to connect first")

type fileRow struct {
	info  projectstore.FileInfo
	click widgets.InteractiveLabel
}

// RemoteProjectView lets the user pick (or create) a project on the
// connected remote server, browse its file tree one directory at a time
// (breadcrumb-style -- clicking into a folder replaces the listing;
// ".." goes back up), and open/edit/save one .typ or text file at a time.
type RemoteProjectView struct {
	*view.BaseView
	srv    *service.ServiceFacade
	client *remote.Client
	store  *projectstore.RemoteStore

	pathInput  gvwidget.TextField
	openBtn    widget.Clickable
	createBtn  widget.Clickable
	recent     []remote.RemoteProjectSummary
	recentRows []widgets.InteractiveLabel

	projectOpen atomic.Bool
	currentDir  string // slash-separated, relative to project root; "" = root
	entries     []fileRow
	upRow       widget.Clickable
	fileList    widget.List

	activeFile string
	dirty      bool
	editor     widget.Editor
	saveBtn    widget.Clickable

	status string
	err    error
	busy   atomic.Bool
}

func NewRemoteProjectView(srv *service.ServiceFacade) *RemoteProjectView {
	return &RemoteProjectView{
		BaseView: &view.BaseView{},
		srv:      srv,
		fileList: widget.List{List: layout.List{Axis: layout.Vertical}},
	}
}

func (v *RemoteProjectView) ID() view.ViewID { return RemoteProjectViewID }
func (v *RemoteProjectView) Title() string   { return i18n.Translate("Remote Project") }

func (v *RemoteProjectView) OnNavTo(intent view.Intent) error {
	v.BaseView.OnNavTo(intent)

	rs := v.srv.Settings().Remote()
	if !rs.Enabled || rs.Token == "" {
		v.err = errNotConnected
		return nil
	}

	v.client = remote.NewClient(rs.ServerURL, rs.Token)
	v.store = projectstore.NewRemoteStore(v.client)
	v.err = nil

	go v.refreshRecent()
	go v.checkCurrentProject()
	return nil
}

func (v *RemoteProjectView) refreshRecent() {
	recent, err := v.client.RecentProjects()
	if err != nil {
		log.Println("remote project: list recent failed:", err)
		return
	}
	v.recent = recent
	v.srv.RefreshWindow()
}

func (v *RemoteProjectView) checkCurrentProject() {
	path, err := v.client.CurrentProject()
	if err != nil || path == "" {
		return
	}
	v.projectOpen.Store(true)
	v.loadDir("")
}

func (v *RemoteProjectView) Layout(gtx C, th *theme.Theme) D {
	v.update(gtx)

	if v.err != nil {
		return layout.Center.Layout(gtx, func(gtx C) D {
			return misc.LayoutErrorLabel(gtx, th, v.err)
		})
	}

	if !v.projectOpen.Load() {
		return v.layoutProjectPicker(gtx, th)
	}

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(0.35, func(gtx C) D { return v.layoutFileTree(gtx, th) }),
		layout.Rigid(func(gtx C) D {
			return layout.Inset{Left: unit.Dp(1), Right: unit.Dp(1)}.Layout(gtx, func(gtx C) D { return D{} })
		}),
		layout.Flexed(0.65, func(gtx C) D { return v.layoutEditor(gtx, th) }),
	)
}

func (v *RemoteProjectView) layoutProjectPicker(gtx C, th *theme.Theme) D {
	return layout.UniformInset(unit.Dp(24)).Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				label := material.Label(th.Theme, th.TextSize, i18n.Translate("Enter the project folder path on the server (absolute path on the server, not this machine):"))
				label.LineHeightScale = 1.5
				return label.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Rigid(func(gtx C) D {
				v.pathInput.SingleLine = true
				v.pathInput.Alignment = text.Start
				return v.pathInput.Layout(gtx, th, "/path/to/project")
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Rigid(func(gtx C) D {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return material.Button(th.Theme, &v.openBtn, i18n.Translate("Open")).Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Rigid(func(gtx C) D {
						return material.Button(th.Theme, &v.createBtn, i18n.Translate("Create new project")).Layout(gtx)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(24)}.Layout),
			layout.Rigid(func(gtx C) D {
				if len(v.recent) == 0 {
					return D{}
				}
				label := material.Subtitle2(th.Theme, i18n.Translate("Recent projects on server"))
				return label.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx C) D { return v.layoutRecent(gtx, th) }),
		)
	})
}

func (v *RemoteProjectView) layoutRecent(gtx C, th *theme.Theme) D {
	for len(v.recentRows) < len(v.recent) {
		v.recentRows = append(v.recentRows, widgets.InteractiveLabel{})
	}

	children := make([]layout.FlexChild, 0, len(v.recent))
	for i, p := range v.recent {
		i, p := i, p
		children = append(children, layout.Rigid(func(gtx C) D {
			row := &v.recentRows[i]
			if row.Update(gtx) {
				v.openProject(p.Path)
			}
			return row.Layout(gtx, th, func(gtx C, textColor color.NRGBA) D {
				label := material.Label(th.Theme, th.TextSize, p.Path)
				label.Color = textColor
				return label.Layout(gtx)
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (v *RemoteProjectView) layoutFileTree(gtx C, th *theme.Theme) D {
	rows := len(v.entries)
	if v.currentDir != "" {
		rows++
	}

	list := material.List(th.Theme, &v.fileList)
	return list.Layout(gtx, rows, func(gtx C, index int) D {
		if v.currentDir != "" {
			if index == 0 {
				if v.upRow.Clicked(gtx) {
					parent := path.Dir(v.currentDir)
					if parent == "." {
						parent = ""
					}
					v.loadDir(parent)
				}
				btn := material.Button(th.Theme, &v.upRow, "..")
				return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(4)}.Layout(gtx, btn.Layout)
			}
			index--
		}

		row := &v.entries[index]
		if row.click.Update(gtx) {
			if row.info.IsDir {
				v.loadDir(row.info.Path)
			} else {
				v.openFile(row.info.Path)
			}
		}

		return layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(6)}.Layout(gtx, func(gtx C) D {
			return row.click.Layout(gtx, th, func(gtx C, textColor color.NRGBA) D {
				name := row.info.Name
				if row.info.IsDir {
					name = name + "/"
				}
				label := material.Label(th.Theme, th.TextSize, name)
				label.Color = textColor
				return label.Layout(gtx)
			})
		})
	})
}

func (v *RemoteProjectView) layoutEditor(gtx C, th *theme.Theme) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			title := v.activeFile
			if title == "" {
				title = i18n.Translate("(no file opened)")
			}
			if v.dirty {
				title += " *"
			}
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, material.Label(th.Theme, th.TextSize, title).Layout),
				layout.Rigid(func(gtx C) D {
					if v.activeFile == "" {
						return D{}
					}
					if !v.dirty {
						gtx = gtx.Disabled()
					}
					return material.Button(th.Theme, &v.saveBtn, i18n.Translate("Save")).Layout(gtx)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx C) D {
			if v.status != "" {
				label := material.Label(th.Theme, th.TextSize*0.85, v.status)
				label.Color = misc.WithAlpha(th.Fg, 0xa0)
				return label.Layout(gtx)
			}
			return D{}
		}),
		layout.Flexed(1, func(gtx C) D {
			ed := material.Editor(th.Theme, &v.editor, "")
			v.editor.WrapPolicy = text.WrapWords
			return widget.Border{Color: misc.WithAlpha(th.Fg, 0x30), Width: unit.Dp(0.5)}.Layout(gtx, func(gtx C) D {
				return layout.UniformInset(unit.Dp(8)).Layout(gtx, ed.Layout)
			})
		}),
	)
}

func (v *RemoteProjectView) update(gtx C) {
	if v.openBtn.Clicked(gtx) {
		v.openProject(strings.TrimSpace(v.pathInput.Text()))
	}
	if v.createBtn.Clicked(gtx) {
		v.createProject(strings.TrimSpace(v.pathInput.Text()))
	}
	if v.saveBtn.Clicked(gtx) {
		v.saveActiveFile()
	}
	if ev, ok := v.editor.Update(gtx); ok {
		if _, isChange := ev.(widget.ChangeEvent); isChange {
			v.dirty = true
		}
	}
}

func (v *RemoteProjectView) openProject(remotePath string) {
	if remotePath == "" || !v.busy.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer v.busy.Store(false)
		if _, err := v.client.OpenProject(remotePath); err != nil {
			v.err = err
			v.srv.RefreshWindow()
			return
		}
		v.projectOpen.Store(true)
		v.err = nil
		v.loadDir("")
	}()
}

func (v *RemoteProjectView) createProject(remotePath string) {
	if remotePath == "" || !v.busy.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer v.busy.Store(false)
		if _, err := v.client.CreateProject(remotePath); err != nil {
			v.err = err
			v.srv.RefreshWindow()
			return
		}
		v.projectOpen.Store(true)
		v.err = nil
		v.loadDir("")
	}()
}

func (v *RemoteProjectView) loadDir(dir string) {
	go func() {
		entries, err := v.store.Tree(context.Background(), dir)
		if err != nil {
			v.err = err
			v.srv.RefreshWindow()
			return
		}
		rows := make([]fileRow, len(entries))
		for i, e := range entries {
			rows[i] = fileRow{info: e}
		}
		v.currentDir = dir
		v.entries = rows
		v.err = nil
		v.srv.RefreshWindow()
	}()
}

func (v *RemoteProjectView) openFile(relPath string) {
	go func() {
		content, err := v.store.ReadFile(context.Background(), relPath)
		if err != nil {
			v.err = err
			v.srv.RefreshWindow()
			return
		}
		v.activeFile = relPath
		v.editor.SetText(string(content))
		v.dirty = false
		v.status = ""
		v.srv.RefreshWindow()
	}()
}

func (v *RemoteProjectView) saveActiveFile() {
	if v.activeFile == "" {
		return
	}
	relPath := v.activeFile
	content := v.editor.Text()
	go func() {
		if err := v.store.WriteFile(context.Background(), relPath, []byte(content)); err != nil {
			v.err = err
			v.srv.RefreshWindow()
			return
		}
		v.dirty = false
		v.status = i18n.Translate("Saved.")
		v.err = nil
		v.srv.RefreshWindow()
	}()
}
