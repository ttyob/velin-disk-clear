package rules

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

//go:embed builtin/*.json
var builtinFS embed.FS

var ruleIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z0-9_]+)+$`)

type Service struct {
	rules []Rule
}

func LoadBuiltin() (*Service, error) {
	entries, err := fs.Glob(builtinFS, "builtin/*.json")
	if err != nil {
		return nil, fmt.Errorf("list builtin rules: %w", err)
	}
	if len(entries) == 0 {
		return nil, errors.New("no builtin rules found")
	}

	var loaded []Rule
	seen := make(map[string]struct{})
	for _, entry := range entries {
		data, err := builtinFS.ReadFile(entry)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry, err)
		}
		var catalog Catalog
		if err := json.Unmarshal(data, &catalog); err != nil {
			return nil, fmt.Errorf("decode %s: %w", entry, err)
		}
		for i := range catalog.Rules {
			rule := catalog.Rules[i]
			if err := Validate(rule); err != nil {
				return nil, fmt.Errorf("%s: %w", entry, err)
			}
			if _, exists := seen[rule.ID]; exists {
				return nil, fmt.Errorf("duplicate rule id %q", rule.ID)
			}
			seen[rule.ID] = struct{}{}
			loaded = append(loaded, rule)
		}
	}
	sort.Slice(loaded, func(i, j int) bool {
		if loaded[i].Category == loaded[j].Category {
			return loaded[i].Name < loaded[j].Name
		}
		return loaded[i].Category < loaded[j].Category
	})
	return &Service{rules: loaded}, nil
}

func Validate(rule Rule) error {
	if !ruleIDPattern.MatchString(rule.ID) {
		return fmt.Errorf("invalid rule id %q", rule.ID)
	}
	if rule.Version < 1 {
		return fmt.Errorf("rule %s has invalid version", rule.ID)
	}
	required := map[string]string{
		"name": rule.Name, "description": rule.Description, "purpose": rule.Purpose,
		"clean_effect": rule.CleanEffect, "recommendation_reason": rule.RecommendationReason,
		"category": rule.Category, "scope": rule.Scope, "size_mode": rule.SizeMode,
		"recovery_type": rule.RecoveryType, "last_verified_at": rule.LastVerifiedAt,
		"source": rule.Source, "help.summary": rule.Help.Summary, "action.type": rule.Action.Type,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("rule %s is missing %s", rule.ID, field)
		}
	}
	if len(rule.Modes) == 0 {
		return fmt.Errorf("rule %s has no scan mode", rule.ID)
	}
	if len(rule.Scan.Roots) == 0 {
		return fmt.Errorf("rule %s has no scan roots", rule.ID)
	}
	if len(rule.Safety.AllowedRoots) == 0 {
		return fmt.Errorf("rule %s has no allowed roots", rule.ID)
	}
	validRisk := rule.Risk == RiskLow || rule.Risk == RiskMedium || rule.Risk == RiskHigh || rule.Risk == RiskForbidden
	if !validRisk {
		return fmt.Errorf("rule %s has invalid risk %q", rule.ID, rule.Risk)
	}
	validRecommendation := rule.Recommendation == RecommendationRecommended ||
		rule.Recommendation == RecommendationOptional ||
		rule.Recommendation == RecommendationAnalyzeOnly ||
		rule.Recommendation == RecommendationNotRecommended ||
		rule.Recommendation == RecommendationForbidden
	if !validRecommendation {
		return fmt.Errorf("rule %s has invalid recommendation %q", rule.ID, rule.Recommendation)
	}
	if rule.DefaultSelected && (rule.Risk != RiskLow || rule.Recommendation != RecommendationRecommended) {
		return fmt.Errorf("rule %s cannot be selected by default", rule.ID)
	}
	if rule.DefaultSelected && rule.Action.Type == "analyze" {
		return fmt.Errorf("analysis-only rule %s cannot be selected by default", rule.ID)
	}
	if (rule.Risk == RiskHigh || rule.Risk == RiskForbidden) && (rule.Help.Details == "" || rule.Help.SpecialWarning == "") {
		return fmt.Errorf("rule %s requires detailed help and a special warning", rule.ID)
	}
	if rule.Action.Type == "shell" || rule.Action.Type == "command" {
		return fmt.Errorf("rule %s uses a forbidden action type", rule.ID)
	}
	if rule.Action.Type != "permanent_delete" && rule.Action.Type != "empty_recycle_bin" && rule.Action.Type != "analyze" {
		return fmt.Errorf("rule %s uses an unsupported action type %q", rule.ID, rule.Action.Type)
	}
	return nil
}

func (s *Service) List() []Rule {
	result := make([]Rule, 0, len(s.rules))
	for _, rule := range s.rules {
		if rule.Enabled && (rule.Platform == "all" || rule.Platform == runtime.GOOS || (rule.Platform == "windows" && runtime.GOOS != "windows")) {
			result = append(result, rule)
		}
	}
	return result
}

func (s *Service) Get(id string) (Rule, bool) {
	for _, rule := range s.rules {
		if rule.ID == id {
			return rule, true
		}
	}
	return Rule{}, false
}
