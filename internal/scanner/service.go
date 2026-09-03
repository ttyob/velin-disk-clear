package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ai-clear/internal/disks"
	"ai-clear/internal/fsmeta"
	"ai-clear/internal/platform"
	"ai-clear/internal/rules"
)

const maxRecordedErrors = 100

type jobState struct {
	mu        sync.RWMutex
	job       Job
	request   Request
	items     []Item
	itemIndex map[string]int
	cancel    context.CancelFunc
}

type Service struct {
	rules *rules.Service
	mu    sync.RWMutex
	jobs  map[string]*jobState
	seq   atomic.Uint64
}

func New(ruleService *rules.Service) *Service {
	return &Service{rules: ruleService, jobs: make(map[string]*jobState)}
}

func (s *Service) Start(parent context.Context, request Request) (Job, error) {
	if request.Mode == "" {
		request.Mode = string(rules.ScanModeQuick)
	}
	if request.Mode != string(rules.ScanModeQuick) &&
		request.Mode != string(rules.ScanModeStandard) &&
		request.Mode != string(rules.ScanModeDeep) {
		return Job{}, fmt.Errorf("unsupported scan mode %q", request.Mode)
	}
	selectedRules, err := s.selectRules(request)
	if err != nil {
		return Job{}, err
	}
	ctx, cancel := context.WithCancel(parent)
	now := time.Now()
	id := fmt.Sprintf("scan-%d-%d", now.UnixMilli(), s.seq.Add(1))
	state := &jobState{
		job:       Job{ID: id, Status: StatusCreated, Mode: request.Mode, StartedAt: now},
		request:   request,
		itemIndex: make(map[string]int),
		cancel:    cancel,
	}
	s.mu.Lock()
	s.jobs[id] = state
	s.mu.Unlock()

	go s.run(ctx, state, request, selectedRules)
	return state.snapshot(), nil
}

// CleanCandidate is an immutable view of a scanned item plus the roots that
// the matching rule authorized for this scan request.
type CleanCandidate struct {
	Item         Item
	AllowedRoots []string
}

// CleanCandidates resolves user-selected item IDs from a completed scan.
// Callers never provide paths or actions directly.
func (s *Service) CleanCandidates(id string, itemIDs []string) ([]CleanCandidate, error) {
	state, err := s.state(id)
	if err != nil {
		return nil, err
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	if state.job.Status != StatusCompleted {
		return nil, errors.New("clean plans require a completed scan")
	}
	if len(itemIDs) == 0 {
		return nil, errors.New("no scan items selected")
	}
	byID := make(map[string]Item, len(state.items))
	for _, item := range state.items {
		byID[item.ID] = item
	}
	seen := make(map[string]struct{}, len(itemIDs))
	result := make([]CleanCandidate, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		if _, duplicate := seen[itemID]; duplicate {
			continue
		}
		seen[itemID] = struct{}{}
		item, ok := byID[itemID]
		if !ok {
			return nil, fmt.Errorf("scan item %q not found", itemID)
		}
		if !item.Selectable || item.Action == "analyze" {
			return nil, fmt.Errorf("scan item %q is analysis-only", itemID)
		}
		rule, ok := s.rules.Get(item.RuleID)
		if !ok {
			return nil, fmt.Errorf("rule %q is no longer available", item.RuleID)
		}
		roots := resolveRoots(rule.Safety.AllowedRoots, state.request.Roots)
		if len(roots) == 0 {
			return nil, fmt.Errorf("rule %q has no resolved allowed roots", item.RuleID)
		}
		result = append(result, CleanCandidate{Item: item, AllowedRoots: roots})
	}
	return result, nil
}

func (s *Service) selectRules(request Request) ([]rules.Rule, error) {
	requested := make(map[string]struct{}, len(request.RuleIDs))
	for _, id := range request.RuleIDs {
		requested[id] = struct{}{}
	}
	var selected []rules.Rule
	for _, rule := range s.rules.List() {
		if len(requested) > 0 {
			if _, ok := requested[rule.ID]; !ok {
				continue
			}
		}
		if !containsMode(rule.Modes, rules.ScanMode(request.Mode)) {
			continue
		}
		usesSelectedRoot := false
		for _, root := range rule.Scan.Roots {
			usesSelectedRoot = usesSelectedRoot || strings.Contains(root, "$SCAN_ROOT")
		}
		if len(request.Roots) > 0 && !usesSelectedRoot {
			continue
		}
		if len(request.Roots) == 0 && usesSelectedRoot {
			continue
		}
		selected = append(selected, rule)
	}
	if len(selected) == 0 {
		return nil, errors.New("no enabled rules match this scan request")
	}
	return selected, nil
}

func containsMode(modes []rules.ScanMode, mode rules.ScanMode) bool {
	for _, candidate := range modes {
		if candidate == mode {
			return true
		}
	}
	return false
}

func (s *Service) run(ctx context.Context, state *jobState, request Request, selectedRules []rules.Rule) {
	state.setStatus(StatusValidating)
	for _, rule := range selectedRules {
		roots := resolveRoots(rule.Scan.Roots, request.Roots)
		for _, root := range roots {
			if ctx.Err() != nil {
				state.finish(StatusCancelled)
				return
			}
			state.setStatus(StatusRunning)
			if err := s.scanRoot(ctx, state, root, rule); err != nil && !errors.Is(err, context.Canceled) {
				state.addError(root, err)
			}
		}
	}
	if ctx.Err() != nil {
		state.finish(StatusCancelled)
		return
	}
	state.finish(StatusCompleted)
}

func (s *Service) scanRoot(ctx context.Context, state *jobState, root string, rule rules.Rule) error {
	if rule.Action.Type == "empty_recycle_bin" {
		return scanRecycleBin(state, root, rule)
	}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		state.incrementScanned(root)
		if matches(root, root, info, rule.Scan) {
			state.addItem(makeItem(root, info, rule))
		}
		return nil
	}
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
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
		info, err := entry.Info()
		if err != nil {
			state.addError(path, err)
			return nil
		}
		if matches(root, path, info, rule.Scan) {
			state.addItem(makeItem(path, info, rule))
		}
		return nil
	})
}

func scanRecycleBin(state *jobState, root string, rule rules.Rule) error {
	info, err := platform.QueryRecycleBin(root)
	if err != nil {
		return err
	}
	state.incrementScanned(info.Root)
	if info.ItemCount == 0 || info.Size == 0 {
		return nil
	}
	hash := sha256.Sum256([]byte(rule.ID + "\x00" + info.Root))
	volumeName := strings.TrimSuffix(info.Root, string(filepath.Separator))
	state.addItem(Item{
		ID: hex.EncodeToString(hash[:12]), RuleID: rule.ID, MatchedRuleIDs: []string{rule.ID},
		Name: volumeName + " 回收站", Path: info.Root, Directory: info.Root,
		Category: rule.Category, Purpose: rule.Purpose, CleanEffect: rule.CleanEffect,
		Recommendation: string(rule.Recommendation), RecommendationReason: rule.RecommendationReason,
		Risk: string(rule.Risk), DefaultSelected: rule.DefaultSelected, Selectable: true,
		Action: rule.Action.Type, RecoveryType: rule.RecoveryType, RequiresAdmin: rule.RequiresAdmin,
		LogicalSize: info.Size, AllocatedSize: info.Size, EstimatedReclaimable: info.Size,
		VolumeID: info.Root, FileID: strconv.FormatInt(info.ItemCount, 10), ModifiedAt: time.Now(),
		HelpSummary: rule.Help.Summary, HelpDetails: rule.Help.Details,
		SpecialWarning: rule.Help.SpecialWarning, ManualSteps: rule.Help.ManualSteps,
	})
	return nil
}

func matches(root, path string, info os.FileInfo, spec rules.ScanSpec) bool {
	if spec.MinSizeBytes > 0 && info.Size() < spec.MinSizeBytes {
		return false
	}
	if minAge := spec.MinimumAge(); minAge > 0 && time.Since(info.ModTime()) < minAge {
		return false
	}
	if len(spec.Extensions) > 0 {
		extension := strings.ToLower(filepath.Ext(path))
		matched := false
		for _, allowed := range spec.Extensions {
			matched = matched || extension == strings.ToLower(allowed)
		}
		if !matched {
			return false
		}
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if len(spec.Include) > 0 && !matchesAny(relative, filepath.Base(path), spec.Include) {
		return false
	}
	if len(spec.Exclude) > 0 && matchesAny(relative, filepath.Base(path), spec.Exclude) {
		return false
	}
	return true
}

func matchesAny(relative, name string, patterns []string) bool {
	relative = filepath.ToSlash(relative)
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(pattern)
		if pattern == "*" || pattern == "**/*" {
			return true
		}
		if matched, _ := filepath.Match(pattern, relative); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

func makeItem(path string, info os.FileInfo, rule rules.Rule) Item {
	metadata := fsmeta.Read(path, info)
	reclaimable := metadata.AllocatedSize
	selectable := rule.Risk != rules.RiskForbidden && rule.Action.Type != "analyze"
	if metadata.LinkCount > 1 || !selectable {
		reclaimable = 0
	}
	hash := sha256.Sum256([]byte(rule.ID + "\x00" + path))
	return Item{
		ID: hex.EncodeToString(hash[:12]), RuleID: rule.ID, MatchedRuleIDs: []string{rule.ID},
		Name: info.Name(), Path: path, Directory: filepath.Dir(path), Extension: strings.ToLower(filepath.Ext(path)),
		Category: rule.Category, Purpose: rule.Purpose, CleanEffect: rule.CleanEffect,
		Recommendation: string(rule.Recommendation), RecommendationReason: rule.RecommendationReason,
		Risk: string(rule.Risk), DefaultSelected: rule.DefaultSelected, Selectable: selectable,
		Action: rule.Action.Type, RecoveryType: rule.RecoveryType, RequiresAdmin: rule.RequiresAdmin,
		RequiresRestart: rule.RequiresRestart, LogicalSize: metadata.LogicalSize,
		AllocatedSize: metadata.AllocatedSize, EstimatedReclaimable: reclaimable,
		VolumeID: metadata.VolumeID, FileID: metadata.FileID, LinkCount: metadata.LinkCount,
		ModifiedAt: info.ModTime(), HelpSummary: rule.Help.Summary, HelpDetails: rule.Help.Details,
		SpecialWarning: rule.Help.SpecialWarning, ManualSteps: rule.Help.ManualSteps,
	}
}

func resolveRoots(patterns, selectedRoots []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, pattern := range patterns {
		if pattern == "$FIXED_VOLUMES" {
			if runtime.GOOS != "windows" {
				continue
			}
			volumes, err := disks.List()
			if err != nil {
				continue
			}
			for _, volume := range volumes {
				if volume.Ready {
					appendUniqueRoot(&result, seen, volume.MountPoint)
				}
			}
			continue
		}
		candidates := []string{pattern}
		if strings.Contains(pattern, "$SCAN_ROOT") {
			candidates = make([]string, 0, len(selectedRoots))
			for _, selected := range selectedRoots {
				candidates = append(candidates, strings.ReplaceAll(pattern, "$SCAN_ROOT", selected))
			}
		}
		for _, candidate := range candidates {
			candidate = expandPercentEnvironment(candidate)
			if strings.ContainsAny(candidate, "*?[") {
				matches, _ := filepath.Glob(candidate)
				for _, match := range matches {
					appendUniqueRoot(&result, seen, match)
				}
				continue
			}
			appendUniqueRoot(&result, seen, candidate)
		}
	}
	return result
}

func expandPercentEnvironment(value string) string {
	for start := strings.IndexByte(value, '%'); start >= 0; start = strings.IndexByte(value, '%') {
		endRelative := strings.IndexByte(value[start+1:], '%')
		if endRelative < 0 {
			break
		}
		end := start + 1 + endRelative
		key := value[start+1 : end]
		replacement, exists := os.LookupEnv(key)
		if !exists {
			break
		}
		value = value[:start] + replacement + value[end+1:]
	}
	return filepath.Clean(value)
}

func appendUniqueRoot(result *[]string, seen map[string]struct{}, root string) {
	clean := filepath.Clean(root)
	if _, exists := seen[clean]; exists {
		return
	}
	seen[clean] = struct{}{}
	*result = append(*result, clean)
}

func (state *jobState) snapshot() Job {
	state.mu.RLock()
	defer state.mu.RUnlock()
	result := state.job
	result.Errors = append([]ErrorItem(nil), state.job.Errors...)
	return result
}

func (state *jobState) setStatus(status Status) {
	state.mu.Lock()
	state.job.Status = status
	state.mu.Unlock()
}

func (state *jobState) finish(status Status) {
	now := time.Now()
	state.mu.Lock()
	state.job.Status = status
	state.job.CurrentPath = ""
	state.job.CompletedAt = &now
	state.mu.Unlock()
}

func (state *jobState) incrementScanned(path string) {
	state.mu.Lock()
	state.job.ScannedFiles++
	state.job.CurrentPath = path
	state.mu.Unlock()
}

func (state *jobState) addError(path string, err error) {
	state.mu.Lock()
	state.job.ErrorCount++
	if len(state.job.Errors) < maxRecordedErrors {
		state.job.Errors = append(state.job.Errors, ErrorItem{Path: path, Message: err.Error()})
	}
	state.mu.Unlock()
}

func (state *jobState) addItem(item Item) {
	state.mu.Lock()
	defer state.mu.Unlock()
	canonical := strings.ToLower(filepath.Clean(item.Path))
	if existingIndex, ok := state.itemIndex[canonical]; ok {
		existing := &state.items[existingIndex]
		if !containsString(existing.MatchedRuleIDs, item.RuleID) {
			existing.MatchedRuleIDs = append(existing.MatchedRuleIDs, item.RuleID)
		}
		if riskRank(item.Risk) > riskRank(existing.Risk) {
			deltaLogical := item.LogicalSize - existing.LogicalSize
			deltaAllocated := item.AllocatedSize - existing.AllocatedSize
			deltaReclaimable := item.EstimatedReclaimable - existing.EstimatedReclaimable
			state.job.LogicalBytes += deltaLogical
			state.job.AllocatedBytes += deltaAllocated
			state.job.EstimatedReclaimable += deltaReclaimable
			item.MatchedRuleIDs = existing.MatchedRuleIDs
			state.items[existingIndex] = item
		}
		return
	}
	state.itemIndex[canonical] = len(state.items)
	state.items = append(state.items, item)
	state.job.MatchedFiles++
	state.job.LogicalBytes += item.LogicalSize
	state.job.AllocatedBytes += item.AllocatedSize
	state.job.EstimatedReclaimable += item.EstimatedReclaimable
}

func riskRank(risk string) int {
	switch risk {
	case string(rules.RiskForbidden):
		return 4
	case string(rules.RiskHigh):
		return 3
	case string(rules.RiskMedium):
		return 2
	default:
		return 1
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (s *Service) Job(id string) (Job, error) {
	state, err := s.state(id)
	if err != nil {
		return Job{}, err
	}
	return state.snapshot(), nil
}

func (s *Service) Items(id string, offset, limit int) (ItemPage, error) {
	state, err := s.state(id)
	if err != nil {
		return ItemPage{}, err
	}
	state.mu.RLock()
	items := append(make([]Item, 0, len(state.items)), state.items...)
	state.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool {
		if items[i].AllocatedSize == items[j].AllocatedSize {
			return items[i].Path < items[j].Path
		}
		return items[i].AllocatedSize > items[j].AllocatedSize
	})
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	if offset > len(items) {
		offset = len(items)
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return ItemPage{Items: items[offset:end], Total: len(items), Offset: offset, Limit: limit}, nil
}

func (s *Service) Folders(id string) ([]*Folder, error) {
	state, err := s.state(id)
	if err != nil {
		return nil, err
	}
	state.mu.RLock()
	items := append(make([]Item, 0, len(state.items)), state.items...)
	state.mu.RUnlock()
	return buildFolders(items), nil
}

func (s *Service) Cancel(id string) error {
	state, err := s.state(id)
	if err != nil {
		return err
	}
	state.mu.Lock()
	if state.job.Status == StatusCompleted || state.job.Status == StatusCancelled || state.job.Status == StatusFailed {
		state.mu.Unlock()
		return nil
	}
	state.job.Status = StatusCancelling
	cancel := state.cancel
	state.mu.Unlock()
	cancel()
	return nil
}

func (s *Service) state(id string) (*jobState, error) {
	s.mu.RLock()
	state, ok := s.jobs[id]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("scan job %q not found", id)
	}
	return state, nil
}
