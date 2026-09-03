package cleaner

import "time"

type BuildRequest struct {
	ScanID  string   `json:"scan_id"`
	ItemIDs []string `json:"item_ids"`
}

type ExecuteRequest struct {
	PlanID            string `json:"plan_id"`
	ConfirmationToken string `json:"confirmation_token"`
}

type PlanItem struct {
	ID                   string `json:"id"`
	RuleID               string `json:"rule_id"`
	Name                 string `json:"name"`
	Path                 string `json:"path"`
	Risk                 string `json:"risk"`
	Action               string `json:"action"`
	RecoveryType         string `json:"recovery_type"`
	AllocatedSize        int64  `json:"allocated_size"`
	EstimatedReclaimable int64  `json:"estimated_reclaimable"`
	DefaultSelected      bool   `json:"default_selected"`
	RequiresAdmin        bool   `json:"requires_admin"`
}

type Plan struct {
	ID                   string     `json:"id"`
	ScanID               string     `json:"scan_id"`
	Status               string     `json:"status"`
	Items                []PlanItem `json:"items"`
	ItemCount            int        `json:"item_count"`
	DefaultSelectedCount int        `json:"default_selected_count"`
	ManualSelectedCount  int        `json:"manual_selected_count"`
	LowRiskCount         int        `json:"low_risk_count"`
	MediumRiskCount      int        `json:"medium_risk_count"`
	HighRiskCount        int        `json:"high_risk_count"`
	PermanentDeleteCount int        `json:"permanent_delete_count"`
	EstimatedReclaimable int64      `json:"estimated_reclaimable"`
	ConfirmationToken    string     `json:"confirmation_token"`
	CreatedAt            time.Time  `json:"created_at"`
	ExpiresAt            time.Time  `json:"expires_at"`
}

type ItemResult struct {
	ItemID         string `json:"item_id"`
	Path           string `json:"path"`
	Action         string `json:"action"`
	Status         string `json:"status"`
	BytesProcessed int64  `json:"bytes_processed"`
	Error          string `json:"error,omitempty"`
}

type Result struct {
	ID                string       `json:"id"`
	PlanID            string       `json:"plan_id"`
	Status            string       `json:"status"`
	Succeeded         int          `json:"succeeded"`
	Skipped           int          `json:"skipped"`
	Failed            int          `json:"failed"`
	DeletedBytes      int64        `json:"deleted_bytes"`
	ActuallyReclaimed int64        `json:"actually_reclaimed"`
	Items             []ItemResult `json:"items"`
	StartedAt         time.Time    `json:"started_at"`
	CompletedAt       time.Time    `json:"completed_at"`
}
