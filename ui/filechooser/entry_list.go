package filechooser

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/dustin/go-humanize"
	"github.com/oligo/gioview/theme"
	"golang.org/x/exp/shiny/materialdesign/icons"
	"looz.ws/typstify/ui/uitokens"
	appicons "looz.ws/typstify/widgets/icons"
)

var (
	iconFolder, _   = widget.NewIcon(icons.FileFolder)
	iconFileText    = appicons.NewSvgIcon(appicons.FileText)
	iconFileCode    = appicons.NewSvgIcon(appicons.FileCode)
	iconFileImage   = appicons.NewSvgIcon(appicons.FileImage)
	iconFileType    = appicons.NewSvgIcon(appicons.FileType)
	iconFileBinary  = appicons.NewSvgIcon(appicons.FileBinary)
	iconArrowUp     = appicons.NewSvgIcon(appicons.ArrowUp)
	iconArrowDown   = appicons.NewSvgIcon(appicons.ArrowDown)
)

// ReadDirectory scans the target directory, applies filters and sorting rules.
func ReadDirectory(dir string, filter EntryFilter, sortConfig SortConfig, searchQuery string) ([]*FileEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var folders []*FileEntry
	var files []*FileEntry

	searchLower := strings.ToLower(searchQuery)

	for _, d := range entries {
		name := d.Name()

		// Skip hidden files unless search matches explicitly
		if strings.HasPrefix(name, ".") && searchLower == "" {
			continue
		}

		info, err := d.Info()
		if err != nil {
			continue
		}

		entry := &FileEntry{
			Name:    name,
			Path:    filepath.Join(dir, name),
			IsDir:   d.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Ext:     strings.ToLower(filepath.Ext(name)),
		}

		// Apply external filter
		if filter != nil && !filter(entry) {
			continue
		}

		// Apply search filter
		if searchLower != "" && !strings.Contains(strings.ToLower(name), searchLower) {
			continue
		}

		if entry.IsDir {
			folders = append(folders, entry)
		} else {
			files = append(files, entry)
		}
	}

	// Sort helper function
	sortSlice := func(items []*FileEntry) {
		sort.Slice(items, func(i, j int) bool {
			a, b := items[i], items[j]
			var cmp bool
			switch sortConfig.Field {
			case SortBySize:
				if a.Size == b.Size {
					cmp = strings.ToLower(a.Name) < strings.ToLower(b.Name)
				} else {
					cmp = a.Size < b.Size
				}
			case SortByDate:
				if a.ModTime.Equal(b.ModTime) {
					cmp = strings.ToLower(a.Name) < strings.ToLower(b.Name)
				} else {
					cmp = a.ModTime.Before(b.ModTime)
				}
			case SortByName:
				fallthrough
			default:
				cmp = strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}

			if !sortConfig.Ascending {
				return !cmp
			}
			return cmp
		})
	}

	sortSlice(folders)
	sortSlice(files)

	// Combine: Folders ALWAYS first
	result := make([]*FileEntry, 0, len(folders)+len(files))
	result = append(result, folders...)
	result = append(result, files...)

	return result, nil
}

// EntryListView manages the rendering and user interaction of directory contents.
type EntryListView struct {
	entries     []*FileEntry
	clickables  []widget.Clickable
	selected    map[int]bool
	lastClick   map[int]time.Time
	multiSelect bool

	sortConfig SortConfig
	list       widget.List

	nameSortBtn widget.Clickable
	dateSortBtn widget.Clickable
	sizeSortBtn widget.Clickable

	OnSelect      func(entry *FileEntry, selected []*FileEntry)
	OnOpenFolder  func(folderPath string)
	OnConfirmFile func(filePath string)
	OnSortChanged func(config SortConfig)
}

func NewEntryListView(multiSelect bool) *EntryListView {
	return &EntryListView{
		selected:    make(map[int]bool),
		lastClick:   make(map[int]time.Time),
		multiSelect: multiSelect,
		sortConfig: SortConfig{
			Field:     SortByName,
			Ascending: true,
		},
		list: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}
}

func (v *EntryListView) SetEntries(entries []*FileEntry) {
	v.entries = entries
	v.clickables = make([]widget.Clickable, len(entries))
	v.selected = make(map[int]bool)
	v.lastClick = make(map[int]time.Time)
}

func (v *EntryListView) SelectedEntries() []*FileEntry {
	var result []*FileEntry
	for idx := range v.entries {
		if v.selected[idx] {
			result = append(result, v.entries[idx])
		}
	}
	return result
}

func (v *EntryListView) ClearSelection() {
	v.selected = make(map[int]bool)
}

func (v *EntryListView) SelectEntryByPath(path string) {
	v.selected = make(map[int]bool)
	cleaned := filepath.Clean(path)
	for i, e := range v.entries {
		if filepath.Clean(e.Path) == cleaned {
			v.selected[i] = true
			break
		}
	}
}

func (v *EntryListView) Layout(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	// Handle column header clicks for sorting
	if v.nameSortBtn.Clicked(gtx) {
		v.toggleSort(SortByName)
	}
	if v.dateSortBtn.Clicked(gtx) {
		v.toggleSort(SortByDate)
	}
	if v.sizeSortBtn.Clicked(gtx) {
		v.toggleSort(SortBySize)
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		// Header Row
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return v.layoutHeader(gtx, th)
		}),
		// Divider
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				defer clip.Rect(image.Rectangle{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(1)))}).Push(gtx.Ops).Pop()
				paint.ColorOp{Color: uitokens.BorderColor(th)}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(1)))}
			})
		}),
		// Table Body or Empty State
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(v.entries) == 0 {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th.Theme, th.TextSize, "No files or folders found")
					lbl.Color = uitokens.SubtleTextColor(th)
					return lbl.Layout(gtx)
				})
			}
			return v.layoutList(gtx, th)
		}),
	)
}

func (v *EntryListView) toggleSort(field SortField) {
	if v.sortConfig.Field == field {
		v.sortConfig.Ascending = !v.sortConfig.Ascending
	} else {
		v.sortConfig.Field = field
		v.sortConfig.Ascending = true
	}
	if v.OnSortChanged != nil {
		v.OnSortChanged(v.sortConfig)
	}
}

func (v *EntryListView) layoutHeader(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	return layout.Inset{
		Left:   unit.Dp(12),
		Right:  unit.Dp(12),
		Top:    unit.Dp(6),
		Bottom: unit.Dp(6),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Alignment: layout.Middle,
		}.Layout(gtx,
			// Name column
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return v.layoutHeaderCol(gtx, th, &v.nameSortBtn, "Name", SortByName)
			}),
			// Date Modified column (fixed width)
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(150))
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(150))
				return v.layoutHeaderCol(gtx, th, &v.dateSortBtn, "Date Modified", SortByDate)
			}),
			// Size column (fixed width)
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(90))
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(90))
				return v.layoutHeaderCol(gtx, th, &v.sizeSortBtn, "Size", SortBySize)
			}),
		)
	})
}

func (v *EntryListView) layoutHeaderCol(gtx layout.Context, th *theme.Theme, clk *widget.Clickable, title string, field SortField) layout.Dimensions {
	isCurrent := v.sortConfig.Field == field

	return material.Clickable(gtx, clk, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Alignment: layout.Middle,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th.Theme, th.TextSize*0.9, title)
				lbl.Font.Weight = font.SemiBold
				if isCurrent {
					lbl.Color = th.Fg
				} else {
					lbl.Color = uitokens.SubtleTextColor(th)
				}
				return lbl.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !isCurrent {
					return layout.Dimensions{}
				}
				return layout.Inset{Left: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					if v.sortConfig.Ascending {
						return iconArrowUp.Layout(gtx, th.Fg, unit.Sp(12))
					}
					return iconArrowDown.Layout(gtx, th.Fg, unit.Sp(12))
				})
			}),
		)
	})
}

func (v *EntryListView) layoutList(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	if len(v.clickables) < len(v.entries) {
		v.clickables = make([]widget.Clickable, len(v.entries))
	}

	return material.List(th.Theme, &v.list).Layout(gtx, len(v.entries), func(gtx layout.Context, index int) layout.Dimensions {
		if index >= len(v.entries) {
			return layout.Dimensions{}
		}

		entry := v.entries[index]
		clk := &v.clickables[index]
		isSelected := v.selected[index]

		// Handle item clicks & double clicks
		if clk.Clicked(gtx) {
			now := gtx.Now
			last, hasLast := v.lastClick[index]
			v.lastClick[index] = now

			// Double-click check (< 400ms)
			if hasLast && now.Sub(last) < 400*time.Millisecond {
				if entry.IsDir {
					if v.OnOpenFolder != nil {
						v.OnOpenFolder(entry.Path)
					}
				} else {
					if v.OnConfirmFile != nil {
						v.OnConfirmFile(entry.Path)
					}
				}
			} else {
				// Single click selection
				if v.multiSelect {
					v.selected[index] = !v.selected[index]
				} else {
					v.selected = make(map[int]bool)
					v.selected[index] = true
				}

				if v.OnSelect != nil {
					v.OnSelect(entry, v.SelectedEntries())
				}
			}
		}

		return v.layoutRow(gtx, th, clk, entry, isSelected)
	})
}

func (v *EntryListView) layoutRow(gtx layout.Context, th *theme.Theme, clk *widget.Clickable, entry *FileEntry, isSelected bool) layout.Dimensions {
	return material.Clickable(gtx, clk, func(gtx layout.Context) layout.Dimensions {
		// Row background
		var bgColor color.NRGBA
		if isSelected {
			bgColor = th.ContrastBg
			bgColor.A = 40
		} else if clk.Hovered() {
			bgColor = uitokens.HoverBgColor(th)
		}

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
					Left:   unit.Dp(12),
					Right:  unit.Dp(12),
					Top:    unit.Dp(5),
					Bottom: unit.Dp(5),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						// Icon & Name column
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{
								Axis:      layout.Horizontal,
								Alignment: layout.Middle,
							}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layoutEntryIcon(gtx, th, entry)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									lbl := material.Label(th.Theme, th.TextSize, entry.Name)
									if isSelected {
										lbl.Font.Weight = font.Medium
									}
									return lbl.Layout(gtx)
								}),
							)
						}),
						// Date Modified column (fixed width)
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(150))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(150))
							dateStr := entry.ModTime.Format("2006-01-02 15:04")
							lbl := material.Label(th.Theme, th.TextSize*0.9, dateStr)
							lbl.Color = uitokens.SubtleTextColor(th)
							return lbl.Layout(gtx)
						}),
						// Size column (fixed width)
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(90))
							gtx.Constraints.Max.X = gtx.Dp(unit.Dp(90))
							sizeStr := "--"
							if !entry.IsDir {
								sizeStr = humanize.Bytes(uint64(entry.Size))
							}
							lbl := material.Label(th.Theme, th.TextSize*0.9, sizeStr)
							lbl.Color = uitokens.SubtleTextColor(th)
							return lbl.Layout(gtx)
						}),
					)
				})
			},
		)
	})
}

func layoutEntryIcon(gtx layout.Context, th *theme.Theme, entry *FileEntry) layout.Dimensions {
	if entry.IsDir {
		// Folder icon with warm amber/gold color
		folderColor := color.NRGBA{R: 245, G: 166, B: 35, A: 255}
		return iconFolder.Layout(gtx, folderColor)
	}

	switch entry.Ext {
	case ".typ":
		// Typst file: vibrant cyan
		typstColor := color.NRGBA{R: 35, G: 157, B: 173, A: 255}
		return iconFileText.Layout(gtx, typstColor, unit.Sp(16))
	case ".pdf":
		// PDF file: red
		pdfColor := color.NRGBA{R: 229, G: 57, B: 53, A: 255}
		return iconFileText.Layout(gtx, pdfColor, unit.Sp(16))
	case ".png", ".jpg", ".jpeg", ".svg", ".webp", ".gif":
		// Image file: green/emerald
		imgColor := color.NRGBA{R: 46, G: 125, B: 50, A: 255}
		return iconFileImage.Layout(gtx, imgColor, unit.Sp(16))
	case ".ttf", ".otf", ".ttc", ".woff", ".woff2":
		// Font file: purple
		fontColor := color.NRGBA{R: 142, G: 36, B: 170, A: 255}
		return iconFileType.Layout(gtx, fontColor, unit.Sp(16))
	case ".go", ".json", ".toml", ".yaml", ".yml", ".md", ".bib":
		// Code file: blue
		codeColor := color.NRGBA{R: 33, G: 150, B: 243, A: 255}
		return iconFileCode.Layout(gtx, codeColor, unit.Sp(16))
	default:
		// Generic file
		return iconFileBinary.Layout(gtx, uitokens.SubtleTextColor(th), unit.Sp(16))
	}
}
