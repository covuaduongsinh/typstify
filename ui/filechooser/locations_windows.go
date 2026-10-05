//go:build windows

package filechooser

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/windows"
)

func detectVolumes() []*VolumeItem {
	var volumes []*VolumeItem

	mask, err := windows.GetLogicalDrives()
	if err != nil {
		// Fallback to checking C: through Z: if bitmask retrieval fails
		mask = 0xFFFFFFFF
	}
	for i := range 26 {
		if (mask & (1 << i)) == 0 {
			continue
		}

		driveLetter := rune('A' + i)
		driveRoot := fmt.Sprintf("%c:\\", driveLetter)

		driveType := windows.GetDriveType(windows.StringToUTF16Ptr(driveRoot))
		if driveType == windows.DRIVE_NO_ROOT_DIR || driveType == windows.DRIVE_UNKNOWN {
			continue
		}

		driveTypeDesc := "Local Disk"
		switch driveType {
		case windows.DRIVE_REMOVABLE:
			driveTypeDesc = "Removable Disk"
		case windows.DRIVE_FIXED:
			driveTypeDesc = "Local Disk"
		case windows.DRIVE_REMOTE:
			driveTypeDesc = "Network Drive"
		case windows.DRIVE_CDROM:
			driveTypeDesc = "CD/DVD Drive"
		case windows.DRIVE_RAMDISK:
			driveTypeDesc = "RAM Disk"
		}

		var volNameBuf [256]uint16
		var fsNameBuf [256]uint16
		var serialNum, maxComponentLen, fsFlags uint32

		err = windows.GetVolumeInformation(
			windows.StringToUTF16Ptr(driveRoot),
			&volNameBuf[0],
			uint32(len(volNameBuf)),
			&serialNum,
			&maxComponentLen,
			&fsFlags,
			&fsNameBuf[0],
			uint32(len(fsNameBuf)),
		)

		var label string
		if err == nil {
			volName := syscall.UTF16ToString(volNameBuf[:])
			if volName != "" {
				label = fmt.Sprintf("%s (%c:)", volName, driveLetter)
			}
		}

		if label == "" {
			label = fmt.Sprintf("%s (%c:)", driveTypeDesc, driveLetter)
		}

		volumes = append(volumes, &VolumeItem{
			Label:      label,
			MountPoint: driveRoot,
			DriveType:  driveTypeDesc,
		})
	}

	return volumes
}
