//go:build windows

package platform

import (
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

type UninstallRemnant struct {
	DisplayName  string
	RegistryPath string
}

// FindUninstallRemnants inspects both native and WOW64 uninstall views. It
// reports only entries with a non-empty install path that no longer exists.
func FindUninstallRemnants() ([]UninstallRemnant, error) {
	targets := []struct {
		root registry.Key
		path string
	}{
		{registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `Software\Microsoft\Windows\CurrentVersion\Uninstall`},
	}
	seen := make(map[string]struct{})
	result := make([]UninstallRemnant, 0)
	for _, target := range targets {
		for _, access := range []uint32{registry.READ, registry.WOW64_32KEY, registry.WOW64_64KEY} {
			key, err := registry.OpenKey(target.root, target.path, access)
			if err != nil {
				continue
			}
			names, _ := key.ReadSubKeyNames(-1)
			for _, name := range names {
				child, childErr := registry.OpenKey(key, name, registry.READ)
				if childErr != nil {
					continue
				}
				display, _, _ := child.GetStringValue("DisplayName")
				install, _, _ := child.GetStringValue("InstallLocation")
				_ = child.Close()
				install = strings.TrimSpace(install)
				if display == "" || install == "" || pathExists(install) {
					continue
				}
				registryPath := target.path + `\` + name
				if _, exists := seen[registryPath]; exists {
					continue
				}
				seen[registryPath] = struct{}{}
				result = append(result, UninstallRemnant{DisplayName: display, RegistryPath: registryPath})
			}
			_ = key.Close()
		}
	}
	return result, nil
}
