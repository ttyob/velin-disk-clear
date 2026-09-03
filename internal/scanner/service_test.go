package scanner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ai-clear/internal/rules"
)

func TestDeepScanFindsOldLogAndLargeFile(t *testing.T) {
	root := t.TempDir()
	oldLog := filepath.Join(root, "old.log")
	if err := os.WriteFile(oldLog, []byte("old log"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-31 * 24 * time.Hour)
	if err := os.Chtimes(oldLog, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	largePath := filepath.Join(root, "archive.iso")
	large, err := os.Create(largePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := large.Truncate(1024*1024*1024 + 1); err != nil {
		large.Close()
		t.Fatal(err)
	}
	if err := large.Close(); err != nil {
		t.Fatal(err)
	}

	outside := filepath.Join(t.TempDir(), "outside.log")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(outside, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked.log")); err != nil {
		t.Fatal(err)
	}

	ruleService, err := rules.LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	service := New(ruleService)
	job, err := service.Start(context.Background(), Request{Mode: "deep", Roots: []string{root}})
	if err != nil {
		t.Fatal(err)
	}
	job = awaitJob(t, service, job.ID)
	if job.Status != StatusCompleted {
		t.Fatalf("status = %s, want completed", job.Status)
	}

	page, err := service.Items(job.ID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 {
		t.Fatalf("matched items = %d, want 2: %#v", page.Total, page.Items)
	}
	for _, item := range page.Items {
		if item.DefaultSelected {
			t.Fatalf("custom analysis item %s must not be selected by default", item.Path)
		}
		if item.Path == filepath.Join(root, "linked.log") {
			t.Fatal("symbolic link must not be scanned")
		}
		if !item.Selectable {
			t.Fatalf("user-confirmed custom item %s should be selectable", item.Path)
		}
	}

	folders, err := service.Folders(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) == 0 || folders[0].FileCount != 2 {
		t.Fatalf("unexpected folder aggregation: %#v", folders)
	}
}

func TestEmptyResultsUseJSONArrays(t *testing.T) {
	ruleService, err := rules.LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	service := New(ruleService)
	job, err := service.Start(context.Background(), Request{Mode: "deep", Roots: []string{t.TempDir()}, RuleIDs: []string{"generic.large_files"}})
	if err != nil {
		t.Fatal(err)
	}
	job = awaitJob(t, service, job.ID)
	if job.Status != StatusCompleted {
		t.Fatalf("status = %s, want completed", job.Status)
	}
	page, err := service.Items(job.ID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	folders, err := service.Folders(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	pageJSON, _ := json.Marshal(page)
	foldersJSON, _ := json.Marshal(folders)
	if !strings.Contains(string(pageJSON), `"items":[]`) || string(foldersJSON) != "[]" {
		t.Fatalf("empty scan results must use arrays, got page=%s folders=%s", pageJSON, foldersJSON)
	}
}

func TestWindowsUpdateAnalysisCannotBecomeCleanCandidate(t *testing.T) {
	packagePath := filepath.Join(t.TempDir(), "update.cab")
	if err := os.WriteFile(packagePath, []byte("update payload"), 0o600); err != nil {
		t.Fatal(err)
	}

	ruleService, err := rules.LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	rule, ok := ruleService.Get("windows.update_download_analysis")
	if !ok {
		t.Fatal("Windows Update analysis rule is missing")
	}
	info, err := os.Stat(packagePath)
	if err != nil {
		t.Fatal(err)
	}
	item := makeItem(packagePath, info, rule)
	service := New(ruleService)
	service.jobs["analysis-scan"] = &jobState{
		job:   Job{ID: "analysis-scan", Status: StatusCompleted},
		items: []Item{item},
	}
	if item.Selectable || item.DefaultSelected || item.Action != "analyze" {
		t.Fatalf("unsafe Windows Update analysis item: %#v", item)
	}
	if _, err := service.CleanCandidates("analysis-scan", []string{item.ID}); err == nil {
		t.Fatal("Windows Update analysis item must be rejected as a clean candidate")
	}
}

func awaitJob(t *testing.T, service *Service, id string) Job {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		job, err := service.Job(id)
		if err != nil {
			t.Fatal(err)
		}
		if job.Status == StatusCompleted || job.Status == StatusCancelled || job.Status == StatusFailed {
			return job
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("scan did not finish")
	return Job{}
}
