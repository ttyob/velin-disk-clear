package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"time"

	"ai-clear/internal/platform"
	"ai-clear/internal/rules"
)

// scanUninstallRemnants is read-only. Registry entries whose install path has
// disappeared are useful leads, but are never converted into delete items.
func scanUninstallRemnants(ctx context.Context, state *jobState, rule rules.Rule) error {
	entries, err := platform.FindUninstallRemnants()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if ctx.Err() != nil {
			return context.Canceled
		}
		hash := sha256.Sum256([]byte(rule.ID + "\x00" + entry.RegistryPath))
		state.incrementScanned(entry.RegistryPath)
		state.addItem(Item{
			ID: hex.EncodeToString(hash[:12]), RuleID: rule.ID, MatchedRuleIDs: []string{rule.ID},
			Name: entry.DisplayName, Path: entry.RegistryPath, Directory: filepath.Dir(entry.RegistryPath),
			Category: rule.Category, Purpose: rule.Purpose, CleanEffect: rule.CleanEffect,
			Recommendation: string(rule.Recommendation), RecommendationReason: rule.RecommendationReason,
			Risk: string(rule.Risk), DefaultSelected: false, Selectable: false, Action: "analyze",
			RecoveryType: rule.RecoveryType, RequiresAdmin: rule.RequiresAdmin, HelpSummary: rule.Help.Summary,
			HelpDetails: rule.Help.Details, SpecialWarning: rule.Help.SpecialWarning, ManualSteps: rule.Help.ManualSteps,
			VolumeID: "registry", FileID: entry.RegistryPath, ModifiedAt: time.Now(),
		})
	}
	return nil
}
