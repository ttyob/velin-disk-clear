//go:build linux

package disks

import "syscall"

func list() ([]Volume, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return nil, err
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	return []Volume{{
		ID:         "dev-root",
		Name:       "开发环境",
		MountPoint: "/",
		FileSystem: "linux",
		TotalBytes: total,
		FreeBytes:  free,
		UsedBytes:  total - free,
		System:     true,
		Ready:      true,
	}}, nil
}
