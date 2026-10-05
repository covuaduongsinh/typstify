package filechooser

import (
	"io"
	"time"

	"github.com/oligo/gioview/view"
)

// OpKind represents the type of file chooser operation.
type OpKind uint8

const (
	OpenFileOp OpKind = iota
	OpenFilesOp
	OpenFolderOp
	SaveFileOp
)

var (
	// FileChooserID is the view ID registered in ViewManager.
	FileChooserID = view.NewViewID("FileChooser")
)

// SortField indicates which column to sort by.
type SortField uint8

const (
	SortByName SortField = iota
	SortByDate
	SortBySize
)

// SortConfig holds sorting parameters.
type SortConfig struct {
	Field     SortField
	Ascending bool
}

// FileEntry represents a file or directory item.
type FileEntry struct {
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	ModTime time.Time
	Ext     string
}

// EntryFilter determines whether a file entry should be displayed.
// Note: Directories are usually kept so users can navigate into them.
type EntryFilter func(entry *FileEntry) bool

// Result holds the response from the file chooser dialog.
type Result struct {
	Paths []string
	Err   error
}

// ReadCloserWithName wraps an io.ReadCloser with its original file name.
type FileReadCloser struct {
	io.ReadCloser
	Name string
	Path string
}
