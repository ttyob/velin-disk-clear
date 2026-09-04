package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"ai-clear/internal/ai/provider"
	"ai-clear/internal/scanner"
)

const systemPrompt = `你是 Velin Clear 的内置 AI 助手，可以正常回答用户的各类问题，并使用中文、简洁清楚地交流。
你没有任意文件系统、网络、命令或删除权限。仅当用户明确要求检查本机磁盘、目录、文件占用、清理候选或删除影响时，才调用受控的 scan_files 工具；其他对话直接回答，不调用工具。scan_files 是只读工具，本地 Go 程序会在指定目录内扫描全部常规文件，忽略任何模式、规则或大小阈值要求。不得在没有工具结果时编造文件、路径或 item_id。你不能删除文件；任何清理仍需用户在软件中勾选并确认。
扫描模式必须返回 JSON：{"summary":"...","items":[{"item_id":"...","recommendation":"recommended|optional|review|keep","default_selected":false,"suggested_action":"select|keep|manual","confidence":0.0,"reason":"..."}]}。
调用 scan_files 后的对话必须返回 JSON：{"reply":"...","items":[...]}。items 只能引用 USER_DATA 中的 item_id。路径、用途、大小和安全属性以 USER_DATA 为准；不要改写它们。USER_DATA.scan_summary 的 allocated_bytes 是所有匹配文件的扫描占用汇总，可能包含未展示、不可删除或未纳入 items 的文件，绝不能把它称为“可清理空间”。只有在 items 中实际列出且 selectable 的文件才可说明为可清理候选；无法准确合计时不要给出可清理空间数字。
高风险、forbidden、analysis-only、分页文件、休眠文件、Windows.old、系统转储、个人大文件和不可验证项目只能 review/keep、default_selected=false、suggested_action=manual/keep。
只有明确低风险且扫描器标记 selectable 的缓存/临时项才可 recommended/select；模型的 default_selected 只是建议，最终由 Go 安全规则决定。`

type Service struct {
	provider *provider.Service
	scanner  *scanner.Service
	mu       sync.Mutex
	sessions map[string][]ChatMessage
}

var scanTools = []provider.ToolDefinition{{
	Name: "scan_files", Description: "在授权的 Windows 绝对目录内执行一次无筛选的只读文件扫描并返回真实文件元数据。roots 必须至少包含一个目录；用户没有指定目录时使用 C:\\。只传 roots；不要传规则、扫描模式或大小阈值。不会删除文件。",
	Parameters: map[string]any{"type": "object", "properties": map[string]any{
		"roots": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "要扫描的绝对目录"},
	}, "required": []string{"roots"}, "additionalProperties": false},
}}

func New(providerService *provider.Service, scannerService *scanner.Service) *Service {
	return &Service{provider: providerService, scanner: scannerService, sessions: make(map[string][]ChatMessage)}
}

func (s *Service) Run(ctx context.Context, request Request) (Result, error) {
	mode := strings.ToLower(strings.TrimSpace(request.Mode))
	if mode == "" {
		mode = "scan"
	}
	if mode != "scan" && mode != "chat" {
		return Result{}, errors.New("unsupported AI Assistant mode")
	}
	request.Mode = mode
	request.Objective = limitText(request.Objective, 1000)
	if mode == "scan" && request.Objective == "" {
		request.Objective = "分析扫描结果，列出真实文件及其用途、清理影响、风险和建议。"
	}
	if len(request.Roots) == 0 {
		if root := defaultWindowsScanRoot(); root != "" {
			request.Roots = []string{root}
		}
	}
	if err := validateRoots(request.Roots); err != nil {
		return Result{}, err
	}
	var scanID string
	var job scanner.Job
	var page scanner.ItemPage
	var content, model string
	if request.ScanID != "" {
		var err error
		scanID, err = s.ensureScan(ctx, request)
		if err != nil {
			return Result{}, err
		}
		job, err = s.scanner.Job(scanID)
		if err != nil {
			return Result{}, err
		}
		page, err = s.scanner.Items(scanID, 0, 500)
		if err != nil {
			return Result{}, err
		}
		payload, _ := json.Marshal(map[string]any{"kind": "USER_DATA", "mode": mode, "objective": request.Objective, "scan_id": scanID, "scan_summary": scanSummary(job), "items": buildModelItems(page.Items), "items_total": page.Total})
		content, model, err = s.provider.CompleteJSON(ctx, systemPrompt, string(payload))
		if err != nil {
			return Result{}, err
		}
	} else if mode == "chat" {
		var err error
		scanID, job, page, content, model, err = s.runChatTurn(ctx, request)
		if err != nil {
			return Result{}, err
		}
	} else {
		var err error
		scanID, job, page, content, model, err = s.runToolDrivenScan(ctx, request)
		if err != nil {
			return Result{}, err
		}
	}
	allowed := make(map[string]scanner.Item, len(page.Items))
	for _, item := range page.Items {
		allowed[item.ID] = item
	}
	userData := map[string]any{"kind": "USER_DATA", "mode": mode, "objective": request.Objective, "scan_id": scanID, "scan_summary": scanSummary(job), "items": buildModelItems(page.Items), "items_total": page.Total}
	if mode == "chat" {
		userData["conversation"] = s.conversation(request)
	}
	if request.ScanID != "" {
		// The legacy snapshot path already completed its model turn above.
		_ = userData
	}
	if mode == "chat" {
		return s.parseChat(content, model, scanID, request, page.Items, page.Total, allowed), nil
	}
	result, err := s.parseScan(content, model, scanID, page.Items, allowed, page.Total)
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func scanSummary(job scanner.Job) map[string]any {
	return map[string]any{"matched_files": job.MatchedFiles, "scanned_files": job.ScannedFiles, "allocated_bytes": job.AllocatedBytes, "estimated_reclaimable": job.EstimatedReclaimable, "error_count": job.ErrorCount}
}

func (s *Service) runToolDrivenScan(ctx context.Context, request Request) (string, scanner.Job, scanner.ItemPage, string, string, error) {
	messages := []map[string]any{{"role": "user", "content": request.Objective}}
	completion, _, err := s.provider.CompleteTools(ctx, systemPrompt, messages, scanTools, true)
	if err != nil {
		return "", scanner.Job{}, scanner.ItemPage{}, "", "", err
	}
	if len(completion.ToolCall) == 0 {
		return "", scanner.Job{}, scanner.ItemPage{}, "", "", errors.New("model did not call scan_files; confirm that the configured model supports OpenAI function calling")
	}
	return s.completeToolDrivenScan(ctx, request, completion.ToolCall)
}

// runChatTurn lets the model decide whether this message needs a local scan.
// Ordinary questions receive a normal assistant reply without touching disk.
func (s *Service) runChatTurn(ctx context.Context, request Request) (string, scanner.Job, scanner.ItemPage, string, string, error) {
	history := s.conversation(request)
	messages := make([]map[string]any, 0, len(history)+1)
	for _, message := range history {
		if message.Role == "user" || message.Role == "assistant" {
			messages = append(messages, map[string]any{"role": message.Role, "content": message.Content})
		}
	}
	messages = append(messages, map[string]any{"role": "user", "content": request.Objective})
	completion, model, err := s.provider.CompleteTools(ctx, systemPrompt, messages, scanTools, false)
	if err != nil {
		return "", scanner.Job{}, scanner.ItemPage{}, "", "", err
	}
	if len(completion.ToolCall) == 0 {
		return "", scanner.Job{}, scanner.ItemPage{}, completion.Content, model, nil
	}
	return s.completeToolDrivenScan(ctx, request, completion.ToolCall)
}

func (s *Service) completeToolDrivenScan(ctx context.Context, request Request, calls []provider.ToolCall) (string, scanner.Job, scanner.ItemPage, string, string, error) {
	if len(calls) > 1 {
		return "", scanner.Job{}, scanner.ItemPage{}, "", "", errors.New("agent returned multiple scan_files calls; only one scan is allowed per request")
	}
	call := calls[0]
	if call.Name != "scan_files" {
		return "", scanner.Job{}, scanner.ItemPage{}, "", "", fmt.Errorf("agent requested unsupported tool %q", call.Name)
	}
	var args struct {
		Roots []string `json:"roots"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return "", scanner.Job{}, scanner.ItemPage{}, "", "", fmt.Errorf("invalid scan_files arguments: %w", err)
	}
	if len(args.Roots) == 0 {
		// Some otherwise compatible providers omit required function arguments.
		// The caller's default Windows system drive remains the only fallback.
		args.Roots = append([]string(nil), request.Roots...)
	}
	if len(args.Roots) == 0 {
		return "", scanner.Job{}, scanner.ItemPage{}, "", "", errors.New("scan_files requires at least one absolute directory; specify a Windows directory such as C:\\")
	}
	if err := validateRoots(args.Roots); err != nil {
		return "", scanner.Job{}, scanner.ItemPage{}, "", "", err
	}

	// AI scanning deliberately does not inherit model-supplied rule IDs, modes,
	// or thresholds. The generic rule supplies the Cleaner allowlist while the
	// one-byte threshold returns every regular file under the selected roots.
	next, err := s.scanner.Start(ctx, scanner.Request{Mode: "deep", Roots: args.Roots, RuleIDs: []string{"generic.large_files"}, ExcludeRoots: request.ExcludeRoots, MinSizeBytes: 1})
	if err != nil {
		return "", scanner.Job{}, scanner.ItemPage{}, "", "", err
	}
	var job scanner.Job
	for {
		job, err = s.scanner.Job(next.ID)
		if err != nil {
			return "", scanner.Job{}, scanner.ItemPage{}, "", "", err
		}
		if job.Status == scanner.StatusCompleted {
			break
		}
		if job.Status == scanner.StatusCancelled || job.Status == scanner.StatusFailed {
			return "", scanner.Job{}, scanner.ItemPage{}, "", "", fmt.Errorf("scan did not complete: %s", job.Status)
		}
		select {
		case <-ctx.Done():
			return "", scanner.Job{}, scanner.ItemPage{}, "", "", ctx.Err()
		case <-time.After(40 * time.Millisecond):
		}
	}
	page, err := s.scanner.Items(next.ID, 0, 500)
	if err != nil {
		return "", scanner.Job{}, scanner.ItemPage{}, "", "", err
	}
	toolResult, _ := json.Marshal(map[string]any{
		"kind": "USER_DATA", "mode": request.Mode, "objective": request.Objective,
		"scan_id": next.ID, "scan_summary": scanSummary(job), "items": buildModelItems(page.Items), "items_total": page.Total,
	})
	finalPrompt := systemPrompt + "\nscan_files 已执行完成。现在只能依据 USER_DATA 输出最终 JSON，不得再次调用工具，也不得输出 JSON 以外的内容。"
	content, model, err := s.provider.CompleteJSON(ctx, finalPrompt, string(toolResult))
	if err != nil {
		return "", scanner.Job{}, scanner.ItemPage{}, "", "", err
	}
	return next.ID, job, page, content, model, nil
}

func (s *Service) ensureScan(ctx context.Context, request Request) (string, error) {
	if request.ScanID != "" {
		job, err := s.scanner.Job(request.ScanID)
		if err != nil {
			return "", err
		}
		if job.Status != scanner.StatusCompleted {
			return "", errors.New("AI Assistant requires a completed scan snapshot")
		}
		return request.ScanID, nil
	}
	mode := request.ScanMode
	if mode == "" {
		mode = "standard"
	}
	job, err := s.scanner.Start(ctx, scanner.Request{Mode: mode, Roots: request.Roots, RuleIDs: request.RuleIDs})
	if err != nil {
		return "", err
	}
	ticker := time.NewTicker(40 * time.Millisecond)
	defer ticker.Stop()
	for {
		current, err := s.scanner.Job(job.ID)
		if err != nil {
			return "", err
		}
		switch current.Status {
		case scanner.StatusCompleted:
			return current.ID, nil
		case scanner.StatusFailed, scanner.StatusCancelled:
			return "", fmt.Errorf("scan did not complete: %s", current.Status)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
		}
	}
}

func buildModelItems(items []scanner.Item) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"item_id": item.ID, "name": item.Name, "path": item.Path, "directory": item.Directory, "is_directory": item.IsDirectory, "extension": item.Extension,
			"category": item.Category, "purpose": item.Purpose, "clean_effect": item.CleanEffect, "recommendation": item.Recommendation,
			"recommendation_reason": item.RecommendationReason, "risk": item.Risk, "default_selected": item.DefaultSelected,
			"selectable": item.Selectable, "action": item.Action, "logical_size": item.LogicalSize, "allocated_size": item.AllocatedSize,
			"modified_at": item.ModifiedAt.Format(time.RFC3339), "help_summary": item.HelpSummary, "help_details": item.HelpDetails,
			"special_warning": item.SpecialWarning, "manual_steps": item.ManualSteps,
		})
	}
	return result
}

type modelItem struct {
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
	HelpDetails          string   `json:"help_details"`
	SpecialWarning       string   `json:"special_warning"`
	ManualSteps          []string `json:"manual_steps"`
	Classification       string   `json:"classification"`
	SuggestedAction      string   `json:"suggested_action"`
	Confidence           float64  `json:"confidence"`
	Reason               string   `json:"reason"`
}

func (s *Service) parseScan(content, model, scanID string, items []scanner.Item, allowed map[string]scanner.Item, total int) (Result, error) {
	var output struct {
		Summary string      `json:"summary"`
		Items   []modelItem `json:"items"`
	}
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		return Result{}, fmt.Errorf("Agent output failed schema validation: %w", err)
	}
	if len(output.Items) > 500 {
		return Result{}, errors.New("Agent returned too many findings")
	}
	seen := make(map[string]struct{})
	findings := make([]Finding, 0, len(output.Items))
	for _, candidate := range output.Items {
		item, ok := allowed[candidate.ItemID]
		if !ok {
			return Result{}, fmt.Errorf("Agent referenced unknown item ID %q", candidate.ItemID)
		}
		if _, duplicate := seen[candidate.ItemID]; duplicate {
			continue
		}
		seen[candidate.ItemID] = struct{}{}
		if candidate.Confidence < 0 || candidate.Confidence > 1 {
			return Result{}, fmt.Errorf("invalid confidence for item %q", candidate.ItemID)
		}
		finding := findingFromItem(item)
		finding.Confidence = candidate.Confidence
		finding.Reason = limitText(candidate.Reason, 500)
		finding.Recommendation = normalizeRecommendation(candidate.Recommendation, item.Recommendation)
		finding.SuggestedAction = normalizeAction(candidate.SuggestedAction)
		finding.DefaultSelected = candidate.DefaultSelected && item.DefaultSelected && item.Selectable && item.Risk == "low" && item.Action != "analyze"
		if item.Risk == "high" || item.Risk == "forbidden" || !item.Selectable || item.Action == "analyze" || item.Recommendation == "analyze_only" || item.Recommendation == "not_recommended" || item.Recommendation == "forbidden" {
			finding.DefaultSelected = false
			finding.Recommendation = "review"
			finding.SuggestedAction = "manual"
		}
		if finding.Recommendation == "keep" {
			finding.DefaultSelected = false
			finding.SuggestedAction = "keep"
		}
		if finding.DefaultSelected {
			finding.SuggestedAction = "select"
		}
		findings = append(findings, finding)
	}
	suggestions := make([]Suggestion, 0, len(findings))
	for _, finding := range findings {
		suggestions = append(suggestions, Suggestion{ItemID: finding.ItemID, Classification: finding.Category, Recommendation: finding.Recommendation, Risk: finding.Risk, Confidence: finding.Confidence, Reason: finding.Reason, SuggestedAction: finding.SuggestedAction})
	}
	return Result{Mode: "scan", Summary: limitText(output.Summary, 1000), Items: findings, Suggestions: suggestions, ScanID: scanID, ProviderModel: model, Truncated: total > len(items)}, nil
}

func findingFromItem(item scanner.Item) Finding {
	return Finding{ItemID: item.ID, RuleID: item.RuleID, Name: item.Name, Path: item.Path, Directory: item.Directory, IsDirectory: item.IsDirectory, Extension: item.Extension, Category: item.Category, Purpose: item.Purpose, CleanEffect: item.CleanEffect, Recommendation: item.Recommendation, RecommendationReason: item.RecommendationReason, Risk: item.Risk, DefaultSelected: item.DefaultSelected, Selectable: item.Selectable, Action: item.Action, LogicalSize: item.LogicalSize, AllocatedSize: item.AllocatedSize, ModifiedAt: item.ModifiedAt.Format(time.RFC3339), HelpSummary: item.HelpSummary, HelpDetails: item.HelpDetails, SpecialWarning: item.SpecialWarning, ManualSteps: item.ManualSteps}
}

func (s *Service) parseChat(content, model, scanID string, request Request, items []scanner.Item, total int, allowed map[string]scanner.Item) Result {
	var output struct {
		Reply string      `json:"reply"`
		Items []modelItem `json:"items"`
	}
	if err := json.Unmarshal([]byte(content), &output); err != nil {
		// General assistant replies do not require structured output. Structured
		// JSON remains mandatory only after the model has requested a local scan.
		output.Reply = strings.TrimSpace(content)
		if output.Reply == "" {
			output.Reply = "模型没有返回内容，请重试。"
		}
	}
	output.Reply = limitText(output.Reply, 3000)
	findings := make([]Finding, 0, len(output.Items))
	seen := make(map[string]struct{})
	for _, candidate := range output.Items {
		item, ok := allowed[candidate.ItemID]
		if !ok {
			continue
		}
		if _, exists := seen[candidate.ItemID]; exists {
			continue
		}
		seen[candidate.ItemID] = struct{}{}
		if candidate.Confidence < 0 || candidate.Confidence > 1 {
			continue
		}
		finding := findingFromItem(item)
		finding.Confidence = candidate.Confidence
		finding.Reason = limitText(candidate.Reason, 500)
		finding.Recommendation = normalizeRecommendation(candidate.Recommendation, item.Recommendation)
		finding.SuggestedAction = normalizeAction(candidate.SuggestedAction)
		finding.DefaultSelected = candidate.DefaultSelected && item.DefaultSelected && item.Selectable && item.Risk == "low" && item.Action != "analyze"
		if item.Risk == "high" || item.Risk == "forbidden" || !item.Selectable || item.Action == "analyze" {
			finding.DefaultSelected = false
			finding.SuggestedAction = "manual"
		}
		findings = append(findings, finding)
	}
	if scanID != "" {
		// The model may focus on only a subset of the returned metadata. Preserve
		// the complete local page in the review UI so its visible total cannot be
		// mistaken for a smaller, model-selected scan result.
		for _, item := range items {
			if _, exists := seen[item.ID]; exists {
				continue
			}
			finding := findingFromItem(item)
			finding.DefaultSelected = false
			finding.Recommendation = "review"
			finding.SuggestedAction = "manual"
			finding.Reason = "AI 未对该文件生成单独建议，请人工确认用途和删除影响。"
			findings = append(findings, finding)
		}
	}
	if request.SessionID == "" {
		request.SessionID = fmt.Sprintf("session-%d", time.Now().UnixNano())
	}
	s.mu.Lock()
	s.sessions[request.SessionID] = append(s.sessions[request.SessionID], ChatMessage{Role: "user", Content: request.Objective}, ChatMessage{Role: "assistant", Content: output.Reply})
	s.mu.Unlock()
	return Result{Mode: "chat", Reply: output.Reply, ScanID: scanID, SessionID: request.SessionID, ProviderModel: model, Items: findings, Truncated: total > len(items)}
}

func (s *Service) conversation(request Request) []ChatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := append([]ChatMessage(nil), s.sessions[request.SessionID]...)
	if len(request.Messages) > 0 {
		result = append(result, request.Messages...)
	}
	if len(result) > 20 {
		result = result[len(result)-20:]
	}
	return result
}

func normalizeRecommendation(value, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "recommended", "candidate":
		return "recommended"
	case "optional":
		return "optional"
	case "review":
		return "review"
	case "keep":
		return "keep"
	default:
		if fallback == "recommended" {
			return "recommended"
		}
		return "review"
	}
}

func normalizeAction(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "select", "add_candidate":
		return "select"
	case "keep", "ignore":
		return "keep"
	default:
		return "manual"
	}
}

func validateRoots(roots []string) error {
	if len(roots) > 8 {
		return errors.New("at most 8 scan roots are allowed")
	}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" || strings.ContainsAny(root, "*?[]") {
			return errors.New("scan roots must be explicit directories")
		}
		if !filepath.IsAbs(root) {
			return fmt.Errorf("scan root must be absolute: %q", root)
		}
	}
	return nil
}

func defaultWindowsScanRoot() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	drive := strings.TrimSpace(os.Getenv("SystemDrive"))
	if len(drive) == 2 && drive[1] == ':' {
		return strings.ToUpper(drive[:1]) + `:\`
	}
	return `C:\`
}

func limitText(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > max {
		return string(runes[:max])
	}
	return string(runes)
}
