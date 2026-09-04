package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	cleaningagent "ai-clear/internal/ai/agent"
	"ai-clear/internal/ai/provider"
	"ai-clear/internal/cleaner"
	"ai-clear/internal/scanner"
)

// serveDevAPI exposes the same App methods used by Wails to the browser-only
// Vite preview. It is intentionally opt-in and intended for local development.
func serveDevAPI(app *App, addr string) error {
	if app == nil {
		return errors.New("app is required")
	}
	server := &http.Server{Addr: addr, Handler: devAPIHandler(app)}
	return server.ListenAndServe()
}

func devAPIHandler(app *App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		if err := dispatchDevAPI(w, r, app); err != nil {
			writeDevError(w, err)
		}
	})
}

func dispatchDevAPI(w http.ResponseWriter, r *http.Request, app *App) error {
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	switch {
	case r.Method == http.MethodGet && path == "dashboard":
		value, err := app.Dashboard()
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodGet && path == "rules":
		value, err := app.Rules()
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodGet && path == "rules/stats":
		value, err := app.RuleStatistics()
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodGet && path == "update/check":
		value, err := app.CheckForUpdates()
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodPost && path == "update/download":
		value, err := app.DownloadUpdate()
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodPost && path == "update/install":
		return writeDevJSON(w, map[string]string{"status": "installing"}, app.InstallUpdate())
	case r.Method == http.MethodGet && path == "network/settings":
		value, err := app.NetworkSettings()
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodPost && path == "network/settings":
		var input NetworkSettings
		if err := decodeDevJSON(r, &input); err != nil {
			return err
		}
		value, err := app.SaveNetworkSettings(input)
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodGet && path == "scan/settings":
		value, err := app.ScanSettings()
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodPost && path == "scan/settings":
		var input ScanSettings
		if err := decodeDevJSON(r, &input); err != nil {
			return err
		}
		value, err := app.SaveScanSettings(input)
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodPost && path == "rules/sync":
		value, err := app.SyncRules()
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodGet && path == "provider":
		value, err := app.AIProvider()
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodPost && path == "provider/save":
		var input provider.ConfigInput
		if err := decodeDevJSON(r, &input); err != nil {
			return err
		}
		value, err := app.SaveAIProvider(input)
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodPost && path == "provider/test":
		var input provider.ConfigInput
		if err := decodeDevJSON(r, &input); err != nil {
			return err
		}
		value, err := app.TestAIProvider(input)
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodPost && path == "scan":
		var input scanner.Request
		if err := decodeDevJSON(r, &input); err != nil {
			return err
		}
		value, err := app.StartScan(input)
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodGet && strings.HasPrefix(path, "scan/"):
		return dispatchDevScan(w, r, app, strings.TrimPrefix(path, "scan/"))
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "scan/"):
		id := strings.TrimPrefix(path, "scan/")
		return writeDevJSON(w, map[string]string{"status": "cancelled"}, app.CancelScan(id))
	case r.Method == http.MethodGet && path == "clean/history":
		value, err := app.CleanHistory()
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodPost && path == "clean/plan":
		var input cleaner.BuildRequest
		if err := decodeDevJSON(r, &input); err != nil {
			return err
		}
		value, err := app.BuildCleanPlan(input)
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodPost && path == "clean/execute":
		var input cleaner.ExecuteRequest
		if err := decodeDevJSON(r, &input); err != nil {
			return err
		}
		value, err := app.ExecuteCleanPlan(input)
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodPost && path == "agent":
		var input cleaningagent.Request
		if err := decodeDevJSON(r, &input); err != nil {
			return err
		}
		value, err := app.RunCleaningAgent(input)
		return writeDevJSON(w, value, err)
	case r.Method == http.MethodGet && path == "agent/presets":
		return writeDevJSON(w, app.CleaningAgentPresets(), nil)
	case r.Method == http.MethodPost && path == "settings":
		var input struct {
			Action string `json:"action"`
		}
		if err := decodeDevJSON(r, &input); err != nil {
			return err
		}
		return writeDevJSON(w, map[string]string{"status": "ok"}, app.OpenSystemSettings(input.Action))
	default:
		http.NotFound(w, r)
		return nil
	}
}

func dispatchDevScan(w http.ResponseWriter, r *http.Request, app *App, path string) error {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return nil
	}
	id := parts[0]
	if len(parts) == 1 {
		value, err := app.ScanJob(id)
		return writeDevJSON(w, value, err)
	}
	switch parts[1] {
	case "items":
		offset, limit := queryInt(r, "offset", 0), queryInt(r, "limit", 200)
		value, err := app.ScanItems(id, offset, limit)
		return writeDevJSON(w, value, err)
	case "folders":
		value, err := app.ScanFolders(id)
		return writeDevJSON(w, value, err)
	default:
		http.NotFound(w, r)
		return nil
	}
}

func queryInt(r *http.Request, key string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func decodeDevJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON request: %w", err)
	}
	return nil
}

func writeDevJSON(w http.ResponseWriter, value any, err error) error {
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(value)
}

func writeDevError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if strings.Contains(err.Error(), "not found") {
		status = http.StatusNotFound
	}
	http.Error(w, err.Error(), status)
}
