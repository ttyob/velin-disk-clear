//go:build !windows

package platform

import "fmt"

func OpenSettings(action string) error {
	return fmt.Errorf("system settings action %q is only available on Windows", action)
}
