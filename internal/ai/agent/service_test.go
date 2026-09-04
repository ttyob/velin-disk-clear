package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ai-clear/internal/ai/provider"
	"ai-clear/internal/rules"
	"ai-clear/internal/scanner"
)

func TestRunUsesRedactedSnapshotAndValidatedJSON(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "private-name.log")
	if err := os.WriteFile(file, []byte("log"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-31 * 24 * time.Hour)
	_ = os.Chtimes(file, old, old)
	catalog, _ := rules.LoadBuiltin()
	scanService := scanner.New(catalog)
	job, err := scanService.Start(context.Background(), scanner.Request{Mode: "standard", Roots: []string{root}, RuleIDs: []string{"generic.old_logs"}})
	if err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(time.Second); ; {
		current, _ := scanService.Job(job.ID)
		if current.Status == scanner.StatusCompleted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("scan timed out")
		}
		time.Sleep(time.Millisecond)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_, _ = io.WriteString(w, `{"data":[{"id":"agent-model"}]}`)
		case "/v1/chat/completions":
			data, _ := io.ReadAll(r.Body)
			var request map[string]any
			_ = json.Unmarshal(data, &request)
			if _, ok := request["tools"]; ok {
				_, _ = io.WriteString(w, `{"choices":[{"message":{"tool_calls":[{"type":"function","function":{"name":"get_disk_overview","arguments":"{}"}}]}}]}`)
				return
			}
			if !strings.Contains(string(data), root) {
				t.Errorf("Agent request did not include the scanner path required for analysis")
			}
			_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"{\"summary\":\"优先检查规则候选\",\"items\":[]}"}}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	providerService, err := provider.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tested, err := providerService.Test(context.Background(), provider.ConfigInput{BaseURL: server.URL + "/v1", Model: "agent-model"})
	if err != nil || !tested.CapabilityOK {
		t.Fatalf("provider test: %+v %v", tested, err)
	}
	result, err := New(providerService, scanService).Run(context.Background(), Request{Objective: "安全释放空间", ScanID: job.ID})
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary == "" || result.ProviderModel != "agent-model" || result.Mode != "scan" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRunForcesOneToolCallThenAnalyzesToolResult(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "candidate.tmp")
	if err := os.WriteFile(file, []byte("temporary data"), 0o600); err != nil {
		t.Fatal(err)
	}
	var toolCallRequests, finalRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_, _ = io.WriteString(w, `{"data":[{"id":"agent-model"}]}`)
		case "/v1/chat/completions":
			data, _ := io.ReadAll(r.Body)
			var request map[string]any
			if err := json.Unmarshal(data, &request); err != nil {
				t.Fatal(err)
			}
			if _, hasTools := request["tools"]; hasTools {
				// Provider capability testing uses a different tool; the run itself
				// must force scan_files rather than leave it to model discretion.
				tools := request["tools"].([]any)
				name := tools[0].(map[string]any)["function"].(map[string]any)["name"]
				if name == "get_disk_overview" {
					_, _ = io.WriteString(w, `{"choices":[{"message":{"tool_calls":[{"id":"capability","type":"function","function":{"name":"get_disk_overview","arguments":"{}"}}]}}]}`)
					return
				}
				toolCallRequests++
				choice := request["tool_choice"].(map[string]any)
				if choice["function"].(map[string]any)["name"] != "scan_files" {
					t.Error("scan_files was not forced")
				}
				// Simulate providers that honor the tool call but occasionally omit a
				// required argument. The service must fall back to request.Roots.
				_, _ = io.WriteString(w, `{"choices":[{"message":{"tool_calls":[{"id":"scan-1","type":"function","function":{"name":"scan_files","arguments":"{}"}}]}}]}`)
				return
			}
			finalRequests++
			if !strings.Contains(string(data), file) {
				t.Error("final analysis did not receive scanned file metadata")
			}
			_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"{\"summary\":\"已分析\",\"items\":[]}"}}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	providerService, err := provider.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tested, err := providerService.Test(context.Background(), provider.ConfigInput{BaseURL: server.URL + "/v1", Model: "agent-model"})
	if err != nil || !tested.CapabilityOK {
		t.Fatalf("provider test: %+v %v", tested, err)
	}
	catalog, _ := rules.LoadBuiltin()
	result, err := New(providerService, scanner.New(catalog)).Run(context.Background(), Request{Mode: "scan", Objective: "检查可清理文件", Roots: []string{root}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary != "已分析" || toolCallRequests != 1 || finalRequests != 1 {
		t.Fatalf("unexpected calls/result: tools=%d final=%d result=%+v", toolCallRequests, finalRequests, result)
	}
}
