package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-clear/internal/fsmeta"
	"ai-clear/internal/rules"
)

type directoryUsage struct {
	logical   int64
	allocated int64
	files     int
	modified  time.Time
}

// scanLargeDirectories aggregates real file usage without retaining every file
// as a cleanup candidate. Directory results are analysis-only by design.
func scanLargeDirectories(ctx context.Context, state *jobState, root string, rule rules.Rule) error {
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
	usage := map[string]*directoryUsage{root: {}}
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
			if entry != nil && entry.IsDir() {
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
			usage[path] = &directoryUsage{}
			return nil
		}
		state.incrementScanned(path)
		fileInfo, infoErr := entry.Info()
		if infoErr != nil {
			state.addError(path, infoErr)
			return nil
		}
		metadata := fsmeta.Read(path, fileInfo)
		for directory := filepath.Dir(path); ; directory = filepath.Dir(directory) {
			stats := usage[directory]
			if stats == nil {
				stats = &directoryUsage{}
				usage[directory] = stats
			}
			stats.logical += metadata.LogicalSize
			stats.allocated += metadata.AllocatedSize
			stats.files++
			if fileInfo.ModTime().After(stats.modified) {
				stats.modified = fileInfo.ModTime()
			}
			if strings.EqualFold(directory, root) {
				break
			}
			parent := filepath.Dir(directory)
			if parent == directory {
				break
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	threshold := rule.Scan.MinDirectorySizeBytes
	for path, stats := range usage {
		if strings.EqualFold(path, root) || stats.files == 0 || stats.allocated < threshold {
			continue
		}
		directoryInfo, infoErr := os.Stat(path)
		if infoErr != nil {
			state.addError(path, infoErr)
			continue
		}
		state.addItem(makeDirectoryItem(path, directoryInfo, *stats, rule))
	}
	return nil
}

func makeDirectoryItem(path string, info os.FileInfo, stats directoryUsage, rule rules.Rule) Item {
	hash := sha256.Sum256([]byte(rule.ID + "\x00" + path))
	return Item{
		ID: hex.EncodeToString(hash[:12]), RuleID: rule.ID, MatchedRuleIDs: []string{rule.ID},
		Name: info.Name(), Path: path, Directory: filepath.Dir(path), IsDirectory: true,
		Category: rule.Category, Purpose: rule.Purpose, CleanEffect: rule.CleanEffect,
		Recommendation: string(rule.Recommendation), RecommendationReason: rule.RecommendationReason,
		Risk: string(rule.Risk), DefaultSelected: false, Selectable: false,
		Action: rule.Action.Type, RecoveryType: rule.RecoveryType, RequiresAdmin: rule.RequiresAdmin,
		RequiresRestart: rule.RequiresRestart, LogicalSize: stats.logical, AllocatedSize: stats.allocated,
		EstimatedReclaimable: 0, VolumeID: filepath.VolumeName(path), FileID: "directory",
		ModifiedAt: stats.modified, HelpSummary: rule.Help.Summary, HelpDetails: rule.Help.Details,
		SpecialWarning: rule.Help.SpecialWarning, ManualSteps: rule.Help.ManualSteps,
	}
}
