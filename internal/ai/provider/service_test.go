package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSaveAndTestDoesNotPersistSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v1/chat/completions" {
			_, _ = io.WriteString(w, `{"choices":[{"message":{"tool_calls":[{"type":"function","function":{"name":"get_disk_overview","arguments":"{}"}}]}}]}`)
			return
		}
		if request.URL.Path != "/v1/models" {
			http.NotFound(w, request)
			return
		}
		if request.Header.Get("Authorization") != "Bearer top-secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, `{"data":[{"id":"clean-model"}]}`)
	}))
	defer server.Close()
	dir := t.TempDir()
	service, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Test(context.Background(), ConfigInput{BaseURL: server.URL + "/v1", Model: "clean-model", APIKey: "top-secret"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || !result.ModelFound || !result.CapabilityOK {
		t.Fatalf("unexpected test result: %+v", result)
	}
	payload, found, err := service.store.Provider("provider-default")
	if err != nil || !found {
		t.Fatalf("provider not stored: %v", err)
	}
	if strings.Contains(payload, "top-secret") {
		t.Fatal("API key was persisted in provider config")
	}
}

func TestRejectsInsecureRemoteURL(t *testing.T) {
	service, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Save(ConfigInput{BaseURL: "http://example.com/v1", Model: "model"})
	if err == nil {
		t.Fatal("expected insecure remote URL rejection")
	}
}
