package cleaner

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"ai-clear/internal/fsmeta"
	"ai-clear/internal/platform"
	"ai-clear/internal/scanner"
	"ai-clear/internal/storage"
)

const planLifetime = 15 * time.Minute

type plannedItem struct {
	view         PlanItem
	snapshot     scanner.Item
	allowedRoots []string
}

type storedPlan struct {
	view  Plan
	items []plannedItem
}

type Service struct {
	scanner *scanner.Service
	store   *storage.Store
	mu      sync.RWMutex
	plans   map[string]*storedPlan
	results []Result
}

func New(scanService *scanner.Service, dataDir string) (*Service, error) {
	if scanService == nil {
		return nil, errors.New("scanner service is required")
	}
	if dataDir == "" {
		return nil, errors.New("data directory is required")
	}
	store, err := storage.Open(dataDir)
	if err != nil {
		return nil, fmt.Errorf("open application database: %w", err)
	}
	service := &Service{scanner: scanService, store: store, plans: make(map[string]*storedPlan)}
	if err := service.loadDatabase(); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) BuildPlan(request BuildRequest) (Plan, error) {
	candidates, err := s.scanner.CleanCandidates(request.ScanID, request.ItemIDs)
	if err != nil {
		return Plan{}, err
	}
	now := time.Now()
	plan := Plan{ID: "clean-plan-" + randomID(12), ScanID: request.ScanID, Status: "awaiting_confirmation", ConfirmationToken: randomID(24), CreatedAt: now, ExpiresAt: now.Add(planLifetime)}
	stored := &storedPlan{items: make([]plannedItem, 0, len(candidates))}
	for _, candidate := range candidates {
		item := candidate.Item
		if item.Action != "permanent_delete" && item.Action != "empty_recycle_bin" {
			return Plan{}, fmt.Errorf("action %q is not executable", item.Action)
		}
		if !pathWithinAny(item.Path, candidate.AllowedRoots) {
			return Plan{}, fmt.Errorf("item %q is outside its allowed roots", item.ID)
		}
		view := PlanItem{ID: item.ID, RuleID: item.RuleID, Name: item.Name, Path: item.Path, Risk: item.Risk, Action: item.Action, RecoveryType: "none", AllocatedSize: item.AllocatedSize, EstimatedReclaimable: item.EstimatedReclaimable, DefaultSelected: item.DefaultSelected, RequiresAdmin: item.RequiresAdmin}
		plan.Items = append(plan.Items, view)
		stored.items = append(stored.items, plannedItem{view: view, snapshot: item, allowedRoots: append([]string(nil), candidate.AllowedRoots...)})
		plan.EstimatedReclaimable += item.EstimatedReclaimable
		if item.DefaultSelected {
			plan.DefaultSelectedCount++
		} else {
			plan.ManualSelectedCount++
		}
		switch item.Risk {
		case "low":
			plan.LowRiskCount++
		case "medium":
			plan.MediumRiskCount++
		default:
			plan.HighRiskCount++
		}
	}
	plan.ItemCount = len(plan.Items)
	plan.PermanentDeleteCount = plan.ItemCount
	stored.view = plan
	s.mu.Lock()
	s.plans[plan.ID] = stored
	s.mu.Unlock()
	return plan, nil
}

func (s *Service) Execute(request ExecuteRequest) (Result, error) {
	s.mu.Lock()
	stored, ok := s.plans[request.PlanID]
	if !ok {
		s.mu.Unlock()
		return Result{}, errors.New("clean plan not found")
	}
	if stored.view.Status != "awaiting_confirmation" {
		s.mu.Unlock()
		return Result{}, errors.New("clean plan is not executable")
	}
	if time.Now().After(stored.view.ExpiresAt) {
		stored.view.Status = "expired"
		s.mu.Unlock()
		return Result{}, errors.New("clean plan has expired; scan again")
	}
	if request.ConfirmationToken == "" || request.ConfirmationToken != stored.view.ConfirmationToken {
		s.mu.Unlock()
		return Result{}, errors.New("invalid confirmation token")
	}
	stored.view.Status = "running"
	stored.view.ConfirmationToken = ""
	items := append([]plannedItem(nil), stored.items...)
	s.mu.Unlock()

	result := Result{ID: "clean-" + randomID(12), PlanID: request.PlanID, Status: "running", StartedAt: time.Now()}
	for _, item := range items {
		entry := ItemResult{ItemID: item.view.ID, Path: item.view.Path, Action: item.view.Action}
		if err := revalidate(item); err != nil {
			entry.Status, entry.Error = "skipped", err.Error()
			result.Skipped++
			result.Items = append(result.Items, entry)
			continue
		}
		var cleanErr error
		switch item.view.Action {
		case "permanent_delete":
			cleanErr = os.Remove(item.view.Path)
		case "empty_recycle_bin":
			cleanErr = platform.EmptyRecycleBin(item.view.Path)
		default:
			cleanErr = fmt.Errorf("unsupported clean action %q", item.view.Action)
		}
		if cleanErr != nil {
			entry.Status, entry.Error = "failed", cleanErr.Error()
			result.Failed++
		} else {
			entry.Status = "succeeded"
			entry.BytesProcessed = item.view.AllocatedSize
			result.Succeeded++
			result.DeletedBytes += item.view.AllocatedSize
			result.ActuallyReclaimed += item.view.EstimatedReclaimable
		}
		result.Items = append(result.Items, entry)
	}
	result.CompletedAt = time.Now()
	if result.Failed > 0 || result.Skipped > 0 {
		result.Status = "completed_with_issues"
	} else {
		result.Status = "completed"
	}
	s.mu.Lock()
	stored.view.Status = result.Status
	s.results = append([]Result{result}, s.results...)
	err := s.saveResult(result)
	s.mu.Unlock()
	if err != nil {
		return result, fmt.Errorf("clean completed but history could not be saved: %w", err)
	}
	return result, nil
}

func (s *Service) History() []Result {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append(make([]Result, 0, len(s.results)), s.results...)
}

func revalidate(item plannedItem) error {
	if !pathWithinAny(item.view.Path, item.allowedRoots) {
		return errors.New("path is outside the rule allowlist")
	}
	if item.view.Action == "empty_recycle_bin" {
		info, err := platform.QueryRecycleBin(item.view.Path)
		if err != nil {
			return err
		}
		if info.Root != item.snapshot.VolumeID || info.Size != item.snapshot.AllocatedSize || strconv.FormatInt(info.ItemCount, 10) != item.snapshot.FileID {
			return errors.New("recycle bin contents changed after scanning")
		}
		return nil
	}
	if !pathWithinResolvedRoots(item.view.Path, item.allowedRoots) {
		return errors.New("resolved path is outside the rule allowlist")
	}
	info, err := os.Lstat(item.view.Path)
	if err != nil {
		return fmt.Errorf("file is no longer available: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("file type changed after scanning")
	}
	meta := fsmeta.Read(item.view.Path, info)
	if info.Size() != item.snapshot.LogicalSize || !info.ModTime().Equal(item.snapshot.ModifiedAt) {
		return errors.New("file changed after scanning")
	}
	if item.snapshot.FileID != "" && meta.FileID != "" && (meta.FileID != item.snapshot.FileID || meta.VolumeID != item.snapshot.VolumeID) {
		return errors.New("file identity changed after scanning")
	}
	return nil
}

func pathWithinResolvedRoots(path string, roots []string) bool {
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return false
	}
	resolvedPath := filepath.Join(parent, filepath.Base(path))
	for _, root := range roots {
		resolvedRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			resolvedParent, parentErr := filepath.EvalSymlinks(filepath.Dir(root))
			if parentErr != nil {
				continue
			}
			resolvedRoot = filepath.Join(resolvedParent, filepath.Base(root))
		}
		if pathWithinAny(resolvedPath, []string{resolvedRoot}) {
			return true
		}
	}
	return false
}

func pathWithinAny(path string, roots []string) bool {
	path = filepath.Clean(path)
	for _, root := range roots {
		root = filepath.Clean(root)
		if samePath(path, root) {
			return true
		}
		rel, err := filepath.Rel(root, path)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel) {
			return true
		}
	}
	return false
}
func samePath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func (s *Service) loadDatabase() error {
	records, err := s.store.CleanRecords()
	if err != nil {
		return err
	}
	for _, record := range records {
		var result Result
		if err := json.Unmarshal([]byte(record.Payload), &result); err != nil {
			return fmt.Errorf("decode clean history: %w", err)
		}
		s.results = append(s.results, result)
	}
	return nil
}
func (s *Service) saveResult(result Result) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return s.store.SaveClean(storage.CleanRecord{ID: result.ID, PlanID: result.PlanID, Status: result.Status, Payload: string(payload), StartedAt: result.StartedAt, CompletedAt: result.CompletedAt, ReclaimedBytes: result.ActuallyReclaimed})
}

func randomID(size int) string {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(data)
}
