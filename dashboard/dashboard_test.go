package dashboard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"sweepr/scanner"
)

func press(model Model, key tea.Key) Model {
	next, _ := model.Update(tea.KeyPressMsg(key))
	return next.(Model)
}

func TestUpdateNavigatesAndTogglesWithKeyboardEvents(t *testing.T) {
	model := NewModel([]scanner.Item{
		{Kind: "first", SizeBytes: 10},
		{Kind: "second", SizeBytes: 20},
	})

	model = press(model, tea.Key{Code: tea.KeyDown})
	if model.cursor != 1 {
		t.Fatalf("down key moved cursor to %d; want 1", model.cursor)
	}

	model = press(model, tea.Key{Code: tea.KeySpace})
	if _, selected := model.selected[1]; !selected {
		t.Fatal("space key did not select the current item")
	}

	// Navigation is bounded: pressing down on the final item stays there.
	model = press(model, tea.Key{Code: tea.KeyDown})
	if model.cursor != 1 {
		t.Fatalf("down at final item moved cursor to %d; want 1", model.cursor)
	}
}

func TestToggleCurrentTracksSelectionAndBytes(t *testing.T) {
	model := NewModel([]scanner.Item{
		{Kind: "npm-cache", SizeBytes: 1024},
		{Kind: "go-build-cache", SizeBytes: 2048},
	})

	model.toggleCurrent()
	if len(model.selected) != 1 || model.selectedBytes() != 1024 {
		t.Fatalf("first toggle selected %d items and %d bytes; want 1 item and 1024 bytes",
			len(model.selected), model.selectedBytes())
	}

	model.toggleCurrent()
	if len(model.selected) != 0 || model.selectedBytes() != 0 {
		t.Fatalf("second toggle selected %d items and %d bytes; want empty selection",
			len(model.selected), model.selectedBytes())
	}
}

func TestViewUsesDisplayNameForNonFilesystemResources(t *testing.T) {
	model := NewModel([]scanner.Item{{
		Path:        "sha256:abc123",
		DisplayName: "dangling image abc123",
		Kind:        "docker-image",
	}})

	content := model.View().Content
	if !strings.Contains(content, "dangling image abc123") {
		t.Fatalf("dashboard view did not contain friendly display name: %q", content)
	}
}
