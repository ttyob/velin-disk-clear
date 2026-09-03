package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cleaningagent "ai-clear/internal/ai/agent"
	"ai-clear/internal/ai/provider"
	"ai-clear/internal/cleaner"
	"ai-clear/internal/disks"
	"ai-clear/internal/platform"
	"ai-clear/internal/rules"
	"ai-clear/internal/scanner"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Dashboard struct {
	Volumes   []disks.Volume `json:"volumes"`
	RuleCount int            `json:"rule_count"`
	Version   string         `json:"version"`
}

type App struct {
	ctx      context.Context
	rules    *rules.Service
	scanner  *scanner.Service
	cleaner  *cleaner.Service
	provider *provider.Service
	agent    *cleaningagent.Service
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
		app.cleaner, app.initErr = cleaner.New(app.scanner, dataDir)
		if app.initErr == nil {
			app.provider, app.initErr = provider.New(dataDir)
			if app.initErr == nil {
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
}

func (a *App) Dashboard() (Dashboard, error) {
	if a.initErr != nil {
		return Dashboard{}, a.initErr
	}
	volumes, err := disks.List()
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{Volumes: volumes, RuleCount: len(a.rules.List()), Version: "0.1.0-dev"}, nil
}

func (a *App) Rules() ([]rules.Rule, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.rules.List(), nil
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
	return a.agent.Run(ctx, request)
}

// CleaningAgentPresets exposes the reviewed, domain-specific objectives used
// by the AI page. Presets never grant the model additional filesystem access.
func (a *App) CleaningAgentPresets() []cleaningagent.Preset {
	return cleaningagent.Presets
}
