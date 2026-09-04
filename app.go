package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	cleaningagent "ai-clear/internal/ai/agent"
	"ai-clear/internal/ai/provider"
	"ai-clear/internal/cleaner"
	"ai-clear/internal/disks"
	"ai-clear/internal/network"
	"ai-clear/internal/platform"
	"ai-clear/internal/rules"
	"ai-clear/internal/scanner"
	"ai-clear/internal/storage"
	"ai-clear/internal/updater"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Dashboard struct {
	Volumes   []disks.Volume `json:"volumes"`
	RuleCount int            `json:"rule_count"`
	Version   string         `json:"version"`
}

const AppVersion = "0.2.2"

type UpdateInfo struct {
	Available      bool   `json:"available"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	TagName        string `json:"tag_name"`
	ReleaseName    string `json:"release_name"`
	Notes          string `json:"notes"`
	PublishedAt    string `json:"published_at"`
	AssetName      string `json:"asset_name"`
	AssetSize      int64  `json:"asset_size"`
	Digest         string `json:"digest"`
	CheckedAt      string `json:"checked_at"`
}

type UpdateDownload struct {
	Version  string `json:"version"`
	Size     int64  `json:"size"`
	Verified bool   `json:"verified"`
}

type NetworkSettings struct {
	HTTPProxy string `json:"http_proxy"`
}

// ScanSettings are persisted beside the application database so portable
// installs keep the user's scan boundaries with the program.
type ScanSettings struct {
	ExcludeRoots []string `json:"exclude_roots"`
}

type App struct {
	ctx      context.Context
	rules    *rules.Service
	scanner  *scanner.Service
	cleaner  *cleaner.Service
	provider *provider.Service
	agent    *cleaningagent.Service
	updater  *updater.Service
	settings *storage.Store
	initErr  error
}

func NewApp() *App {
	ruleService, err := rules.LoadBuiltin()
	app := &App{rules: ruleService, initErr: err}
	if err == nil {
		app.scanner = scanner.New(ruleService)
		dataDir, dataErr := applicationDataDir()
		if dataErr != nil {
			app.initErr = dataErr
			return app
		}
		app.settings, app.initErr = storage.Open(dataDir)
		if app.initErr != nil {
			return app
		}
		app.updater = updater.New(dataDir)
		var savedNetwork NetworkSettings
		if found, settingsErr := app.settings.GetSetting("network", &savedNetwork); settingsErr != nil {
			app.initErr = settingsErr
			return app
		} else if !found {
			savedNetwork = NetworkSettings{}
		}
		if settingsErr := app.applyNetworkSettings(savedNetwork); settingsErr != nil {
			app.initErr = settingsErr
			return app
		}
		app.cleaner, app.initErr = cleaner.New(app.scanner, dataDir)
		if app.initErr == nil {
			app.provider, app.initErr = provider.New(dataDir)
			if app.initErr == nil {
				if app.initErr = app.provider.SetProxy(savedNetwork.HTTPProxy); app.initErr != nil {
					return app
				}
				app.agent = cleaningagent.New(app.provider, app.scanner)
			}
		}
	}
	return app
}

func applicationDataDir() (string, error) {
	baseDir, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate application executable: %w", err)
	}
	baseDir = filepath.Dir(baseDir)
	if hasArgument("--dev-api") || isWithin(baseDir, os.TempDir()) {
		if workingDir, workingErr := os.Getwd(); workingErr == nil {
			baseDir = workingDir
		}
	}
	dataDir := filepath.Join(baseDir, "data")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if configDir, configErr := os.UserConfigDir(); configErr == nil {
			for _, legacyName := range []string{"Velin Disk Clear", "AI Clear"} {
				legacyDir := filepath.Join(configDir, legacyName)
				if _, legacyErr := os.Stat(legacyDir); legacyErr == nil {
					if err := os.Rename(legacyDir, dataDir); err != nil {
						return "", fmt.Errorf("migrate legacy application data: %w", err)
					}
					break
				}
			}
		}
	} else if err != nil {
		return "", fmt.Errorf("check application data directory: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return "", fmt.Errorf("create application data directory: %w", err)
	}
	return dataDir, nil
}

func isWithin(path, root string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) && !filepath.IsAbs(relative)
}

func hasArgument(target string) bool {
	for _, argument := range os.Args[1:] {
		if argument == target {
			return true
		}
	}
	return false
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Update checks are deliberately silent and asynchronous so a slow or
	// unavailable network never delays the first usable screen.
	go func() {
		info, err := a.CheckForUpdates()
		if err == nil {
			runtime.EventsEmit(ctx, "velin:update-available", info)
		}
	}()
}

func (a *App) NetworkSettings() (NetworkSettings, error) {
	if a.initErr != nil {
		return NetworkSettings{}, a.initErr
	}
	var value NetworkSettings
	if a.settings == nil {
		return value, nil
	}
	_, err := a.settings.GetSetting("network", &value)
	return value, err
}

func (a *App) SaveNetworkSettings(value NetworkSettings) (NetworkSettings, error) {
	if a.initErr != nil {
		return NetworkSettings{}, a.initErr
	}
	value.HTTPProxy = strings.TrimSpace(value.HTTPProxy)
	if value.HTTPProxy != "" {
		parsed, err := url.Parse(value.HTTPProxy)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return NetworkSettings{}, errors.New("HTTP 代理地址无效，应为 http://host:port")
		}
		if _, err := network.HTTPClient(value.HTTPProxy, 30*time.Second, nil); err != nil {
			return NetworkSettings{}, err
		}
	}
	if err := a.applyNetworkSettings(value); err != nil {
		return NetworkSettings{}, err
	}
	if a.settings != nil {
		if err := a.settings.SaveSetting("network", value); err != nil {
			return NetworkSettings{}, err
		}
	}
	return value, nil
}

func (a *App) ScanSettings() (ScanSettings, error) {
	if a.initErr != nil {
		return ScanSettings{}, a.initErr
	}
	value := ScanSettings{ExcludeRoots: []string{}}
	if a.settings == nil {
		return value, nil
	}
	if _, err := a.settings.GetSetting("scan", &value); err != nil {
		return ScanSettings{}, err
	}
	value.ExcludeRoots = normalizeExcludeRoots(value.ExcludeRoots)
	return value, nil
}

func (a *App) SaveScanSettings(value ScanSettings) (ScanSettings, error) {
	if a.initErr != nil {
		return ScanSettings{}, a.initErr
	}
	value.ExcludeRoots = normalizeExcludeRoots(value.ExcludeRoots)
	if a.settings != nil {
		if err := a.settings.SaveSetting("scan", value); err != nil {
			return ScanSettings{}, err
		}
	}
	return value, nil
}

func normalizeExcludeRoots(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		clean := filepath.Clean(value)
		key := strings.ToLower(clean)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, clean)
	}
	return result
}

func (a *App) applyNetworkSettings(value NetworkSettings) error {
	if a.updater != nil {
		if err := a.updater.SetProxy(value.HTTPProxy); err != nil {
			return err
		}
	}
	if a.rules != nil {
		if err := a.rules.SetProxy(value.HTTPProxy); err != nil {
			return err
		}
	}
	if a.provider != nil {
		if err := a.provider.SetProxy(value.HTTPProxy); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) Dashboard() (Dashboard, error) {
	if a.initErr != nil {
		return Dashboard{}, a.initErr
	}
	volumes, err := disks.List()
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{Volumes: volumes, RuleCount: len(a.rules.List()), Version: AppVersion}, nil
}

func (a *App) CheckForUpdates() (UpdateInfo, error) {
	if a.initErr != nil {
		return UpdateInfo{}, a.initErr
	}
	if a.updater == nil {
		return UpdateInfo{}, errors.New("updater is not ready")
	}
	value, err := a.updater.Check(a.appContext(), AppVersion)
	if err != nil {
		return UpdateInfo{}, err
	}
	return UpdateInfo{Available: value.Available, CurrentVersion: value.CurrentVersion, LatestVersion: value.LatestVersion, TagName: value.TagName, ReleaseName: value.ReleaseName, Notes: value.Notes, PublishedAt: value.PublishedAt, AssetName: value.AssetName, AssetSize: value.AssetSize, Digest: value.Digest, CheckedAt: value.CheckedAt}, nil
}

func (a *App) DownloadUpdate() (UpdateDownload, error) {
	if a.initErr != nil {
		return UpdateDownload{}, a.initErr
	}
	if a.updater == nil {
		return UpdateDownload{}, errors.New("updater is not ready")
	}
	value, err := a.updater.Download(a.appContext(), AppVersion)
	if err != nil {
		return UpdateDownload{}, err
	}
	return UpdateDownload{Version: value.Version, Size: value.Size, Verified: value.Verified}, nil
}

func (a *App) InstallUpdate() error {
	if a.initErr != nil {
		return a.initErr
	}
	if a.updater == nil {
		return errors.New("updater is not ready")
	}
	if err := a.updater.Install(); err != nil {
		return err
	}
	if a.ctx != nil {
		ctx := a.ctx
		time.AfterFunc(250*time.Millisecond, func() { runtime.Quit(ctx) })
	}
	return nil
}

func (a *App) appContext() context.Context {
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

func (a *App) Rules() ([]rules.Rule, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.rules.List(), nil
}

func (a *App) RuleStatistics() (rules.Statistics, error) {
	if a.initErr != nil {
		return rules.Statistics{}, a.initErr
	}
	return a.rules.Statistics(), nil
}

func (a *App) SyncRules() (rules.SyncResult, error) {
	if a.initErr != nil {
		return rules.SyncResult{}, a.initErr
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return a.rules.Sync(ctx)
}

func (a *App) SelectScanDirectory() (string, error) {
	if a.ctx == nil {
		return "", errors.New("application is not ready")
	}
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "选择要扫描的文件夹"})
}

func (a *App) StartScan(request scanner.Request) (scanner.Job, error) {
	if a.initErr != nil {
		return scanner.Job{}, a.initErr
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return a.scanner.Start(ctx, request)
}

func (a *App) ScanJob(id string) (scanner.Job, error) {
	if a.initErr != nil {
		return scanner.Job{}, a.initErr
	}
	return a.scanner.Job(id)
}

func (a *App) ScanItems(id string, offset, limit int) (scanner.ItemPage, error) {
	if a.initErr != nil {
		return scanner.ItemPage{}, a.initErr
	}
	return a.scanner.Items(id, offset, limit)
}

func (a *App) ScanFolders(id string) ([]*scanner.Folder, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.scanner.Folders(id)
}

func (a *App) CancelScan(id string) error {
	if a.initErr != nil {
		return a.initErr
	}
	return a.scanner.Cancel(id)
}

func (a *App) OpenSystemSettings(action string) error {
	return platform.OpenSettings(action)
}

func (a *App) BuildCleanPlan(request cleaner.BuildRequest) (cleaner.Plan, error) {
	if a.initErr != nil {
		return cleaner.Plan{}, a.initErr
	}
	return a.cleaner.BuildPlan(request)
}

func (a *App) ExecuteCleanPlan(request cleaner.ExecuteRequest) (cleaner.Result, error) {
	if a.initErr != nil {
		return cleaner.Result{}, a.initErr
	}
	return a.cleaner.Execute(request)
}

func (a *App) CleanHistory() ([]cleaner.Result, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.cleaner.History(), nil
}

func (a *App) AIProvider() (provider.Config, error) {
	if a.initErr != nil {
		return provider.Config{}, a.initErr
	}
	return a.provider.Get()
}

func (a *App) SaveAIProvider(input provider.ConfigInput) (provider.Config, error) {
	if a.initErr != nil {
		return provider.Config{}, a.initErr
	}
	return a.provider.Save(input)
}

func (a *App) TestAIProvider(input provider.ConfigInput) (provider.TestResult, error) {
	if a.initErr != nil {
		return provider.TestResult{}, a.initErr
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return a.provider.Test(ctx, input)
}

func (a *App) RunCleaningAgent(request cleaningagent.Request) (cleaningagent.Result, error) {
	if a.initErr != nil {
		return cleaningagent.Result{}, a.initErr
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if settings, err := a.ScanSettings(); err == nil {
		request.ExcludeRoots = settings.ExcludeRoots
	}
	return a.agent.Run(ctx, request)
}

// CleaningAgentPresets exposes the reviewed, domain-specific objectives used
// by the AI page. Presets never grant the model additional filesystem access.
func (a *App) CleaningAgentPresets() []cleaningagent.Preset {
	return cleaningagent.Presets
}
