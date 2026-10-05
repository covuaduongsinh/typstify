package filechooser

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/theme"
	gvwidget "github.com/oligo/gioview/widget"
	"golang.org/x/exp/shiny/materialdesign/icons"
	"looz.ws/typstify/ui/uitokens"
	appicons "looz.ws/typstify/widgets/icons"
)

// HistoryStack tracks visited directories for backward/forward navigation.
type HistoryStack struct {
	paths  []string
	cursor int
}

func NewHistoryStack() *HistoryStack {
	return &HistoryStack{
		paths:  make([]string, 0),
		cursor: -1,
	}
}

func (h *HistoryStack) Push(path string) {
	cleaned := filepath.Clean(path)
	if h.cursor >= 0 && h.cursor < len(h.paths) && h.paths[h.cursor] == cleaned {
		return
	}

	// Truncate forward history if navigating from a past point
	if h.cursor+1 < len(h.paths) {
		h.paths = h.paths[:h.cursor+1]
	}

	h.paths = append(h.paths, cleaned)
	h.cursor = len(h.paths) - 1
}

func (h *HistoryStack) CanBack() bool {
	return h.cursor > 0
}

func (h *HistoryStack) Back() string {
	if !h.CanBack() {
		return ""
	}
	h.cursor--
	return h.paths[h.cursor]
}

func (h *HistoryStack) CanForward() bool {
	return h.cursor >= 0 && h.cursor < len(h.paths)-1
}

func (h *HistoryStack) Forward() string {
	if !h.CanForward() {
		return ""
	}
	h.cursor++
	return h.paths[h.cursor]
}

func (h *HistoryStack) CanUp(currentPath string) bool {
	cleaned := filepath.Clean(currentPath)
	parent := filepath.Dir(cleaned)
	return parent != "" && parent != cleaned && parent != "."
}

func (h *HistoryStack) Up(currentPath string) string {
	cleaned := filepath.Clean(currentPath)
	parent := filepath.Dir(cleaned)
	if parent == "" || parent == cleaned || parent == "." {
		return currentPath
	}
	return parent
}

var (
	iconBack, _    = widget.NewIcon(icons.NavigationArrowBack)
	iconForward, _ = widget.NewIcon(icons.NavigationArrowForward)
	iconUp, _      = widget.NewIcon(icons.NavigationArrowUpward)
	iconRefresh, _ = widget.NewIcon(icons.NavigationRefresh)
	iconEdit, _    = widget.NewIcon(icons.EditorModeEdit)
	iconCheck, _   = widget.NewIcon(icons.NavigationCheck)
	iconSearch     = appicons.NewSvgIcon(appicons.Search)
	iconClose      = appicons.NewSvgIcon(appicons.X)
)

// NavigationBar holds the navigation buttons, breadcrumb/address bar, and search box.
type NavigationBar struct {
	history *HistoryStack

	backBtn       widget.Clickable
	forwardBtn    widget.Clickable
	upBtn         widget.Clickable
	refreshBtn    widget.Clickable
	editToggleBtn widget.Clickable
	submitPathBtn widget.Clickable

	isEditingAddress bool
	addressInput     gvwidget.TextField
	searchInput      gvwidget.TextField
	lastSearchQuery  string

	currentPath string
	breadcrumbs []*BreadcrumbSegment

	OnNavigate func(path string)
	OnRefresh  func()
	OnSearch   func(query string)
}

func NewNavigationBar(history *HistoryStack) *NavigationBar {
	nav := &NavigationBar{
		history: history,
	}
	return nav
}

func (nav *NavigationBar) SetCurrentPath(path string) {
	cleaned := filepath.Clean(path)
	nav.currentPath = cleaned
	nav.breadcrumbs = ParseBreadcrumbs(cleaned)
	if !nav.isEditingAddress {
		nav.addressInput.SetText(cleaned)
	}
}

func (nav *NavigationBar) CurrentPath() string {
	return nav.currentPath
}

func (nav *NavigationBar) Layout(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	// Handle button clicks
	if nav.backBtn.Clicked(gtx) && nav.history.CanBack() {
		target := nav.history.Back()
		if target != "" && nav.OnNavigate != nil {
			nav.OnNavigate(target)
		}
	}

	if nav.forwardBtn.Clicked(gtx) && nav.history.CanForward() {
		target := nav.history.Forward()
		if target != "" && nav.OnNavigate != nil {
			nav.OnNavigate(target)
		}
	}

	if nav.upBtn.Clicked(gtx) && nav.history.CanUp(nav.currentPath) {
		target := nav.history.Up(nav.currentPath)
		if target != "" && nav.OnNavigate != nil {
			nav.OnNavigate(target)
		}
	}

	if nav.refreshBtn.Clicked(gtx) && nav.OnRefresh != nil {
		nav.OnRefresh()
	}

	if nav.editToggleBtn.Clicked(gtx) {
		nav.isEditingAddress = !nav.isEditingAddress
		if nav.isEditingAddress {
			nav.addressInput.SetText(nav.currentPath)
		}
	}

	if nav.submitPathBtn.Clicked(gtx) && nav.isEditingAddress {
		nav.commitAddressInput()
	}

	// Handle search input events
	if nav.searchInput.Changed() {
		searchQuery := strings.TrimSpace(nav.searchInput.Text())
		if searchQuery != nav.lastSearchQuery {
			nav.lastSearchQuery = searchQuery
			if nav.OnSearch != nil {
				nav.OnSearch(searchQuery)
			}
		}
	}


	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
		Spacing:   layout.SpaceBetween,
	}.Layout(gtx,
		// Nav buttons: Back, Forward, Up, Refresh
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return nav.layoutNavControls(gtx, th)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		// Breadcrumbs or Address Input Bar
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return nav.layoutAddressOrBreadcrumbs(gtx, th)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		// Search Input
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return nav.layoutSearch(gtx, th)
		}),
	)
}

func (nav *NavigationBar) commitAddressInput() {
	path := strings.TrimSpace(nav.addressInput.Text())
	if path == "" {
		return
	}
	cleaned := filepath.Clean(path)
	info, err := os.Stat(cleaned)
	if err == nil && info.IsDir() {
		nav.isEditingAddress = false
		if nav.OnNavigate != nil {
			nav.OnNavigate(cleaned)
		}
	} else if err == nil && !info.IsDir() {
		// If path is a file, navigate to its parent directory
		dir := filepath.Dir(cleaned)
		nav.isEditingAddress = false
		if nav.OnNavigate != nil {
			nav.OnNavigate(dir)
		}
	}
}

func (nav *NavigationBar) layoutNavControls(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	canBack := nav.history.CanBack()
	canForward := nav.history.CanForward()
	canUp := nav.history.CanUp(nav.currentPath)

	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return renderIconButton(gtx, th, &nav.backBtn, iconBack, "Back", canBack)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return renderIconButton(gtx, th, &nav.forwardBtn, iconForward, "Forward", canForward)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return renderIconButton(gtx, th, &nav.upBtn, iconUp, "Up Directory", canUp)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return renderIconButton(gtx, th, &nav.refreshBtn, iconRefresh, "Refresh", true)
		}),
	)
}

func (nav *NavigationBar) layoutAddressOrBreadcrumbs(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	borderColor := uitokens.BorderColor(th)
	bg := th.Bg

	return widget.Border{
		Color:        borderColor,
		Width:        unit.Dp(1),
		CornerRadius: uitokens.RadiusMedium,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, int(gtx.Dp(uitokens.RadiusMedium))).Push(gtx.Ops).Pop()
				paint.ColorOp{Color: bg}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Left:   unit.Dp(8),
					Right:  unit.Dp(6),
					Top:    unit.Dp(3),
					Bottom: unit.Dp(3),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					if nav.isEditingAddress {
						return layout.Flex{
							Axis:      layout.Horizontal,
							Alignment: layout.Middle,
						}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return nav.addressInput.Layout(gtx, th, "Enter directory path...")
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return renderIconButton(gtx, th, &nav.submitPathBtn, iconCheck, "Go", true)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return renderIconButton(gtx, th, &nav.editToggleBtn, iconClose, "Cancel", true)
							}),
						)
					}

					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return LayoutBreadcrumbs(gtx, th, nav.breadcrumbs, func(targetPath string) {
								if nav.OnNavigate != nil {
									nav.OnNavigate(targetPath)
								}
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return renderIconButton(gtx, th, &nav.editToggleBtn, iconEdit, "Edit Path", true)
						}),
					)
				})
			},
		)
	})
}

func (nav *NavigationBar) layoutSearch(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	gtx.Constraints.Max.X = gtx.Dp(unit.Dp(180))
	gtx.Constraints.Min.X = gtx.Dp(unit.Dp(140))

	return widget.Border{
		Color:        uitokens.BorderColor(th),
		Width:        unit.Dp(1),
		CornerRadius: uitokens.RadiusMedium,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Left:   unit.Dp(8),
			Right:  unit.Dp(6),
			Top:    unit.Dp(3),
			Bottom: unit.Dp(3),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return iconSearch.Layout(gtx, uitokens.SubtleTextColor(th), unit.Sp(13))
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return nav.searchInput.Layout(gtx, th, "Search...")
				}),
			)
		})
	})
}

func renderIconButton(gtx layout.Context, th *theme.Theme, clk *widget.Clickable, ic any, tooltip string, enabled bool) layout.Dimensions {
	if !enabled {
		return layout.Inset{
			Left: unit.Dp(4), Right: unit.Dp(4), Top: unit.Dp(4), Bottom: unit.Dp(4),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			fg := th.Fg
			fg.A = 60
			return renderIconHelper(gtx, ic, fg)
		})
	}

	return material.Clickable(gtx, clk, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Left: unit.Dp(4), Right: unit.Dp(4), Top: unit.Dp(4), Bottom: unit.Dp(4),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			fg := th.Fg
			if clk.Hovered() {
				fg = th.ContrastBg
			}
			return renderIconHelper(gtx, ic, fg)
		})
	})
}

func renderIconHelper(gtx layout.Context, ic any, fg color.NRGBA) layout.Dimensions {
	sz := unit.Dp(16)
	switch v := ic.(type) {
	case *widget.Icon:
		return v.Layout(gtx, fg)
	case *appicons.SvgIcon:
		return v.Layout(gtx, fg, unit.Sp(14))
	default:
		return layout.Spacer{Width: sz, Height: sz}.Layout(gtx)
	}
}
