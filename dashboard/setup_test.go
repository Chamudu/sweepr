package dashboard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestSetupScopeDisablesIncompatibleScanners(t *testing.T) {
	model := NewSetupModel(DefaultScanSetup(t.TempDir()))
	model.cursor = 1
	next, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	model = next.(SetupModel)
	if model.config.Scope != ScopeGlobal {
		t.Fatalf("scope = %v; want global", model.config.Scope)
	}
	if model.config.Enabled["dev-junk"] || model.config.Enabled["os-junk"] {
		t.Fatal("global scope retained project scanner selections")
	}
	for _, name := range []string{"lang-cache", "docker"} {
		if !model.config.Enabled[name] {
			t.Fatalf("global scope did not enable %s as a useful default", name)
		}
	}
}

func TestSetupUnavailableScannerShowsReason(t *testing.T) {
	model := NewSetupModel(DefaultScanSetup(t.TempDir()))
	model.cursor = 6 // Docker is unavailable in local scope.
	next, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeySpace}))
	model = next.(SetupModel)
	if model.config.Enabled["docker"] {
		t.Fatal("local scope enabled Docker")
	}
	if !strings.Contains(model.notice, "Docker engine") {
		t.Fatalf("Docker notice = %q; want scope explanation", model.notice)
	}
}

func TestSetupValidationRequiresScannerAndDirectory(t *testing.T) {
	config := DefaultScanSetup(t.TempDir())
	clear(config.Enabled)
	if err := validateSetup(config); err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("empty scanner validation = %v", err)
	}
	config.Root = config.Root + "/missing"
	if err := validateSetup(config); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("missing directory validation = %v", err)
	}
}

func TestSetupViewShowsScopeAndDisabledScanners(t *testing.T) {
	content := NewSetupModel(DefaultScanSetup(t.TempDir())).View().Content
	for _, want := range []string{"Configure scan", "Selected folder only", "[-] Global development caches", "[-] System caches & temporary files", "[-] Docker images", "Start scan"} {
		if !strings.Contains(content, want) {
			t.Fatalf("setup did not contain %q: %q", want, content)
		}
	}
}
