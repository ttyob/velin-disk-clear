//go:build windows

package disks

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

func list() ([]Volume, error) {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil, err
	}
	systemDrive := strings.ToUpper(os.Getenv("SystemDrive"))
	volumes := make([]Volume, 0, 8)
	for index := 0; index < 26; index++ {
		if mask&(1<<index) == 0 {
			continue
		}
		mount := fmt.Sprintf("%c:\\", 'A'+index)
		mountPtr, err := windows.UTF16PtrFromString(mount)
		if err != nil {
			continue
		}
		if windows.GetDriveType(mountPtr) != windows.DRIVE_FIXED {
			continue
		}
		var freeToCaller, total, free uint64
		if err := windows.GetDiskFreeSpaceEx(mountPtr, &freeToCaller, &total, &free); err != nil {
			volumes = append(volumes, Volume{
				ID: mount, Name: strings.TrimSuffix(mount, "\\"), MountPoint: mount, Ready: false,
			})
			continue
		}
		drive := strings.TrimSuffix(mount, "\\")
		volumes = append(volumes, Volume{
			ID:         drive,
			Name:       drive,
			MountPoint: mount,
			FileSystem: "windows",
			TotalBytes: total,
			FreeBytes:  free,
			UsedBytes:  total - free,
			System:     strings.EqualFold(drive, systemDrive),
			Ready:      true,
		})
	}
	return volumes, nil
}
