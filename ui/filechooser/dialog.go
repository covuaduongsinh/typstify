package filechooser

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"strings"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/x/component"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"
	"looz.ws/typstify/ui/uitokens"
)

// FileChooserDialog is the modal dialog view for browsing and selecting files/folders.
type FileChooserDialog struct {
	*view.BaseView

	resultChan chan Result
	op         OpKind
	filter     EntryFilter

	currentPath string
	history     *HistoryStack

	sidebar       *Sidebar
	navBar        *NavigationBar
	entryList     *EntryListView
	bottomBar     *BottomBar
	resizer       *component.Resize

	searchQuery string
	resultSent  bool
}

func NewFileChooserDialog() view.View {
	history := NewHistoryStack()

	home, _ := os.UserHomeDir()
	if home == "" {
		home = "."
	}

	dlg := &FileChooserDialog{
		BaseView:    &view.BaseView{},
		history:     history,
		sidebar:     NewSidebar(),
		navBar:      NewNavigationBar(history),
		entryList:   NewEntryListView(false),
		bottomBar:   NewBottomBar(OpenFileOp),
		resizer:     &component.Resize{Axis: layout.Horizontal, Ratio: 0.22},
		currentPath: home,
	}

	dlg.setupCallbacks()
	return dlg
}

func (d *FileChooserDialog) ID() view.ViewID {
	return FileChooserID
}

func (d *FileChooserDialog) Title() string {
	switch d.op {
	case OpenFileOp:
		return "Open File"
	case OpenFilesOp:
		return "Open Files"
	case OpenFolderOp:
		return "Open Folder"
	case SaveFileOp:
		return "Save File"
	default:
		return "File Chooser"
	}
}

func (d *FileChooserDialog) OnNavTo(intent view.Intent) error {
	d.BaseView.OnNavTo(intent)
	d.resultSent = false

	rc, ok := intent.Params["resultChan"]
	if !ok {
		return errors.New("missing resultChan param")
	}
	d.resultChan = rc.(chan Result)

	if opVal, ok := intent.Params["op"]; ok {
		d.op = opVal.(OpKind)
	} else {
		d.op = OpenFileOp
	}

	d.entryList.multiSelect = (d.op == OpenFilesOp)
	d.bottomBar.SetOp(d.op)

	if filterVal, ok := intent.Params["filter"]; ok && filterVal != nil {
		d.filter = filterVal.(EntryFilter)
	} else {
		d.filter = nil
	}

	if filename, ok := intent.Params["filename"]; ok && filename != nil {
		d.bottomBar.SetSaveFileName(filename.(string))
	}

	startPath := d.currentPath
	if initDir, ok := intent.Params["initialDir"]; ok && initDir != nil {
		if pathStr, ok := initDir.(string); ok && pathStr != "" {
			if info, err := os.Stat(pathStr); err == nil && info.IsDir() {
				startPath = pathStr
			}
		}
	}

	d.navigateTo(startPath, true)
	return nil
}

func (d *FileChooserDialog) OnFinish() {
	d.BaseView.OnFinish()
	if !d.resultSent && d.resultChan != nil {
		d.resultSent = true
		d.resultChan <- Result{Err: errors.New("operation cancelled by user")}
	}
}

func (d *FileChooserDialog) setupCallbacks() {
	d.sidebar.OnNavigate = func(path string) {
		d.navigateTo(path, true)
	}

	d.navBar.OnNavigate = func(path string) {
		d.navigateTo(path, true)
	}

	d.navBar.OnRefresh = func() {
		d.reloadCurrentDirectory()
	}

	d.navBar.OnSearch = func(query string) {
		d.searchQuery = query
		d.reloadCurrentDirectory()
	}

	d.entryList.OnSelect = func(entry *FileEntry, selected []*FileEntry) {
		if d.op == OpenFilesOp {
			var names []string
			for _, e := range selected {
				names = append(names, e.Name)
			}
			d.bottomBar.SetSelectedText(strings.Join(names, ", "))
		} else if entry != nil {
			d.bottomBar.SetSelectedText(entry.Name)
			if d.op == SaveFileOp && !entry.IsDir {
				d.bottomBar.SetSaveFileName(entry.Name)
			}
		} else {
			d.bottomBar.SetSelectedText("")
		}
	}

	d.entryList.OnOpenFolder = func(folderPath string) {
		d.navigateTo(folderPath, true)
	}

	d.entryList.OnConfirmFile = func(filePath string) {
		d.confirmResult([]string{filePath})
	}

	d.entryList.OnSortChanged = func(config SortConfig) {
		d.reloadCurrentDirectory()
	}

	d.bottomBar.OnCancel = func() {
		d.OnFinish()
	}

	d.bottomBar.OnConfirm = func() error {
		return d.handleConfirmAction()
	}

	d.bottomBar.OnNewFolder = func(folderName string) error {
		d.reloadCurrentDirectory()
		newFolderPath := filepath.Join(d.currentPath, folderName)
		d.entryList.SelectEntryByPath(newFolderPath)
		return nil
	}
}

func (d *FileChooserDialog) navigateTo(path string, pushHistory bool) {
	cleaned := filepath.Clean(path)
	info, err := os.Stat(cleaned)
	if err != nil || !info.IsDir() {
		return
	}

	d.currentPath = cleaned
	if pushHistory {
		d.history.Push(cleaned)
	}

	d.navBar.SetCurrentPath(cleaned)
	d.sidebar.SetCurrentPath(cleaned)
	d.bottomBar.SetCurrentPath(cleaned)
	d.bottomBar.SetSelectedText("")

	d.reloadCurrentDirectory()
}

func (d *FileChooserDialog) reloadCurrentDirectory() {
	entries, err := ReadDirectory(d.currentPath, d.filter, d.entryList.sortConfig, d.searchQuery)
	if err != nil {
		d.bottomBar.SetError(err)
		return
	}
	d.entryList.SetEntries(entries)
}

func (d *FileChooserDialog) handleConfirmAction() error {
	switch d.op {
	case SaveFileOp:
		filename := d.bottomBar.SaveFileName()
		if filename == "" {
			return errors.New("Please enter a file name")
		}
		targetPath := filepath.Join(d.currentPath, filename)
		d.confirmResult([]string{targetPath})
		return nil

	case OpenFolderOp:
		selected := d.entryList.SelectedEntries()
		if len(selected) > 0 && selected[0].IsDir {
			d.confirmResult([]string{selected[0].Path})
		} else {
			// If no specific folder item is selected, use the current active directory
			d.confirmResult([]string{d.currentPath})
		}
		return nil

	case OpenFileOp:
		selected := d.entryList.SelectedEntries()
		if len(selected) == 0 {
			return errors.New("Please select a file to open")
		}
		if selected[0].IsDir {
			// If user clicked open on a folder, navigate into it instead
			d.navigateTo(selected[0].Path, true)
			return nil
		}
		d.confirmResult([]string{selected[0].Path})
		return nil

	case OpenFilesOp:
		selected := d.entryList.SelectedEntries()
		if len(selected) == 0 {
			return errors.New("Please select one or more files")
		}
		var paths []string
		for _, e := range selected {
			if !e.IsDir {
				paths = append(paths, e.Path)
			}
		}
		if len(paths) == 0 {
			return errors.New("No valid files selected")
		}
		d.confirmResult(paths)
		return nil
	}

	return nil
}

func (d *FileChooserDialog) confirmResult(paths []string) {
	if !d.resultSent && d.resultChan != nil {
		d.resultSent = true
		d.resultChan <- Result{Paths: paths}
	}
	d.OnFinish()
}

func (d *FileChooserDialog) Layout(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	// Handle modal keyboard shortcuts
	for {
		ke, ok := gtx.Event(
			key.FocusFilter{Target: d},
			key.Filter{Focus: d, Name: key.NameEscape},
			key.Filter{Focus: d, Name: key.NameReturn},
			key.Filter{Focus: d, Name: key.NameEnter},
			key.Filter{Focus: d, Name: "F5"},
		)
		if !ok {
			break
		}
		if ev, ok := ke.(key.Event); ok && (ev.State == key.Press || ev.State == key.Release) {
			if ev.Name == key.NameEscape {
				d.OnFinish()
				gtx.Execute(op.InvalidateCmd{})
				return layout.Dimensions{}
			}
			if ev.State == key.Release {
				if ev.Name == key.NameEnter || ev.Name == key.NameReturn {
					_ = d.handleConfirmAction()
					gtx.Execute(op.InvalidateCmd{})
				} else if ev.Name == "F5" {
					d.reloadCurrentDirectory()
					gtx.Execute(op.InvalidateCmd{})
				}
			}
		}
	}

	return d.resizer.Layout(gtx,
		// Left: Sidebar (Favorites & Locations)
		func(gtx layout.Context) layout.Dimensions {
			return layout.Background{}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					bg := th.Bg
					defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()
					paint.ColorOp{Color: bg}.Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)
					return layout.Dimensions{Size: gtx.Constraints.Min}
				},
				func(gtx layout.Context) layout.Dimensions {
					return d.sidebar.Layout(gtx, th)
				},
			)
		},
		// Right: Main Area (Navigation Bar + Entry List + Bottom Bar)
		func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Left:   unit.Dp(12),
				Right:  unit.Dp(12),
				Top:    unit.Dp(10),
				Bottom: unit.Dp(10),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					// Header: Navigation controls + Breadcrumbs + Search
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return d.navBar.Layout(gtx, th)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					// Body: Table of files and folders
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return d.entryList.Layout(gtx, th)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					// Divider above Bottom Bar
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						defer clip.Rect(image.Rectangle{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(1)))}).Push(gtx.Ops).Pop()
						paint.ColorOp{Color: uitokens.BorderColor(th)}.Add(gtx.Ops)
						paint.PaintOp{}.Add(gtx.Ops)
						return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(1)))}
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					// Footer: Bottom Bar (New Folder, Selection info, Action buttons)
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return d.bottomBar.Layout(gtx, th)
					}),
				)
			})
		},
		// Divider handle styling
		func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(1))
			gtx.Constraints.Max.X = gtx.Dp(unit.Dp(1))
			defer clip.Rect(image.Rectangle{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}).Push(gtx.Ops).Pop()
			paint.ColorOp{Color: uitokens.BorderColor(th)}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}
		},
	)
}
