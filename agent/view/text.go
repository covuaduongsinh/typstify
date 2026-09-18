package view

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/widgets/icons"
)

var (
	thoughtIcon = icons.NewSvgIcon(icons.LightBulb)
	planIcon    = icons.NewSvgIcon(icons.SquareCheck)
)

func layoutImageAttachments(gtx C, th *theme.Theme, images []ImageAttachment) D {
	if len(images) == 0 {
		return D{}
	}
	var children []layout.FlexChild
	for i := range images {
		imgAtt := images[i]
		children = append(children, layout.Rigid(func(gtx C) D {
			if imgAtt.Image == nil {
				return D{}
			}
			bounds := imgAtt.Image.Bounds()
			w := bounds.Dx()
			h := bounds.Dy()
			if w <= 0 || h <= 0 {
				return D{}
			}

			maxW := gtx.Dp(unit.Dp(360))
			maxH := gtx.Dp(unit.Dp(260))
			if maxW > gtx.Constraints.Max.X {
				maxW = gtx.Constraints.Max.X
			}

			scaleW := float32(maxW) / float32(w)
			scaleH := float32(maxH) / float32(h)
			scale := scaleW
			if scaleH < scale {
				scale = scaleH
			}
			if scale > 1.0 {
				scale = 1.0
			}

			dispW := int(float32(w) * scale)
			dispH := int(float32(h) * scale)

			rr := gtx.Dp(unit.Dp(6))
			rect := image.Rectangle{Max: image.Pt(dispW, dispH)}

			return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx C) D {
				defer clip.UniformRRect(rect, rr).Push(gtx.Ops).Pop()
				// Border
				defer clip.Stroke{
					Path:  clip.UniformRRect(rect, rr).Path(gtx.Ops),
					Width: float32(gtx.Dp(unit.Dp(1))),
				}.Op().Push(gtx.Ops).Pop()
				paint.Fill(gtx.Ops, misc.WithAlpha(th.Fg, 0x30))

				im := widget.Image{
					Src: imgAtt.ImageOp,
					Fit: widget.Contain,
				}
				gtx.Constraints = layout.Exact(image.Pt(dispW, dispH))
				return im.Layout(gtx)
			})
		}))
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx, children...)
}

type UserMsgStyle struct {
	selection widget.Selectable
}

func (u *UserMsgStyle) Layout(gtx C, th *theme.Theme, msg chatMessage) D {
	return layout.Inset{
		Top: unit.Dp(4), Bottom: unit.Dp(4),
		Left: unit.Dp(40),
	}.Layout(gtx, func(gtx C) D {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{
			Top: unit.Dp(6), Bottom: unit.Dp(6),
			Left: unit.Dp(8), Right: unit.Dp(8),
		}.Layout(gtx, func(gtx C) D {
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return layoutImageAttachments(gtx, th, msg.Images)
				}),
				layout.Rigid(func(gtx C) D {
					if msg.Content == "" {
						return D{}
					}
					label := material.Label(th.Theme, th.TextSize, msg.Content)
					label.Color = th.Fg
					label.State = &u.selection
					label.LineHeightScale = 1.5
					return label.Layout(gtx)
				}),
			)
		})
		call := macro.Stop()

		// Draw background behind the label.
		rect := clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(unit.Dp(6)))
		defer rect.Push(gtx.Ops).Pop()
		paint.FillShape(gtx.Ops, misc.WithAlpha(th.ContrastBg, 0x20), rect.Op(gtx.Ops))

		call.Add(gtx.Ops)
		return dims
	})
}

type AgentMsgStyle struct {
	mdBock    markdownBlock
	selection widget.Selectable
}

func (a *AgentMsgStyle) Layout(gtx C, th *theme.Theme, msg chatMessage) D {
	if msg.Content == "" && len(msg.Images) == 0 {
		return D{}
	}
	return layout.Inset{
		Top: unit.Dp(6), Bottom: unit.Dp(6),
	}.Layout(gtx, func(gtx C) D {
		return layout.Flex{
			Axis: layout.Vertical,
		}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layoutImageAttachments(gtx, th, msg.Images)
			}),
			layout.Rigid(func(gtx C) D {
				if msg.Content == "" {
					return D{}
				}
				return a.mdBock.Layout(gtx, th, []byte(msg.Content))
			}),
		)
	})
}

type ThoughtMsgStyle struct {
	mdBock    markdownBlock
	selection widget.Selectable
	card      CardStyle
}

func (t *ThoughtMsgStyle) Layout(gtx C, th *theme.Theme, msg chatMessage) D {
	if msg.Kind != msgThought || msg.Content == "" {
		return D{}
	}

	t.card.Icon = thoughtIcon

	return t.card.Layout(gtx, th,
		func(gtx C) D {
			return material.Label(th.Theme, th.TextSize, i18n.Translate("Thinking")).Layout(gtx)
		},
		func(gtx C) D {
			return t.mdBock.Layout(gtx, th, []byte(msg.Content))
		},
	)
}

type PlanMsgStyle struct {
	selection widget.Selectable
	card      CardStyle
}

func (p *PlanMsgStyle) Layout(gtx C, th *theme.Theme, msg chatMessage) D {
	p.card.Icon = planIcon
	return p.card.Layout(gtx, th,
		func(gtx C) D {
			return material.Label(th.Theme, th.TextSize, i18n.Translate("Plan")).Layout(gtx)
		},
		func(gtx C) D {
			label := material.Label(th.Theme, th.TextSize, msg.Content)
			label.LineHeightScale = 1.5
			label.State = &p.selection
			return label.Layout(gtx)
		},
	)
}
