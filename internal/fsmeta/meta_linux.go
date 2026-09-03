//go:build linux

package fsmeta

import (
	"fmt"
	"os"
	"syscall"
)

func read(_ string, info os.FileInfo) Metadata {
	result := Metadata{
		LogicalSize:   info.Size(),
		AllocatedSize: info.Size(),
		LinkCount:     1,
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return result
	}
	result.AllocatedSize = stat.Blocks * 512
	result.VolumeID = fmt.Sprintf("%x", uint64(stat.Dev))
	result.FileID = fmt.Sprintf("%x", stat.Ino)
	result.LinkCount = uint32(stat.Nlink)
	return result
}
