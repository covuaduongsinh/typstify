package filechooser

import (
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/theme"
	gvwidget "github.com/oligo/gioview/widget"
	"looz.ws/typstify/ui/uitokens"
	appicons "looz.ws/typstify/widgets/icons"
)

var (
	iconFolderPlus = appicons.NewSvgIcon(appicons.FolderPlus)
)

// BottomBar handles the bottom footer actions including New Folder, Save As filename, and Confirm/Cancel buttons.
type BottomBar struct {
	op OpKind

	isAddingFolder     bool
	newFolderInput     gvwidget.TextField
	createFolderBtn    widget.Clickable
	cancelNewFolderBtn widget.Clickable
	toggleNewFolderBtn widget.Clickable

	saveFileInput gvwidget.TextField

	confirmBtn widget.Clickable
	cancelBtn  widget.Clickable

	selectedText string
	currentPath  string

	err       error
	errShowAt time.Time

	OnConfirm   func() error
	OnCancel    func()
	OnNewFolder func(folderName string) error
}

func NewBottomBar(op OpKind) *BottomBar {
	bar := &BottomBar{
		op: op,
	}
	bar.newFolderInput.SetText("New Folder")
	return bar
}

func (b *BottomBar) SetOp(op OpKind) {
	b.op = op
}

func (b *BottomBar) SetCurrentPath(path string) {
	b.currentPath = path
}

func (b *BottomBar) SetSelectedText(text string) {
	b.selectedText = text
}

func (b *BottomBar) SetSaveFileName(name string) {
	b.saveFileInput.SetText(name)
}

func (b *BottomBar) SaveFileName() string {
	return strings.TrimSpace(b.saveFileInput.Text())
}

func (b *BottomBar) SetError(err error) {
	b.err = err
	b.errShowAt = time.Now()
}

func (b *BottomBar) Layout(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	// Handle action buttons
	if b.cancelBtn.Clicked(gtx) && b.OnCancel != nil {
		b.OnCancel()
	}

	if b.confirmBtn.Clicked(gtx) && b.OnConfirm != nil {
		if err := b.OnConfirm(); err != nil {
			b.SetError(err)
		}
	}

	if b.toggleNewFolderBtn.Clicked(gtx) {
		b.isAddingFolder = !b.isAddingFolder
		if b.isAddingFolder {
			b.newFolderInput.SetText("New Folder")
		}
	}

	if b.cancelNewFolderBtn.Clicked(gtx) {
		b.isAddingFolder = false
	}

	if b.createFolderBtn.Clicked(gtx) {
		folderName := strings.TrimSpace(b.newFolderInput.Text())
		if folderName == "" {
			b.SetError(errors.New("Folder name cannot be empty"))
		} else if strings.ContainsAny(folderName, `\/:*?"<>|`) || folderName == "." || folderName == ".." {
			b.SetError(errors.New("Folder name contains invalid characters"))
		} else {
			targetDir := filepath.Join(b.currentPath, folderName)
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				b.SetError(fmt.Errorf("Failed to create folder: %w", err))
			} else {
				b.isAddingFolder = false
				if b.OnNewFolder != nil {
					_ = b.OnNewFolder(folderName)
				}
			}
		}
	}

	// Auto-dismiss error after 3 seconds
	if b.err != nil {
		if time.Since(b.errShowAt) > 3*time.Second {
			b.err = nil
		} else {
			gtx.Execute(op.InvalidateCmd{At: b.errShowAt.Add(3 * time.Second)})
		}
	}

	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
		Spacing:   layout.SpaceBetween,
	}.Layout(gtx,
		// Left side: New Folder or Inline Create Form
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return b.layoutLeftArea(gtx, th)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
		// Middle: Error message or Selected item info
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return b.layoutCenterArea(gtx, th)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
		// Right side: Cancel & Confirm buttons
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return b.layoutRightButtons(gtx, th)
		}),
	)
}

func (b *BottomBar) layoutLeftArea(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	if b.isAddingFolder {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Alignment: layout.Middle,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(160))
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(220))
				return b.newFolderInput.Layout(gtx, th, "Folder name...")
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(th.Theme, &b.createFolderBtn, "Create")
				btn.Inset = layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10), Top: unit.Dp(6), Bottom: unit.Dp(6)}
				return btn.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(th.Theme, &b.cancelNewFolderBtn, "Cancel")
				btn.Background = th.Bg
				btn.Color = th.Fg
				btn.Inset = layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Top: unit.Dp(6), Bottom: unit.Dp(6)}
				return btn.Layout(gtx)
			}),
		)
	}

	return material.Clickable(gtx, &b.toggleNewFolderBtn, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Alignment: layout.Middle,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				fg := th.Fg
				if b.toggleNewFolderBtn.Hovered() {
					fg = th.ContrastBg
				}
				return iconFolderPlus.Layout(gtx, fg, unit.Sp(16))
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th.Theme, th.TextSize*0.95, "New Folder")
				if b.toggleNewFolderBtn.Hovered() {
					lbl.Color = th.ContrastBg
				}
				return lbl.Layout(gtx)
			}),
		)
	})
}

func (b *BottomBar) layoutCenterArea(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	if b.err != nil {
		lbl := material.Label(th.Theme, th.TextSize*0.9, b.err.Error())
		lbl.Color = uitokens.ErrorColor(th)
		lbl.Alignment = text.Start
		return lbl.Layout(gtx)
	}

	if b.op == SaveFileOp {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Alignment: layout.Middle,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th.Theme, th.TextSize*0.9, "File name:")
				lbl.Color = uitokens.SubtleTextColor(th)
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return b.saveFileInput.Layout(gtx, th, "Enter file name...")
			}),
		)
	}

	if b.selectedText != "" {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Alignment: layout.Middle,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th.Theme, th.TextSize*0.9, "Selected:")
				lbl.Color = uitokens.SubtleTextColor(th)
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th.Theme, th.TextSize*0.9, b.selectedText)
				lbl.MaxLines = 1
				return lbl.Layout(gtx)
			}),
		)
	}

	return layout.Dimensions{}
}

func (b *BottomBar) layoutRightButtons(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	confirmLabel := "Open"
	switch b.op {
	case OpenFolderOp:
		confirmLabel = "Select Folder"
	case SaveFileOp:
		confirmLabel = "Save"
	case OpenFilesOp:
		confirmLabel = "Open"
	}

	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th.Theme, &b.cancelBtn, "Cancel")
			btn.Background = color.NRGBA{R: 0, G: 0, B: 0, A: 0}
			btn.Color = th.Fg
			btn.Inset = layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Top: unit.Dp(8), Bottom: unit.Dp(8)}
			return btn.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th.Theme, &b.confirmBtn, confirmLabel)
			btn.Inset = layout.Inset{Left: unit.Dp(20), Right: unit.Dp(20), Top: unit.Dp(8), Bottom: unit.Dp(8)}
			return btn.Layout(gtx)
		}),
	)
}
