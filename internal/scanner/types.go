package scanner

import "time"

type Status string

const (
	StatusCreated    Status = "created"
	StatusValidating Status = "validating"
	StatusRunning    Status = "running"
	StatusCancelling Status = "cancelling"
	StatusCancelled  Status = "cancelled"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

type Request struct {
	Mode                  string   `json:"mode"`
	Roots                 []string `json:"roots"`
	RuleIDs               []string `json:"rule_ids"`
	ExcludeRoots          []string `json:"exclude_roots,omitempty"`
	MinSizeBytes          int64    `json:"min_size_bytes,omitempty"`
	MinDirectorySizeBytes int64    `json:"min_directory_size_bytes,omitempty"`
}

type ErrorItem struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type Job struct {
	ID                   string      `json:"id"`
	Status               Status      `json:"status"`
	Mode                 string      `json:"mode"`
	CurrentPath          string      `json:"current_path"`
	ScannedFiles         int64       `json:"scanned_files"`
	MatchedFiles         int64       `json:"matched_files"`
	LogicalBytes         int64       `json:"logical_bytes"`
	AllocatedBytes       int64       `json:"allocated_bytes"`
	EstimatedReclaimable int64       `json:"estimated_reclaimable"`
	ErrorCount           int         `json:"error_count"`
	Errors               []ErrorItem `json:"errors,omitempty"`
	StartedAt            time.Time   `json:"started_at"`
	CompletedAt          *time.Time  `json:"completed_at,omitempty"`
}

type Item struct {
	ID                   string    `json:"id"`
	RuleID               string    `json:"rule_id"`
	MatchedRuleIDs       []string  `json:"matched_rule_ids"`
	Name                 string    `json:"name"`
	Path                 string    `json:"path"`
	Directory            string    `json:"directory"`
	IsDirectory          bool      `json:"is_directory"`
	Extension            string    `json:"extension"`
	Category             string    `json:"category"`
	Purpose              string    `json:"purpose"`
	CleanEffect          string    `json:"clean_effect"`
	Recommendation       string    `json:"recommendation"`
	RecommendationReason string    `json:"recommendation_reason"`
	Risk                 string    `json:"risk"`
	DefaultSelected      bool      `json:"default_selected"`
	Selectable           bool      `json:"selectable"`
	Action               string    `json:"action"`
	RecoveryType         string    `json:"recovery_type"`
	RequiresAdmin        bool      `json:"requires_admin"`
	RequiresRestart      bool      `json:"requires_restart"`
	LogicalSize          int64     `json:"logical_size"`
	AllocatedSize        int64     `json:"allocated_size"`
	EstimatedReclaimable int64     `json:"estimated_reclaimable"`
	VolumeID             string    `json:"volume_id"`
	FileID               string    `json:"file_id"`
	LinkCount            uint32    `json:"link_count"`
	ModifiedAt           time.Time `json:"modified_at"`
	HelpSummary          string    `json:"help_summary"`
	HelpDetails          string    `json:"help_details,omitempty"`
	SpecialWarning       string    `json:"special_warning,omitempty"`
	ManualSteps          []string  `json:"manual_steps,omitempty"`
}

type ItemPage struct {
	Items  []Item `json:"items"`
	Total  int    `json:"total"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

type Folder struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Path                 string    `json:"path"`
	FileCount            int       `json:"file_count"`
	LogicalBytes         int64     `json:"logical_bytes"`
	AllocatedBytes       int64     `json:"allocated_bytes"`
	EstimatedReclaimable int64     `json:"estimated_reclaimable"`
	HighestRisk          string    `json:"highest_risk"`
	Children             []*Folder `json:"children"`
	ItemIDs              []string  `json:"item_ids"`
}
