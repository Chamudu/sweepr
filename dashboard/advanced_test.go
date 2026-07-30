package dashboard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestParseSizeValue(t *testing.T) {
	for input, want := range map[string]int64{"100MB": 100 * 1024 * 1024, "1.5GB": 1536 * 1024 * 1024, "500KB": 500 * 1024} {
		got, err := parseSizeValue(input)
		if err != nil || got != want {
			t.Errorf("parseSizeValue(%q) = (%d, %v); want %d", input, got, err, want)
		}
	}
	if _, err := parseSizeValue("ten elephants"); err == nil {
		t.Fatal("invalid size returned no error")
	}
}

func TestAdvancedEditingValidatesFields(t *testing.T) {
	model := NewAdvancedModel(DefaultScanSetup(t.TempDir()))
	model.editing, model.input = true, "bad"
	next, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = next.(AdvancedModel)
	if !model.editing || !strings.Contains(model.notice, "size") {
		t.Fatalf("invalid size state = editing %v notice %q", model.editing, model.notice)
	}

	model.input = "250MB"
	next, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = next.(AdvancedModel)
	if model.editing || model.config.MinSize != "250MB" {
		t.Fatalf("valid size was not applied: %#v", model)
	}
}

func TestAdvancedRemovesFocusedExclusion(t *testing.T) {
	config := DefaultScanSetup(t.TempDir())
	config.Excludes = []string{"one", "two"}
	model := NewAdvancedModel(config)
	model.cursor = 4
	next, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: 'x', Text: "x"}))
	model = next.(AdvancedModel)
	if len(model.config.Excludes) != 1 || model.config.Excludes[0] != "one" {
		t.Fatalf("exclusions = %#v; want one", model.config.Excludes)
	}
}

func TestCloneSetupDoesNotShareMutableState(t *testing.T) {
	original := DefaultScanSetup(t.TempDir())
	original.Excludes = []string{"first"}
	clone := cloneSetup(original)
	clone.Enabled["dev-junk"] = false
	clone.Excludes[0] = "changed"
	if !original.Enabled["dev-junk"] || original.Excludes[0] != "first" {
		t.Fatal("clone mutated original setup")
	}
}
