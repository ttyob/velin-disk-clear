package rules

import "testing"

func TestLoadBuiltin(t *testing.T) {
	service, err := LoadBuiltin()
	if err != nil {
		t.Fatalf("LoadBuiltin() error = %v", err)
	}
	loaded := service.List()
	if len(loaded) < 145 {
		t.Fatalf("expected at least 145 builtin rules, got %d", len(loaded))
	}

	pagefile, ok := service.Get("windows.pagefile_analysis")
	if !ok {
		t.Fatal("pagefile analysis rule is missing")
	}
	if pagefile.DefaultSelected {
		t.Fatal("pagefile analysis must never be selected by default")
	}
	if pagefile.Help.Details == "" || pagefile.Help.SpecialWarning == "" {
		t.Fatal("pagefile analysis must include detailed safety help")
	}

	recycleBin, ok := service.Get("windows.recycle_bin")
	if !ok {
		t.Fatal("recycle bin rule is missing")
	}
	if recycleBin.Action.Type != "empty_recycle_bin" {
		t.Fatalf("recycle bin must use the Windows Shell action, got %q", recycleBin.Action.Type)
	}
	if recycleBin.DefaultSelected {
		t.Fatal("recycle bin must never be selected by default")
	}
	if recycleBin.Risk != RiskMedium {
		t.Fatalf("recycle bin must remain medium risk, got %q", recycleBin.Risk)
	}

	downloads, ok := service.Get("windows.downloads_installers")
	if !ok {
		t.Fatal("downloads installer rule is missing")
	}
	if downloads.DefaultSelected || downloads.Recommendation != RecommendationAnalyzeOnly {
		t.Fatal("downloads installer candidates must be analysis-only and unselected by default")
	}
	if downloads.Risk != RiskHigh || downloads.Help.Details == "" || downloads.Help.SpecialWarning == "" {
		t.Fatal("downloads installer candidates must carry high-risk handling guidance")
	}

	updateDownloads, ok := service.Get("windows.update_download_analysis")
	if !ok {
		t.Fatal("Windows Update download analysis rule is missing")
	}
	if updateDownloads.Action.Type != "analyze" || updateDownloads.DefaultSelected {
		t.Fatal("Windows Update downloads must remain analysis-only and unselected")
	}
	if updateDownloads.Risk != RiskHigh || updateDownloads.Recommendation != RecommendationAnalyzeOnly {
		t.Fatal("Windows Update downloads must retain high-risk analysis guidance")
	}

	prefetch, ok := service.Get("windows.prefetch_analysis")
	if !ok {
		t.Fatal("Windows Prefetch analysis rule is missing")
	}
	if prefetch.Action.Type != "analyze" || prefetch.DefaultSelected {
		t.Fatal("Windows Prefetch must never enter cleanup candidates")
	}
	if prefetch.Risk != RiskHigh || prefetch.Recommendation != RecommendationNotRecommended {
		t.Fatal("Windows Prefetch must remain high-risk and not recommended")
	}

	stats := service.Statistics()
	if stats.Total != 145 || stats.System != 39 || stats.ThirdParty != 96 || stats.General != 10 {
		t.Fatalf("unexpected rule statistics: %#v", stats)
	}
	if stats.AnalysisOnly != 48 || stats.Executable != 97 {
		t.Fatalf("unexpected action statistics: %#v", stats)
	}

	for _, id := range []string{
		"app.360_security_update_analysis",
		"app.360_security_quarantine_analysis",
		"app.tencent_pc_manager_update_analysis",
		"app.tencent_pc_manager_quarantine_analysis",
		"app.ruanmei_cube_analysis",
	} {
		rule, ok := service.Get(id)
		if !ok {
			t.Fatalf("security-tool analysis rule %q is missing", id)
		}
		if rule.Action.Type != "analyze" || rule.DefaultSelected {
			t.Fatalf("security-tool rule %q must be analysis-only and unselected, got action=%q selected=%v", id, rule.Action.Type, rule.DefaultSelected)
		}
		if rule.Risk != RiskHigh || rule.Help.Details == "" || rule.Help.SpecialWarning == "" {
			t.Fatalf("security-tool rule %q must carry high-risk handling guidance", id)
		}
	}

	ruanmeiCache, ok := service.Get("app.ruanmei_cube_cache")
	if !ok {
		t.Fatal("softmedia cache rule is missing")
	}
	if ruanmeiCache.Action.Type != "permanent_delete" || ruanmeiCache.DefaultSelected {
		t.Fatalf("softmedia cache must be optional permanent deletion, got action=%q selected=%v", ruanmeiCache.Action.Type, ruanmeiCache.DefaultSelected)
	}

	for _, id := range []string{"app.wecom_cache_analysis", "dev.go_build_cache", "dev.go_module_cache", "dev.npm_cache", "dev.node_gyp_cache", "generic.large_directories"} {
		if _, ok := service.Get(id); !ok {
			t.Fatalf("expected developer/workplace rule %q", id)
		}
	}
}

func TestValidateRejectsUnsafeDefault(t *testing.T) {
	rule := validRule()
	rule.Risk = RiskHigh
	rule.DefaultSelected = true
	rule.Help.Details = "details"
	rule.Help.SpecialWarning = "warning"
	if err := Validate(rule); err == nil {
		t.Fatal("expected high-risk default selection to be rejected")
	}
}

func TestValidateRejectsCommands(t *testing.T) {
	rule := validRule()
	rule.Action.Type = "command"
	if err := Validate(rule); err == nil {
		t.Fatal("expected arbitrary command action to be rejected")
	}
}

func TestValidateRejectsUnknownAction(t *testing.T) {
	rule := validRule()
	rule.Action.Type = "plugin_delete"
	if err := Validate(rule); err == nil {
		t.Fatal("expected unknown action type to be rejected")
	}
}

func TestValidateRejectsSelectedAnalysisRule(t *testing.T) {
	rule := validRule()
	rule.Action.Type = "analyze"
	if err := Validate(rule); err == nil {
		t.Fatal("expected a selected analysis-only rule to be rejected")
	}
}

func validRule() Rule {
	return Rule{
		ID: "test.safe_rule", Version: 1, Name: "Test", Description: "Test rule",
		Purpose: "Testing", CleanEffect: "None", Recommendation: RecommendationRecommended,
		RecommendationReason: "Safe", Category: "test", Platform: "all", Enabled: true,
		RuleType: RuleTypeGeneral,
		Risk:     RiskLow, DefaultSelected: true, Scope: "selected_root", SizeMode: "allocated",
		RecoveryType: "none", LastVerifiedAt: "2026-09-03", Source: "test",
		Modes: []ScanMode{ScanModeQuick}, Help: Help{Summary: "Safe test rule"},
		Scan:   ScanSpec{Roots: []string{"$SCAN_ROOT"}, StayOnVolume: true},
		Action: ActionSpec{Type: "permanent_delete"},
		Safety: SafetySpec{AllowedRoots: []string{"$SCAN_ROOT"}, RevalidateBeforeClean: true},
	}
}
