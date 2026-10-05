package filechooser

import (
	"path/filepath"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/theme"
	"looz.ws/typstify/ui/uitokens"
	"looz.ws/typstify/widgets/icons"
)

// BreadcrumbSegment represents one clickable component in the breadcrumbs bar.
type BreadcrumbSegment struct {
	Label    string
	FullPath string
	Click    widget.Clickable
}

var (
	chevronRightIcon = icons.NewSvgIcon(icons.ChevronRight)
)

// ParseBreadcrumbs splits a filesystem path into hierarchical clickable segments.
func ParseBreadcrumbs(path string) []*BreadcrumbSegment {
	cleaned := filepath.Clean(path)
	if cleaned == "" || cleaned == "." {
		return nil
	}

	var segments []*BreadcrumbSegment

	// 1. Check if path is Windows drive path (e.g. "C:\" or "C:\foo\bar" or "C:")
	vol := filepath.VolumeName(cleaned)
	if vol != "" {
		volRoot := vol + string(filepath.Separator)
		segments = append(segments, &BreadcrumbSegment{
			Label:    vol,
			FullPath: volRoot,
		})

		rest := strings.TrimPrefix(cleaned[len(vol):], string(filepath.Separator))
		if rest != "" {
			parts := strings.Split(rest, string(filepath.Separator))
			curr := volRoot
			for _, part := range parts {
				if part == "" {
					continue
				}
				curr = filepath.Join(curr, part)
				segments = append(segments, &BreadcrumbSegment{
					Label:    part,
					FullPath: curr,
				})
			}
		}
		return segments
	}

	// 2. Root directory path ("/" or "\")
	if cleaned == "/" || cleaned == `\` || path == "/" {
		return []*BreadcrumbSegment{
			{
				Label:    "/",
				FullPath: path,
			},
		}
	}

	// 3. Unix absolute path (e.g. "/home/user/docs")
	if strings.HasPrefix(path, "/") || strings.HasPrefix(cleaned, "/") || strings.HasPrefix(cleaned, `\`) {
		segments = append(segments, &BreadcrumbSegment{
			Label:    "/",
			FullPath: "/",
		})

		trimmed := strings.Trim(path, `/\`)
		if trimmed == "" {
			trimmed = strings.Trim(cleaned, `/\`)
		}
		parts := strings.FieldsFunc(trimmed, func(r rune) bool {
			return r == '/' || r == '\\'
		})
		curr := "/"
		for _, part := range parts {
			if part == "" {
				continue
			}
			if curr == "/" {
				curr = "/" + part
			} else {
				curr = curr + "/" + part
			}
			segments = append(segments, &BreadcrumbSegment{
				Label:    part,
				FullPath: curr,
			})
		}
		return segments
	}

	// 4. Relative path fallback
	trimmed := strings.Trim(cleaned, `/\`)
	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	curr := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if curr == "" {
			curr = part
		} else {
			curr = filepath.Join(curr, part)
		}
		segments = append(segments, &BreadcrumbSegment{
			Label:    part,
			FullPath: curr,
		})
	}

	return segments
}

// LayoutBreadcrumbs renders the breadcrumb segments horizontally.
func LayoutBreadcrumbs(gtx layout.Context, th *theme.Theme, segments []*BreadcrumbSegment, onSelect func(path string)) layout.Dimensions {
	if len(segments) == 0 {
		return layout.Dimensions{}
	}

	var children []layout.FlexChild

	for i, seg := range segments {
		s := seg
		isLast := (i == len(segments)-1)

		if s.Click.Clicked(gtx) && onSelect != nil {
			onSelect(s.FullPath)
		}

		// Breadcrumb button
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Clickable(gtx, &s.Click, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Left:   unit.Dp(6),
					Right:  unit.Dp(6),
					Top:    unit.Dp(3),
					Bottom: unit.Dp(3),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					label := material.Label(th.Theme, th.TextSize*0.95, s.Label)
					if isLast {
						label.Font.Weight = font.Bold
						label.Color = th.Fg
					} else {
						label.Color = uitokens.SubtleTextColor(th)
					}
					return label.Layout(gtx)
				})
			})
			return btn
		}))

		if !isLast {
			// Separator chevron
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Left:  unit.Dp(2),
					Right: unit.Dp(2),
					Top:   unit.Dp(4),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return chevronRightIcon.Layout(gtx, uitokens.SubtleTextColor(th), unit.Sp(12))
				})
			}))
		}
	}

	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx, children...)
}

