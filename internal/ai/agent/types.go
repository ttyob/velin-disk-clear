package agent

// Request is the only input accepted by AI Assistant. The scanner, not the
// model, owns filesystem access and decides which roots/rules are allowed.
type Request struct {
	Objective    string        `json:"objective"`
	ScanID       string        `json:"scan_id"`
	Mode         string        `json:"mode"`      // scan | chat
	ScanMode     string        `json:"scan_mode"` // quick | standard | deep
	Roots        []string      `json:"roots,omitempty"`
	RuleIDs      []string      `json:"rule_ids,omitempty"`
	ExcludeRoots []string      `json:"exclude_roots,omitempty"`
	SessionID    string        `json:"session_id,omitempty"`
	Messages     []ChatMessage `json:"messages,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Finding is a real scanner item enriched with an AI explanation. Path and
// safety fields always come from the scanner snapshot, never from the model.
type Finding struct {
	ItemID               string   `json:"item_id"`
	RuleID               string   `json:"rule_id"`
	Name                 string   `json:"name"`
	Path                 string   `json:"path"`
	Directory            string   `json:"directory"`
	IsDirectory          bool     `json:"is_directory"`
	Extension            string   `json:"extension"`
	Category             string   `json:"category"`
	Purpose              string   `json:"purpose"`
	CleanEffect          string   `json:"clean_effect"`
	Recommendation       string   `json:"recommendation"`
	RecommendationReason string   `json:"recommendation_reason"`
	Risk                 string   `json:"risk"`
	DefaultSelected      bool     `json:"default_selected"`
	Selectable           bool     `json:"selectable"`
	Action               string   `json:"action"`
	LogicalSize          int64    `json:"logical_size"`
	AllocatedSize        int64    `json:"allocated_size"`
	ModifiedAt           string   `json:"modified_at"`
	HelpSummary          string   `json:"help_summary"`
	HelpDetails          string   `json:"help_details,omitempty"`
	SpecialWarning       string   `json:"special_warning,omitempty"`
	ManualSteps          []string `json:"manual_steps,omitempty"`
	Confidence           float64  `json:"confidence"`
	Reason               string   `json:"reason"`
	SuggestedAction      string   `json:"suggested_action"`
}

// Suggestion is kept for compatibility with older clients.
type Suggestion struct {
	ItemID          string  `json:"item_id"`
	Classification  string  `json:"classification"`
	Recommendation  string  `json:"recommendation"`
	Risk            string  `json:"risk"`
	Confidence      float64 `json:"confidence"`
	Reason          string  `json:"reason"`
	SuggestedAction string  `json:"suggested_action"`
}

type Result struct {
	Mode          string       `json:"mode"`
	Summary       string       `json:"summary"`
	Reply         string       `json:"reply,omitempty"`
	Items         []Finding    `json:"items"`
	Suggestions   []Suggestion `json:"suggestions,omitempty"`
	ScanID        string       `json:"scan_id"`
	SessionID     string       `json:"session_id,omitempty"`
	ProviderModel string       `json:"provider_model"`
	Truncated     bool         `json:"truncated"`
}

type Preset struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Objective   string `json:"objective"`
	Description string `json:"description"`
}

var Presets = []Preset{
	{ID: "safe-space", Name: "安全释放空间", Objective: "优先分析低风险、可恢复的缓存和临时文件，列出可安全清理项。", Description: "仅关注低风险可清理内容"},
	{ID: "large-files", Name: "查找大文件", Objective: "列出占用空间较大的文件，说明用途和清理影响，默认不勾选需要人工确认的项目。", Description: "发现大文件，不自动建议删除"},
	{ID: "system-deep", Name: "系统深度检查", Objective: "分析 Windows 系统缓存、日志、更新残留、回收站和特殊系统文件，明确建议清理或手动处理。", Description: "覆盖更多 Windows 专属规则"},
}
