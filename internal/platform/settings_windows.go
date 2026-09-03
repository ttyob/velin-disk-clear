//go:build windows

package platform

import (
	"fmt"
	"os/exec"
)

func OpenSettings(action string) error {
	switch action {
	case "virtual_memory":
		return exec.Command("SystemPropertiesAdvanced.exe").Start()
	case "system_protection":
		return exec.Command("SystemPropertiesProtection.exe").Start()
	default:
		return fmt.Errorf("unsupported system settings action %q", action)
	}
}
