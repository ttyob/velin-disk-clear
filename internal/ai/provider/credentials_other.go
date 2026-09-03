//go:build !windows

package provider

import (
	"errors"
	"sync"
)

type memoryCredentials struct {
	mu     sync.RWMutex
	values map[string]string
}

func newCredentialStore(_ string) (credentialStore, error) {
	return &memoryCredentials{values: make(map[string]string)}, nil
}
func (s *memoryCredentials) Save(id, secret string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[id] = secret
	return nil
}
func (s *memoryCredentials) Load(id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.values[id]
	if !ok {
		return "", errors.New("credential not found")
	}
	return value, nil
}
