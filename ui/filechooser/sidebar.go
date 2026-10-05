package filechooser

import (
	"image"
	"image/color"
	"path/filepath"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/theme"
	"golang.org/x/exp/shiny/materialdesign/icons"
	"looz.ws/typstify/ui/uitokens"
	appicons "looz.ws/typstify/widgets/icons"
)

var (
	iconHome, _      = widget.NewIcon(icons.ActionHome)
	iconDisk, _      = widget.NewIcon(icons.HardwareComputer)
	iconPresentation = appicons.NewSvgIcon(appicons.Presentation)
	iconFolderSidebar, _ = widget.NewIcon(icons.FileFolder)
)

// Sidebar renders Favorites and Locations (Drives/Volumes).
type Sidebar struct {
	favorites []*FavoriteItem
	favClicks []widget.Clickable

	volumes   []*VolumeItem
	volClicks []widget.Clickable

	currentPath string

	favList widget.List
	volList widget.List

	OnNavigate func(path string)
}

func NewSidebar() *Sidebar {
	sb := &Sidebar{
		favorites: GetFavorites(),
		volumes:   DetectVolumes(),
		favList: widget.List{
			List: layout.List{Axis: layout.Vertical},
		},
		volList: widget.List{
			List: layout.List{Axis: layout.Vertical},
		},
	}
	sb.favClicks = make([]widget.Clickable, len(sb.favorites))
	sb.volClicks = make([]widget.Clickable, len(sb.volumes))
	return sb
}

func (sb *Sidebar) Refresh() {
	sb.favorites = GetFavorites()
	sb.volumes = DetectVolumes()
	sb.favClicks = make([]widget.Clickable, len(sb.favorites))
	sb.volClicks = make([]widget.Clickable, len(sb.volumes))
}

func (sb *Sidebar) SetCurrentPath(path string) {
	sb.currentPath = filepath.Clean(path)
}

func (sb *Sidebar) Layout(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	return layout.Inset{
		Top:    unit.Dp(8),
		Bottom: unit.Dp(8),
		Left:   unit.Dp(8),
		Right:  unit.Dp(8),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis: layout.Vertical,
		}.Layout(gtx,
			// Section: Favorites
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return sb.layoutSectionHeader(gtx, th, "Favorites")
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return sb.layoutFavorites(gtx, th)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			// Section: Locations / Drives
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return sb.layoutSectionHeader(gtx, th, "Locations")
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return sb.layoutLocations(gtx, th)
			}),
		)
	})
}

func (sb *Sidebar) layoutSectionHeader(gtx layout.Context, th *theme.Theme, title string) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(6), Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th.Theme, th.TextSize*0.8, title)
		lbl.Font.Weight = font.Bold
		lbl.Color = uitokens.SubtleTextColor(th)
		return lbl.Layout(gtx)
	})
}

func (sb *Sidebar) layoutFavorites(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	if len(sb.favClicks) < len(sb.favorites) {
		sb.favClicks = make([]widget.Clickable, len(sb.favorites))
	}

	return material.List(th.Theme, &sb.favList).Layout(gtx, len(sb.favorites), func(gtx layout.Context, index int) layout.Dimensions {
		fav := sb.favorites[index]
		clk := &sb.favClicks[index]

		if clk.Clicked(gtx) && sb.OnNavigate != nil {
			sb.OnNavigate(fav.Path)
		}

		isSelected := filepath.Clean(fav.Path) == sb.currentPath

		return sb.layoutSidebarItem(gtx, th, clk, fav.Label, fav.Icon, isSelected)
	})
}

func (sb *Sidebar) layoutLocations(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	if len(sb.volClicks) < len(sb.volumes) {
		sb.volClicks = make([]widget.Clickable, len(sb.volumes))
	}

	return material.List(th.Theme, &sb.volList).Layout(gtx, len(sb.volumes), func(gtx layout.Context, index int) layout.Dimensions {
		vol := sb.volumes[index]
		clk := &sb.volClicks[index]

		if clk.Clicked(gtx) && sb.OnNavigate != nil {
			sb.OnNavigate(vol.MountPoint)
		}

		isSelected := filepath.Clean(vol.MountPoint) == sb.currentPath

		return sb.layoutSidebarItem(gtx, th, clk, vol.Label, "disk", isSelected)
	})
}

func (sb *Sidebar) layoutSidebarItem(gtx layout.Context, th *theme.Theme, clk *widget.Clickable, label string, iconType string, isSelected bool) layout.Dimensions {
	var bgColor color.NRGBA
	if isSelected {
		bgColor = th.ContrastBg
		bgColor.A = 35
	} else if clk.Hovered() {
		bgColor = uitokens.HoverBgColor(th)
	}

	return material.Clickable(gtx, clk, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				if bgColor.A > 0 {
					defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, int(gtx.Dp(uitokens.RadiusSmall))).Push(gtx.Ops).Pop()
					paint.ColorOp{Color: bgColor}.Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)
				}
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Left:   unit.Dp(8),
					Right:  unit.Dp(8),
					Top:    unit.Dp(5),
					Bottom: unit.Dp(5),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return sb.renderSidebarIcon(gtx, th, iconType, isSelected)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th.Theme, th.TextSize*0.95, label)
							if isSelected {
								lbl.Font.Weight = font.SemiBold
								lbl.Color = th.ContrastBg
							}
							return lbl.Layout(gtx)
						}),
					)
				})
			},
		)
	})
}

func (sb *Sidebar) renderSidebarIcon(gtx layout.Context, th *theme.Theme, iconType string, isSelected bool) layout.Dimensions {
	iconColor := th.Fg
	if isSelected {
		iconColor = th.ContrastBg
	} else {
		iconColor = uitokens.SubtleTextColor(th)
	}

	switch iconType {
	case "home":
		return iconHome.Layout(gtx, iconColor)
	case "disk":
		return iconDisk.Layout(gtx, iconColor)
	default:
		return iconFolderSidebar.Layout(gtx, iconColor)
	}
}
