package filechooser

import (
	"os"
	"path/filepath"
)

// VolumeItem represents a disk partition or volume.
type VolumeItem struct {
	Label      string
	MountPoint string
	DriveType  string
}

// FavoriteItem represents a quick-access directory.
type FavoriteItem struct {
	Label string
	Path  string
	Icon  string // "home", "desktop", "documents", "downloads", "folder"
}

// GetFavorites returns common user directories that exist on the filesystem.
func GetFavorites() []*FavoriteItem {
	var favorites []*FavoriteItem

	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		favorites = append(favorites, &FavoriteItem{
			Label: "Home",
			Path:  home,
			Icon:  "home",
		})

		desktop := filepath.Join(home, "Desktop")
		if isDirectory(desktop) {
			favorites = append(favorites, &FavoriteItem{
				Label: "Desktop",
				Path:  desktop,
				Icon:  "desktop",
			})
		}

		documents := filepath.Join(home, "Documents")
		if isDirectory(documents) {
			favorites = append(favorites, &FavoriteItem{
				Label: "Documents",
				Path:  documents,
				Icon:  "documents",
			})
		}

		downloads := filepath.Join(home, "Downloads")
		if isDirectory(downloads) {
			favorites = append(favorites, &FavoriteItem{
				Label: "Downloads",
				Path:  downloads,
				Icon:  "downloads",
			})
		}
	}

	return favorites
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// DetectVolumes returns a list of mounted drives/volumes on the system.
func DetectVolumes() []*VolumeItem {
	return detectVolumes()
}
