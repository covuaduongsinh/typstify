//go:build !windows

package filechooser

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

func detectVolumes() []*VolumeItem {
	var volumes []*VolumeItem

	partitions, err := disk.Partitions(false)
	var warnings *disk.Warnings
	if err != nil && !errors.As(err, &warnings) {
		// Fallback to at least returning root directory
		return []*VolumeItem{
			{
				Label:      "Root (/)",
				MountPoint: "/",
				DriveType:  "Root",
			},
		}
	}

	seen := make(map[string]bool)

	for _, p := range partitions {
		if slices.Contains(p.Opts, "nobrowse") || seen[p.Mountpoint] {
			continue
		}

		// Filter out system virtual filesystems on Linux
		if strings.HasPrefix(p.Mountpoint, "/dev") ||
			strings.HasPrefix(p.Mountpoint, "/proc") ||
			strings.HasPrefix(p.Mountpoint, "/sys") ||
			strings.HasPrefix(p.Mountpoint, "/run") {
			continue
		}

		seen[p.Mountpoint] = true

		label := ""
		if p.Mountpoint == "/" {
			label = "Root (/)"
		} else {
			base := filepath.Base(p.Mountpoint)
			if base != "" && base != "/" && base != "." {
				label = fmt.Sprintf("%s (%s)", base, p.Mountpoint)
			} else {
				label = p.Mountpoint
			}
		}

		driveType := "Disk"
		if strings.Contains(p.Fstype, "ext") || strings.Contains(p.Fstype, "apfs") || strings.Contains(p.Fstype, "btrfs") || strings.Contains(p.Fstype, "zfs") {
			driveType = "Local Disk"
		} else if strings.Contains(p.Fstype, "nfs") || strings.Contains(p.Fstype, "smb") || strings.Contains(p.Fstype, "cifs") {
			driveType = "Network Volume"
		}

		volumes = append(volumes, &VolumeItem{
			Label:      label,
			MountPoint: p.Mountpoint,
			DriveType:  driveType,
		})
	}

	if len(volumes) == 0 {
		volumes = append(volumes, &VolumeItem{
			Label:      "Root (/)",
			MountPoint: "/",
			DriveType:  "Root",
		})
	}

	return volumes
}
