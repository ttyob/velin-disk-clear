//go:build !windows

package platform

type UninstallRemnant struct {
	DisplayName  string
	RegistryPath string
}

func FindUninstallRemnants() ([]UninstallRemnant, error) { return []UninstallRemnant{}, nil }
