//go:build !windows

package updater

import "errors"

func installUpdate(_, _ string, _ int) error {
	return errors.New("automatic updates are only supported on Windows")
}
