//go:build !windows && !linux

package fsmeta

import "os"

func read(_ string, info os.FileInfo) Metadata {
	return Metadata{
		LogicalSize:   info.Size(),
		AllocatedSize: info.Size(),
		LinkCount:     1,
	}
}
