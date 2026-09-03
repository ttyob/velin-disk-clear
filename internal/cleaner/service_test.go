package cleaner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ai-clear/internal/rules"
	"ai-clear/internal/scanner"
)

func TestPermanentDeleteAndHistory(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "old.log")
	if err := os.WriteFile(path, []byte("diagnostic"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-31 * 24 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	scan, scanID := newCompletedScan(t, root)
	page, err := scan.Items(scanID, 0, 20)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("items = %d, err = %v", len(page.Items), err)
	}
	dataDir := filepath.Join(t.TempDir(), "data")
	service, err := New(scan, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := service.BuildPlan(BuildRequest{ScanID: scanID, ItemIDs: []string{page.Items[0].ID}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.PermanentDeleteCount != 1 || plan.ManualSelectedCount != 1 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	result, err := service.Execute(ExecuteRequest{PlanID: plan.ID, ConfirmationToken: plan.ConfirmationToken})
	if err != nil {
		t.Fatal(err)
	}
	if result.Succeeded != 1 || result.DeletedBytes == 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("source still exists: %v", err)
	}
	reloaded, err := New(scan, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.History()) != 1 {
		t.Fatal("SQLite history was not reloaded")
	}
}

func TestChangedFileIsSkipped(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "old.log")
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-31 * 24 * time.Hour)
	_ = os.Chtimes(path, old, old)
	scan, id := newCompletedScan(t, root)
	page, _ := scan.Items(id, 0, 20)
	service, _ := New(scan, filepath.Join(t.TempDir(), "data"))
	plan, err := service.BuildPlan(BuildRequest{ScanID: id, ItemIDs: []string{page.Items[0].ID}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed content"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := service.Execute(ExecuteRequest{PlanID: plan.ID, ConfirmationToken: plan.ConfirmationToken})
	if err != nil {
		t.Fatal(err)
	}
	if result.Skipped != 1 || result.Succeeded != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func newCompletedScan(t *testing.T, root string) (*scanner.Service, string) {
	t.Helper()
	catalog, err := rules.LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	service := scanner.New(catalog)
	job, err := service.Start(context.Background(), scanner.Request{Mode: "standard", Roots: []string{root}, RuleIDs: []string{"generic.old_logs"}})
	if err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		current, err := service.Job(job.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.Status == scanner.StatusCompleted {
			return service, job.ID
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("scan did not complete")
	return nil, ""
}
