//go:build !windows

package platform

import "fmt"

func QueryRecycleBin(root string) (RecycleBinInfo, error) {
	return RecycleBinInfo{}, fmt.Errorf("recycle bin query for %q is only available on Windows", root)
}

func EmptyRecycleBin(root string) error {
	return fmt.Errorf("recycle bin cleanup for %q is only available on Windows", root)
}
