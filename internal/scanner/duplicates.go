package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"

	"ai-clear/internal/rules"
)

type duplicateCandidate struct {
	path string
	info os.FileInfo
}

// scanDuplicateFiles first groups by size, then hashes only possible matches.
// It deliberately returns analysis-only items so the user can compare paths.
func scanDuplicateFiles(ctx context.Context, state *jobState, root string, rule rules.Rule) error {
	if excludedPath(root, state.request.ExcludeRoots) {
		return nil
	}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	bySize := make(map[int64][]duplicateCandidate)
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return context.Canceled
		}
		if walkErr != nil {
			state.addError(path, walkErr)
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path == root {
			return nil
		}
		if excludedPath(path, state.request.ExcludeRoots) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		state.incrementScanned(path)
		fileInfo, infoErr := entry.Info()
		if infoErr != nil {
			state.addError(path, infoErr)
			return nil
		}
		if matches(root, path, fileInfo, rule.Scan) {
			bySize[fileInfo.Size()] = append(bySize[fileInfo.Size()], duplicateCandidate{path: path, info: fileInfo})
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, candidates := range bySize {
		if len(candidates) < 2 {
			continue
		}
		byHash := make(map[string][]duplicateCandidate)
		for _, candidate := range candidates {
			digest, hashErr := fileDigest(ctx, candidate.path)
			if errors.Is(hashErr, context.Canceled) {
				return hashErr
			}
			if hashErr != nil {
				state.addError(candidate.path, hashErr)
				continue
			}
			byHash[digest] = append(byHash[digest], candidate)
		}
		for _, duplicates := range byHash {
			if len(duplicates) < 2 {
				continue
			}
			for _, duplicate := range duplicates {
				state.addItem(makeItem(duplicate.path, duplicate.info, rule))
			}
		}
	}
	return nil
}

func fileDigest(ctx context.Context, path string) (string, error) {
	if ctx.Err() != nil {
		return "", context.Canceled
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
