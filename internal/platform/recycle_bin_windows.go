//go:build windows

package platform

import (
	"fmt"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	shEmptyRecycleBinNoConfirmation = 0x00000001
	shEmptyRecycleBinNoProgressUI   = 0x00000002
	shEmptyRecycleBinNoSound        = 0x00000004
)

var (
	shell32                = windows.NewLazySystemDLL("shell32.dll")
	procSHQueryRecycleBinW = shell32.NewProc("SHQueryRecycleBinW")
	procSHEmptyRecycleBinW = shell32.NewProc("SHEmptyRecycleBinW")
)

type shQueryRecycleBinInfo struct {
	Size      uint32
	TotalSize int64
	ItemCount int64
}

func QueryRecycleBin(root string) (RecycleBinInfo, error) {
	volumeRoot, err := recycleBinVolumeRoot(root)
	if err != nil {
		return RecycleBinInfo{}, err
	}
	rootPointer, err := windows.UTF16PtrFromString(volumeRoot)
	if err != nil {
		return RecycleBinInfo{}, err
	}
	info := shQueryRecycleBinInfo{Size: uint32(unsafe.Sizeof(shQueryRecycleBinInfo{}))}
	result, _, _ := procSHQueryRecycleBinW.Call(uintptr(unsafe.Pointer(rootPointer)), uintptr(unsafe.Pointer(&info)))
	if int32(result) < 0 {
		return RecycleBinInfo{}, fmt.Errorf("query recycle bin for %s: HRESULT 0x%08x", volumeRoot, uint32(result))
	}
	return RecycleBinInfo{Root: volumeRoot, ItemCount: info.ItemCount, Size: info.TotalSize}, nil
}

func EmptyRecycleBin(root string) error {
	volumeRoot, err := recycleBinVolumeRoot(root)
	if err != nil {
		return err
	}
	rootPointer, err := windows.UTF16PtrFromString(volumeRoot)
	if err != nil {
		return err
	}
	flags := uintptr(shEmptyRecycleBinNoConfirmation | shEmptyRecycleBinNoProgressUI | shEmptyRecycleBinNoSound)
	result, _, _ := procSHEmptyRecycleBinW.Call(0, uintptr(unsafe.Pointer(rootPointer)), flags)
	if int32(result) < 0 {
		return fmt.Errorf("empty recycle bin for %s: HRESULT 0x%08x", volumeRoot, uint32(result))
	}
	return nil
}

func recycleBinVolumeRoot(root string) (string, error) {
	clean := filepath.Clean(root)
	volume := filepath.VolumeName(clean)
	if len(volume) != 2 || volume[1] != ':' {
		return "", fmt.Errorf("%q is not a fixed-volume path", root)
	}
	volumeRoot := volume + `\`
	if !filepath.IsAbs(volumeRoot) {
		return "", fmt.Errorf("%q is not an absolute volume path", root)
	}
	return volumeRoot, nil
}
