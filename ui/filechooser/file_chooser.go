package filechooser

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/oligo/gioview/view"
)

// FileChooser provides modal dialog helpers for choosing files and directories.
type FileChooser struct {
	vm         view.ViewManager
	resultChan chan Result
}

// NewFileChooser registers the file chooser view and returns a new FileChooser instance.
func NewFileChooser(vm view.ViewManager) (*FileChooser, error) {
	err := vm.Register(FileChooserID, NewFileChooserDialog)
	if err != nil {
		return nil, err
	}

	return &FileChooser{
		vm:         vm,
		resultChan: make(chan Result, 1),
	}, nil
}

// ChooseFolder opens the dialog allowing the user to select a folder.
// This is a blocking call and MUST be invoked from a separate goroutine.
func (fc *FileChooser) ChooseFolder() (string, error) {
	fc.show(OpenFolderOp, "", nil)

	resp := <-fc.resultChan
	if resp.Err != nil {
		return "", resp.Err
	}
	if len(resp.Paths) == 0 {
		return "", errors.New("no folder selected")
	}
	return resp.Paths[0], nil
}

// ChooseFile opens the dialog allowing the user to select a single file.
// Optionally pass file extensions to filter by (e.g. ".typ", ".pdf").
// This is a blocking call and MUST be invoked from a separate goroutine.
func (fc *FileChooser) ChooseFile(extensions ...string) (io.ReadCloser, error) {
	filter := createExtensionFilter(extensions...)
	fc.show(OpenFileOp, "", filter)

	resp := <-fc.resultChan
	if resp.Err != nil {
		return nil, resp.Err
	}
	if len(resp.Paths) == 0 {
		return nil, errors.New("no file selected")
	}

	return os.Open(resp.Paths[0])
}

// ChooseFiles opens the dialog allowing the user to select multiple files.
// Optionally pass file extensions to filter by (e.g. ".ttf", ".otf").
// This is a blocking call and MUST be invoked from a separate goroutine.
func (fc *FileChooser) ChooseFiles(extensions ...string) ([]io.ReadCloser, error) {
	filter := createExtensionFilter(extensions...)
	fc.show(OpenFilesOp, "", filter)

	resp := <-fc.resultChan
	if resp.Err != nil {
		return nil, resp.Err
	}
	if len(resp.Paths) == 0 {
		return nil, errors.New("no files selected")
	}

	readers := make([]io.ReadCloser, 0, len(resp.Paths))
	for _, p := range resp.Paths {
		f, err := os.Open(p)
		if err != nil {
			// Close any already opened files on error
			for _, r := range readers {
				_ = r.Close()
			}
			return nil, err
		}
		readers = append(readers, f)
	}

	return readers, nil
}

// CreateFile opens the dialog allowing the user to specify a save file location.
// This is a blocking call and MUST be invoked from a separate goroutine.
func (fc *FileChooser) CreateFile(name string) (io.WriteCloser, error) {
	fc.show(SaveFileOp, name, nil)

	resp := <-fc.resultChan
	if resp.Err != nil {
		return nil, resp.Err
	}
	if len(resp.Paths) == 0 {
		return nil, errors.New("no save path selected")
	}

	return os.Create(resp.Paths[0])
}

func (fc *FileChooser) show(op OpKind, filename string, filter EntryFilter) {
	params := map[string]any{
		"resultChan": fc.resultChan,
		"op":         op,
	}
	if filename != "" {
		params["filename"] = filename
	}
	if filter != nil {
		params["filter"] = filter
	}

	_ = fc.vm.RequestSwitch(view.Intent{
		Target:      FileChooserID,
		ShowAsModal: true,
		Params:      params,
	})
}

func createExtensionFilter(extensions ...string) EntryFilter {
	if len(extensions) == 0 {
		return nil
	}

	var validExts []string
	for _, ext := range extensions {
		cleaned := strings.ToLower(strings.TrimSpace(ext))
		if cleaned != "" {
			if !strings.HasPrefix(cleaned, ".") {
				cleaned = "." + cleaned
			}
			validExts = append(validExts, cleaned)
		}
	}

	return func(entry *FileEntry) bool {
		// Keep all directories so users can navigate inside them
		if entry.IsDir {
			return true
		}
		ext := strings.ToLower(filepath.Ext(entry.Name))
		return slices.Contains(validExts, ext)
	}
}
