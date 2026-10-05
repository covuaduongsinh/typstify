package filechooser

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDetectVolumes(t *testing.T) {
	volumes := DetectVolumes()
	if len(volumes) == 0 {
		t.Fatal("expected at least one volume detected")
	}

	for _, v := range volumes {
		t.Logf("Detected volume: Label=%q, MountPoint=%q, DriveType=%q", v.Label, v.MountPoint, v.DriveType)

		if v.Label == "" {
			t.Errorf("volume label should not be empty: %+v", v)
		}
		if v.MountPoint == "" {
			t.Errorf("volume mount point should not be empty: %+v", v)
		}

		// Ensure Windows drives are not just single backslash "\"
		if runtime.GOOS == "windows" {
			if v.Label == "\\" || v.Label == "" {
				t.Errorf("invalid Windows volume label %q (must be like 'Local Disk (C:)' or 'Data (D:)')", v.Label)
			}
			if !strings.Contains(v.MountPoint, ":") {
				t.Errorf("expected Windows mount point to contain drive letter colon: %q", v.MountPoint)
			}
		}
	}
}

func TestGetFavorites(t *testing.T) {
	favs := GetFavorites()
	if len(favs) == 0 {
		t.Fatal("expected at least home directory in favorites")
	}

	for _, f := range favs {
		t.Logf("Favorite: Label=%q, Path=%q, Icon=%q", f.Label, f.Path, f.Icon)
		if f.Label == "" || f.Path == "" {
			t.Errorf("invalid favorite entry: %+v", f)
		}
		if !isDirectory(f.Path) {
			t.Errorf("favorite directory should exist: %q", f.Path)
		}
	}
}

func TestParseBreadcrumbs(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		expectedLen   int
		expectedFirst string
		expectedLast  string
	}{
		{
			name:          "Windows drive root",
			path:          `C:\`,
			expectedLen:   1,
			expectedFirst: "C:",
			expectedLast:  "C:",
		},
		{
			name:          "Windows nested directory",
			path:          `D:\code\typstify`,
			expectedLen:   3,
			expectedFirst: "D:",
			expectedLast:  "typstify",
		},
		{
			name:          "Unix root",
			path:          "/",
			expectedLen:   1,
			expectedFirst: "/",
			expectedLast:  "/",
		},
		{
			name:          "Unix nested path",
			path:          "/home/user/projects/typstify",
			expectedLen:   5,
			expectedFirst: "/",
			expectedLast:  "typstify",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segs := ParseBreadcrumbs(tt.path)
			if len(segs) != tt.expectedLen {
				t.Fatalf("ParseBreadcrumbs(%q) got %d segments, want %d", tt.path, len(segs), tt.expectedLen)
			}
			if segs[0].Label != tt.expectedFirst {
				t.Errorf("first segment got %q, want %q", segs[0].Label, tt.expectedFirst)
			}
			if segs[len(segs)-1].Label != tt.expectedLast {
				t.Errorf("last segment got %q, want %q", segs[len(segs)-1].Label, tt.expectedLast)
			}
		})
	}
}

func TestHistoryStack(t *testing.T) {
	h := NewHistoryStack()

	if h.CanBack() || h.CanForward() {
		t.Error("new history stack should not have back or forward history")
	}

	h.Push("/dir1")
	h.Push("/dir2")
	h.Push("/dir3")

	if !h.CanBack() {
		t.Error("expected CanBack() == true")
	}

	prev := h.Back()
	if filepath.Clean(prev) != filepath.Clean("/dir2") {
		t.Errorf("expected Back() to return /dir2, got %q", prev)
	}

	prev = h.Back()
	if filepath.Clean(prev) != filepath.Clean("/dir1") {
		t.Errorf("expected Back() to return /dir1, got %q", prev)
	}

	if h.CanBack() {
		t.Error("expected CanBack() == false at history start")
	}

	next := h.Forward()
	if filepath.Clean(next) != filepath.Clean("/dir2") {
		t.Errorf("expected Forward() to return /dir2, got %q", next)
	}

	// Test Up navigation
	parent := h.Up("/home/user/docs")
	if filepath.Clean(parent) != filepath.Clean("/home/user") {
		t.Errorf("expected Up() to return /home/user, got %q", parent)
	}
}

func TestReadDirectoryAndSorting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "typstify-fc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test structure:
	// - folders: "z_folder", "a_folder"
	// - files: "b_file.typ", "a_file.pdf", "c_file.txt"
	_ = os.Mkdir(filepath.Join(tmpDir, "z_folder"), 0755)
	_ = os.Mkdir(filepath.Join(tmpDir, "a_folder"), 0755)

	_ = os.WriteFile(filepath.Join(tmpDir, "b_file.typ"), []byte("typst content"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "a_file.pdf"), []byte("pdf"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "c_file.txt"), []byte("txt long content here"), 0644)

	// 1. Test Sort by Name ASC (folders should be first, then files sorted by name)
	entries, err := ReadDirectory(tmpDir, nil, SortConfig{Field: SortByName, Ascending: true}, "")
	if err != nil {
		t.Fatalf("ReadDirectory failed: %v", err)
	}

	if len(entries) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(entries))
	}

	// Folders should always be first
	if !entries[0].IsDir || entries[0].Name != "a_folder" {
		t.Errorf("entry 0 should be a_folder, got %s (isDir=%v)", entries[0].Name, entries[0].IsDir)
	}
	if !entries[1].IsDir || entries[1].Name != "z_folder" {
		t.Errorf("entry 1 should be z_folder, got %s (isDir=%v)", entries[1].Name, entries[1].IsDir)
	}

	// Files come after folders, sorted by name
	if entries[2].IsDir || entries[2].Name != "a_file.pdf" {
		t.Errorf("entry 2 should be a_file.pdf, got %s (isDir=%v)", entries[2].Name, entries[2].IsDir)
	}
	if entries[3].IsDir || entries[3].Name != "b_file.typ" {
		t.Errorf("entry 3 should be b_file.typ, got %s (isDir=%v)", entries[3].Name, entries[3].IsDir)
	}
	if entries[4].IsDir || entries[4].Name != "c_file.txt" {
		t.Errorf("entry 4 should be c_file.txt, got %s (isDir=%v)", entries[4].Name, entries[4].IsDir)
	}

	// 2. Test Search Query filtering
	filtered, err := ReadDirectory(tmpDir, nil, SortConfig{Field: SortByName, Ascending: true}, "typ")
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filtered[0].Name != "b_file.typ" {
		t.Errorf("expected only b_file.typ to match search 'typ', got %+v", filtered)
	}

	// 3. Test Extension Filtering
	extFilter := createExtensionFilter(".typ", ".pdf")
	extEntries, err := ReadDirectory(tmpDir, extFilter, SortConfig{Field: SortByName, Ascending: true}, "")
	if err != nil {
		t.Fatal(err)
	}

	// Should contain both folders ("a_folder", "z_folder") and 2 matching files ("a_file.pdf", "b_file.typ")
	if len(extEntries) != 4 {
		t.Fatalf("expected 4 entries with extension filter, got %d", len(extEntries))
	}

	// 4. Test Sort by Size DESC (Folders still first, then largest files)
	sizeDescEntries, err := ReadDirectory(tmpDir, nil, SortConfig{Field: SortBySize, Ascending: false}, "")
	if err != nil {
		t.Fatal(err)
	}
	// Folders are still entries 0 and 1
	if !sizeDescEntries[0].IsDir || !sizeDescEntries[1].IsDir {
		t.Errorf("folders must remain first when sorting by size: %+v", sizeDescEntries)
	}
	// Files sorted by size descending: c_file.txt (~22b) > b_file.typ (~13b) > a_file.pdf (~3b)
	if sizeDescEntries[2].Name != "c_file.txt" || sizeDescEntries[3].Name != "b_file.typ" || sizeDescEntries[4].Name != "a_file.pdf" {
		t.Errorf("files should be sorted by size DESC: got %s, %s, %s",
			sizeDescEntries[2].Name, sizeDescEntries[3].Name, sizeDescEntries[4].Name)
	}

	// 5. Test Sort by Name DESC (Folders still first, sorted Z-A)
	nameDescEntries, err := ReadDirectory(tmpDir, nil, SortConfig{Field: SortByName, Ascending: false}, "")
	if err != nil {
		t.Fatal(err)
	}
	if nameDescEntries[0].Name != "z_folder" || nameDescEntries[1].Name != "a_folder" {
		t.Errorf("folders should be sorted Z-A: got %s, %s", nameDescEntries[0].Name, nameDescEntries[1].Name)
	}
	if nameDescEntries[2].Name != "c_file.txt" || nameDescEntries[3].Name != "b_file.typ" || nameDescEntries[4].Name != "a_file.pdf" {
		t.Errorf("files should be sorted Z-A: got %s, %s, %s",
			nameDescEntries[2].Name, nameDescEntries[3].Name, nameDescEntries[4].Name)
	}
}

func TestExtensionFilter(t *testing.T) {
	filter := createExtensionFilter(".ttf", ".otf")

	// Directory should always pass
	folderEntry := &FileEntry{Name: "fonts", IsDir: true}
	if !filter(folderEntry) {
		t.Error("extension filter should keep directories")
	}

	// Matching file
	ttfEntry := &FileEntry{Name: "Roboto.ttf", IsDir: false}
	if !filter(ttfEntry) {
		t.Error("extension filter should allow .ttf")
	}

	// Non-matching file
	txtEntry := &FileEntry{Name: "readme.txt", IsDir: false}
	if filter(txtEntry) {
		t.Error("extension filter should exclude .txt")
	}
}
