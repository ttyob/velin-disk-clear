//go:build linux

package fsmeta

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadDetectsHardLinks(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.bin")
	second := filepath.Join(root, "second.bin")
	if err := os.WriteFile(first, make([]byte, 8192), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(first, second); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	metadata := Read(first, info)
	if metadata.LinkCount < 2 {
		t.Fatalf("link count = %d, want at least 2", metadata.LinkCount)
	}
	if metadata.FileID == "" || metadata.VolumeID == "" {
		t.Fatalf("missing file identity: %#v", metadata)
	}
	if metadata.AllocatedSize <= 0 {
		t.Fatalf("allocated size = %d, want positive", metadata.AllocatedSize)
	}
}
