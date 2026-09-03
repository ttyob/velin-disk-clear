package rules

import "time"

type RuleType string

const (
	RuleTypeSystem     RuleType = "system"
	RuleTypeThirdParty RuleType = "third_party"
	RuleTypeGeneral    RuleType = "general"
)

type Risk string

const (
	RiskLow       Risk = "low"
	RiskMedium    Risk = "medium"
	RiskHigh      Risk = "high"
	RiskForbidden Risk = "forbidden"
)

type Recommendation string

const (
	RecommendationRecommended    Recommendation = "recommended"
	RecommendationOptional       Recommendation = "optional"
	RecommendationAnalyzeOnly    Recommendation = "analyze_only"
	RecommendationNotRecommended Recommendation = "not_recommended"
	RecommendationForbidden      Recommendation = "forbidden"
)

type ScanMode string

const (
	ScanModeQuick    ScanMode = "quick"
	ScanModeStandard ScanMode = "standard"
	ScanModeDeep     ScanMode = "deep"
)

type Help struct {
	Summary        string   `json:"summary"`
	Details        string   `json:"details,omitempty"`
	SpecialWarning string   `json:"special_warning,omitempty"`
	ManualSteps    []string `json:"manual_steps,omitempty"`
	LearnMoreURL   string   `json:"learn_more_url,omitempty"`
}

type ScanSpec struct {
	Roots               []string `json:"roots"`
	Include             []string `json:"include,omitempty"`
	Exclude             []string `json:"exclude,omitempty"`
	Extensions          []string `json:"extensions,omitempty"`
	MinAge              string   `json:"min_age,omitempty"`
	MinSizeBytes        int64    `json:"min_size_bytes,omitempty"`
	StayOnVolume        bool     `json:"stay_on_volume"`
	FollowReparsePoints bool     `json:"follow_reparse_points"`
}

func (s ScanSpec) MinimumAge() time.Duration {
	if s.MinAge == "" {
		return 0
	}
	duration, err := time.ParseDuration(s.MinAge)
	if err != nil {
		return 0
	}
	return duration
}

type ActionSpec struct {
	Type string `json:"type"`
}

type SafetySpec struct {
	AllowedRoots          []string `json:"allowed_roots"`
	RevalidateBeforeClean bool     `json:"revalidate_before_clean"`
}

type Rule struct {
	ID                        string         `json:"id"`
	Version                   int            `json:"version"`
	Name                      string         `json:"name"`
	Description               string         `json:"description"`
	Purpose                   string         `json:"purpose"`
	CleanEffect               string         `json:"clean_effect"`
	Recommendation            Recommendation `json:"recommendation"`
	RecommendationReason      string         `json:"recommendation_reason"`
	Category                  string         `json:"category"`
	RuleType                  RuleType       `json:"rule_type"`
	Platform                  string         `json:"platform"`
	Enabled                   bool           `json:"enabled"`
	Risk                      Risk           `json:"risk"`
	DefaultSelected           bool           `json:"default_selected"`
	RequiresAdmin             bool           `json:"requires_admin"`
	SupportedWindowsVersions  []string       `json:"supported_windows_versions"`
	Scope                     string         `json:"scope"`
	SizeMode                  string         `json:"size_mode"`
	RecoveryType              string         `json:"recovery_type"`
	RequiresNetworkAfterClean bool           `json:"requires_network_after_clean"`
	MaySignOut                bool           `json:"may_sign_out"`
	RequiresRestart           bool           `json:"requires_restart"`
	ProcessGuard              []string       `json:"process_guard"`
	Conflicts                 []string       `json:"conflicts"`
	LastVerifiedAt            string         `json:"last_verified_at"`
	Source                    string         `json:"source"`
	Modes                     []ScanMode     `json:"modes"`
	Help                      Help           `json:"help"`
	Scan                      ScanSpec       `json:"scan"`
	Action                    ActionSpec     `json:"action"`
	Safety                    SafetySpec     `json:"safety"`
}

type Catalog struct {
	Version int    `json:"version"`
	Rules   []Rule `json:"rules"`
}
