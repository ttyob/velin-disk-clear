//go:build windows

package provider

import (
	"errors"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

type dpapiCredentials struct{ dir string }

func newCredentialStore(dataDir string) (credentialStore, error) {
	dir := filepath.Join(dataDir, "credentials")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &dpapiCredentials{dir: dir}, nil
}
func (s *dpapiCredentials) Save(id, secret string) error {
	input := []byte(secret)
	if len(input) == 0 {
		return errors.New("empty credential")
	}
	in := windows.DataBlob{Size: uint32(len(input)), Data: &input[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	encrypted := unsafe.Slice(out.Data, out.Size)
	return os.WriteFile(filepath.Join(s.dir, id+".bin"), encrypted, 0o600)
}
func (s *dpapiCredentials) Load(id string) (string, error) {
	encrypted, err := os.ReadFile(filepath.Join(s.dir, id+".bin"))
	if err != nil {
		return "", err
	}
	if len(encrypted) == 0 {
		return "", errors.New("empty encrypted credential")
	}
	in := windows.DataBlob{Size: uint32(len(encrypted)), Data: &encrypted[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return "", err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return string(unsafe.Slice(out.Data, out.Size)), nil
}
