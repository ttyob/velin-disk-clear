//go:build windows

package fsmeta

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

type fileStandardInfo struct {
	AllocationSize int64
	EndOfFile      int64
	NumberOfLinks  uint32
	DeletePending  byte
	Directory      byte
}

func read(path string, info os.FileInfo) Metadata {
	result := Metadata{
		LogicalSize:   info.Size(),
		AllocatedSize: info.Size(),
		LinkCount:     1,
	}
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return result
	}
	handle, err := windows.CreateFile(
		pathPtr,
		windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return result
	}
	defer windows.CloseHandle(handle)

	var standard fileStandardInfo
	if err := windows.GetFileInformationByHandleEx(
		handle,
		windows.FileStandardInfo,
		(*byte)(unsafe.Pointer(&standard)),
		uint32(unsafe.Sizeof(standard)),
	); err == nil {
		result.AllocatedSize = standard.AllocationSize
		result.LinkCount = standard.NumberOfLinks
	}

	var identity windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &identity); err == nil {
		result.VolumeID = fmt.Sprintf("%08x", identity.VolumeSerialNumber)
		result.FileID = fmt.Sprintf("%08x%08x", identity.FileIndexHigh, identity.FileIndexLow)
		result.LinkCount = identity.NumberOfLinks
	}
	return result
}
