package commandbar

import (
	"image"
	"image/color"
	"strings"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/explorer"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"
	gvwidget "github.com/oligo/gioview/widget"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/service"
	"looz.ws/typstify/service/bus"
	"looz.ws/typstify/ui/dialog"
	"looz.ws/typstify/ui/pkgmgmt"
	"looz.ws/typstify/ui/settings"
	"looz.ws/typstify/ui/uitokens"
	"looz.ws/typstify/widgets"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type CommandItem struct {
	ID       string
	Title    string
	Subtitle string
	Shortcut string
	Action   func(srv *service.ServiceFacade, vm view.ViewManager)
}

type CommandBar struct {
	srv         *service.ServiceFacade
	vm          view.ViewManager
	visible     bool
	searchInput gvwidget.TextField
	list        layout.List
	selectedIdx int
	labels      []*widgets.InteractiveLabel
	dismissBtn  widget.Clickable
	commands    []CommandItem
	filtered    []CommandItem
}

func NewCommandBar(srv *service.ServiceFacade, vm view.ViewManager) *CommandBar {
	cb := &CommandBar{
		srv:  srv,
		vm:   vm,
		list: layout.List{Axis: layout.Vertical},
	}
	cb.initCommands()
	return cb
}

func (cb *CommandBar) initCommands() {
	cb.commands = []CommandItem{
		{
			ID:       "file.new_project",
			Title:    "New Project / Document",
			Subtitle: "Create a new Typst document or package",
			Shortcut: "Ctrl+N",
			Action: func(srv *service.ServiceFacade, vm view.ViewManager) {
				vm.RequestSwitch(view.Intent{
					Target:      dialog.CreateProjectDialogViewID,
					ShowAsModal: true,
				})
			},
		},
		{
			ID:       "file.open_folder",
			Title:    "Open Project Folder",
			Subtitle: "Open an existing folder from local disk",
			Shortcut: "Ctrl+O",
			Action: func(srv *service.ServiceFacade, vm view.ViewManager) {
				go func() {
					if chooser, ok := srv.FileChooser().(*explorer.FileChooser); ok {
						if folder, err := chooser.ChooseFolder(); err == nil && folder != "" {
							srv.EventBus().Emit(bus.TopicProjectSwitched, folder)
						}
					}
				}()
			},
		},
		{
			ID:       "general.settings",
			Title:    "Settings",
			Subtitle: "Open application preferences",
			Shortcut: "Ctrl+,",
			Action: func(srv *service.ServiceFacade, vm view.ViewManager) {
				vm.RequestSwitch(view.Intent{
					Target:     settings.SettingViewID,
					RequireNew: true,
				})
			},
		},
		{
			ID:       "view.toggle_chat",
			Title:    "Toggle AI Assistant",
			Subtitle: "Open or close AI chat panel",
			Shortcut: "Ctrl+L",
			Action: func(srv *service.ServiceFacade, vm view.ViewManager) {
				srv.EventBus().Emit(bus.TopicToggleChat, nil)
			},
		},
		{
			ID:       "view.toggle_console",
			Title:    "Toggle Console",
			Subtitle: "Show or hide terminal console",
			Shortcut: "Ctrl+K",
			Action: func(srv *service.ServiceFacade, vm view.ViewManager) {
				srv.EventBus().Emit(bus.TopicToggleConsole, nil)
			},
		},
		{
			ID:       "view.toggle_drawer",
			Title:    "Toggle Sidebar Drawer",
			Subtitle: "Show or hide left navigation drawer",
			Shortcut: "Ctrl+D",
			Action: func(srv *service.ServiceFacade, vm view.ViewManager) {
				srv.EventBus().Emit(bus.TopicToggleDrawer, nil)
			},
		},
		{
			ID:       "pkg.manager",
			Title:    "Package Management",
			Subtitle: "Browse and install Typst universe packages",
			Action: func(srv *service.ServiceFacade, vm view.ViewManager) {
				vm.RequestSwitch(view.Intent{
					Target:     pkgmgmt.PkgListViewID,
					RequireNew: true,
				})
			},
		},
		{
			ID:       "sync.vps_now",
			Title:    "VPS Sync: Sync Now",
			Subtitle: "Perform manual file synchronization with VPS server",
			Action: func(srv *service.ServiceFacade, vm view.ViewManager) {
				srv.EventBus().Emit(bus.TopicVpsSyncNow, nil)
			},
		},
	}
}

func (cb *CommandBar) Show(gtx C) {
	cb.visible = true
	cb.searchInput.SetText("")
	cb.selectedIdx = 0
	cb.filterCommands("")
}

func (cb *CommandBar) Hide() {
	cb.visible = false
	cb.selectedIdx = 0
}

func (cb *CommandBar) IsVisible() bool {
	return cb.visible
}

func (cb *CommandBar) Toggle(gtx C) {
	if cb.visible {
		cb.Hide()
	} else {
		cb.Show(gtx)
	}
}

func (cb *CommandBar) filterCommands(query string) {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		cb.filtered = make([]CommandItem, len(cb.commands))
		copy(cb.filtered, cb.commands)
		return
	}

	cb.filtered = cb.filtered[:0]
	for _, cmd := range cb.commands {
		title := strings.ToLower(i18n.Translate(cmd.Title))
		sub := strings.ToLower(i18n.Translate(cmd.Subtitle))
		if strings.Contains(title, query) || strings.Contains(sub, query) || strings.Contains(strings.ToLower(cmd.ID), query) {
			cb.filtered = append(cb.filtered, cmd)
		}
	}
	if cb.selectedIdx >= len(cb.filtered) {
		cb.selectedIdx = max(0, len(cb.filtered)-1)
	}
}

func (cb *CommandBar) executeSelected() {
	if cb.selectedIdx >= 0 && cb.selectedIdx < len(cb.filtered) {
		item := cb.filtered[cb.selectedIdx]
		cb.Hide()
		if item.Action != nil {
			item.Action(cb.srv, cb.vm)
		}
	}
}

func (cb *CommandBar) Layout(gtx C, th *theme.Theme) D {
	if !cb.visible {
		return D{}
	}

	// Handle search input changes
	if cb.searchInput.Changed() {
		cb.filterCommands(cb.searchInput.Text())
	}

	// Keyboard event handling for command palette
	for {
		ke, ok := gtx.Event(
			key.FocusFilter{Target: cb},
			key.Filter{Focus: cb, Name: key.NameEscape},
			key.Filter{Focus: cb, Name: key.NameUpArrow},
			key.Filter{Focus: cb, Name: key.NameDownArrow},
			key.Filter{Focus: cb, Name: key.NameEnter},
			key.Filter{Focus: cb, Name: key.NameReturn},
		)
		if !ok {
			break
		}
		switch ev := ke.(type) {
		case key.Event:
			if ev.State == key.Press || ev.State == key.Release {
				if ev.Name == key.NameEscape {
					cb.Hide()
					gtx.Execute(op.InvalidateCmd{})
					return D{}
				}
				if ev.State == key.Release {
					if ev.Name == key.NameUpArrow {
						if cb.selectedIdx > 0 {
							cb.selectedIdx--
							gtx.Execute(op.InvalidateCmd{})
						}
					}
					if ev.Name == key.NameDownArrow {
						if cb.selectedIdx < len(cb.filtered)-1 {
							cb.selectedIdx++
							gtx.Execute(op.InvalidateCmd{})
						}
					}
					if ev.Name == key.NameEnter || ev.Name == key.NameReturn {
						cb.executeSelected()
						gtx.Execute(op.InvalidateCmd{})
						return D{}
					}
				}
			}
		}
	}

	// Modal backdrop
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			// Dim background
			rect := clip.Rect(image.Rectangle{Max: gtx.Constraints.Max})
			paint.FillShape(gtx.Ops, color.NRGBA{A: 0x60}, rect.Op())

			// Dismiss on clicking backdrop
			return cb.dismissBtn.Layout(gtx, func(gtx C) D {
				if cb.dismissBtn.Clicked(gtx) {
					cb.Hide()
				}
				return D{Size: gtx.Constraints.Max}
			})
		}),

		layout.Stacked(func(gtx C) D {
			return layout.Inset{
				Top:   unit.Dp(60),
				Left:  unit.Dp(32),
				Right: unit.Dp(32),
			}.Layout(gtx, func(gtx C) D {
				return layout.Center.Layout(gtx, func(gtx C) D {
					return cb.layoutPalette(gtx, th)
				})
			})
		}),
	)
}

func (cb *CommandBar) layoutPalette(gtx C, th *theme.Theme) D {
	gtx.Constraints.Min.X = min(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(520)))
	gtx.Constraints.Max.X = gtx.Constraints.Min.X

	rr := gtx.Dp(uitokens.RadiusMedium)
	rrect := clip.RRect{
		Rect: image.Rectangle{Max: gtx.Constraints.Max},
		NE:   rr, SE: rr, NW: rr, SW: rr,
	}

	macro := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top:    unit.Dp(8),
		Bottom: unit.Dp(8),
		Left:   unit.Dp(8),
		Right:  unit.Dp(8),
	}.Layout(gtx, func(gtx C) D {
		return layout.Flex{
			Axis: layout.Vertical,
		}.Layout(gtx,
			// Search box
			layout.Rigid(func(gtx C) D {
				return layout.Inset{
					Top:    unit.Dp(4),
					Bottom: unit.Dp(8),
					Left:   unit.Dp(4),
					Right:  unit.Dp(4),
				}.Layout(gtx, func(gtx C) D {
					cb.searchInput.SingleLine = true
					cb.searchInput.Alignment = text.Start
					return cb.searchInput.Layout(gtx, th, i18n.Translate("Type a command or search..."))
				})
			}),

			// Divider line
			layout.Rigid(func(gtx C) D {
				rect := clip.Rect{
					Max: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Dp(unit.Dp(1))},
				}
				paint.FillShape(gtx.Ops, misc.WithAlpha(th.Fg, 0x30), rect.Op())
				return D{Size: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Dp(unit.Dp(1))}}
			}),

			// Commands list
			layout.Rigid(func(gtx C) D {
				if len(cb.filtered) == 0 {
					return layout.Inset{
						Top:    unit.Dp(16),
						Bottom: unit.Dp(16),
						Left:   unit.Dp(12),
						Right:  unit.Dp(12),
					}.Layout(gtx, func(gtx C) D {
						lbl := material.Label(th.Theme, th.TextSize*0.9, i18n.Translate("No matching commands"))
						lbl.Color = misc.WithAlpha(th.Fg, 0x80)
						return lbl.Layout(gtx)
					})
				}

				maxListHeight := min(gtx.Dp(unit.Dp(300)), len(cb.filtered)*gtx.Dp(unit.Dp(44)))
				gtx.Constraints.Max.Y = maxListHeight

				for len(cb.labels) < len(cb.filtered) {
					cb.labels = append(cb.labels, &widgets.InteractiveLabel{Focusable: true})
				}

				return cb.list.Layout(gtx, len(cb.filtered), func(gtx C, index int) D {
					item := cb.filtered[index]
					label := cb.labels[index]

					if label.Update(gtx) {
						cb.selectedIdx = index
						cb.executeSelected()
					}

					if index == cb.selectedIdx {
						label.Select()
					} else {
						label.Unselect()
					}

					return label.Layout(gtx, th, func(gtx C, textColor color.NRGBA) D {
						return layout.Inset{
							Top:    unit.Dp(6),
							Bottom: unit.Dp(6),
							Left:   unit.Dp(10),
							Right:  unit.Dp(10),
						}.Layout(gtx, func(gtx C) D {
							return layout.Flex{
								Axis:      layout.Horizontal,
								Alignment: layout.Middle,
							}.Layout(gtx,
								layout.Flexed(1, func(gtx C) D {
									return layout.Flex{
										Axis: layout.Vertical,
									}.Layout(gtx,
										layout.Rigid(func(gtx C) D {
											lbl := material.Label(th.Theme, th.TextSize*0.95, i18n.Translate(item.Title))
											lbl.Font.Weight = font.Medium
											lbl.Color = textColor
											return lbl.Layout(gtx)
										}),
										layout.Rigid(func(gtx C) D {
											if item.Subtitle == "" {
												return D{}
											}
											lbl := material.Label(th.Theme, th.TextSize*0.8, i18n.Translate(item.Subtitle))
											lbl.Color = misc.WithAlpha(textColor, 0x90)
											return lbl.Layout(gtx)
										}),
									)
								}),
								layout.Rigid(func(gtx C) D {
									if item.Shortcut == "" {
										return D{}
									}
									lbl := material.Label(th.Theme, th.TextSize*0.8, item.Shortcut)
									lbl.Color = misc.WithAlpha(textColor, 0x80)
									return lbl.Layout(gtx)
								}),
							)
						})
					})
				})
			}),
		)
	})
	callOp := macro.Stop()

	// Paint card background with shadow/border
	defer rrect.Push(gtx.Ops).Pop()
	paint.FillShape(gtx.Ops, th.Bg, clip.Rect(image.Rectangle{Max: dims.Size}).Op())
	event.Op(gtx.Ops, cb)
	callOp.Add(gtx.Ops)

	// Border
	paint.FillShape(gtx.Ops, misc.WithAlpha(th.ContrastBg, 0x60),
		clip.Stroke{
			Path:  clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NE: rr, SE: rr, NW: rr, SW: rr}.Path(gtx.Ops),
			Width: 1,
		}.Op(),
	)

	return dims
}
