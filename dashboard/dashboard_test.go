package dashboard

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"sweepr/scanner"
)

func press(model Model, key tea.Key) Model {
	next, _ := model.Update(tea.KeyPressMsg(key))
	return next.(Model)
}

func itemsModel(items []scanner.Item, mode Mode) Model {
	model := NewModel(items, mode)
	model.screen = screenItems
	return model
}

func TestUpdateNavigatesAndTogglesWithKeyboardEvents(t *testing.T) {
	model := itemsModel([]scanner.Item{
		{Kind: "first", SizeBytes: 10},
		{Kind: "second", SizeBytes: 20},
	}, ModeReadOnly)

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
	model := itemsModel([]scanner.Item{
		{Kind: "npm-cache", SizeBytes: 1024},
		{Kind: "go-build-cache", SizeBytes: 2048},
	}, ModeReadOnly)

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

func TestReviewRequiresSelectionAndSupportsBackAndConfirm(t *testing.T) {
	model := itemsModel([]scanner.Item{{
		Path:         "/tmp/cache",
		Kind:         "test-cache",
		SizeBytes:    4096,
		ResourceType: scanner.ResourceDirectory,
	}}, ModePermanent)

	model = press(model, tea.Key{Code: 'd', Text: "d"})
	if model.screen != screenItems {
		t.Fatal("d opened review without a selected item")
	}

	model = press(model, tea.Key{Code: tea.KeySpace})
	model = press(model, tea.Key{Code: 'd', Text: "d"})
	if model.screen != screenReview {
		t.Fatal("d did not open review after selecting an item")
	}
	if content := model.View().Content; !strings.Contains(content, "/tmp/cache") {
		t.Fatalf("review did not show the exact selected target: %q", content)
	}

	model = press(model, tea.Key{Code: tea.KeyEscape})
	if model.screen != screenItems {
		t.Fatal("escape did not return from review to the item list")
	}

	model = press(model, tea.Key{Code: 'd', Text: "d"})
	model = press(model, tea.Key{Code: tea.KeyEnter})
	if !model.confirmed {
		t.Fatal("enter on the review screen did not record confirmation intent")
	}
	selected := model.selectedItems()
	if len(selected) != 1 || selected[0].Path != "/tmp/cache" {
		t.Fatalf("confirmed selection was %#v; want /tmp/cache", selected)
	}
}

func TestViewUsesDisplayNameForNonFilesystemResources(t *testing.T) {
	model := itemsModel([]scanner.Item{{
		Path:        "sha256:abc123",
		DisplayName: "dangling image abc123",
		Kind:        "docker-image",
	}}, ModeReadOnly)

	content := model.View().Content
	if !strings.Contains(content, "dangling image abc123") {
		t.Fatalf("dashboard view did not contain friendly display name: %q", content)
	}
}

func TestReviewExplainsWhetherDeletionIsEnabled(t *testing.T) {
	item := scanner.Item{Path: "/tmp/cache", Kind: "cache", ResourceType: scanner.ResourceDirectory}

	readOnly := itemsModel([]scanner.Item{item}, ModeReadOnly)
	readOnly.toggleCurrent()
	readOnly.screen = screenReview
	if content := readOnly.View().Content; !strings.Contains(content, "Read-only preview") {
		t.Fatalf("read-only review omitted its safety status: %q", content)
	}

	destructive := itemsModel([]scanner.Item{item}, ModePermanent)
	destructive.toggleCurrent()
	destructive.screen = screenReview
	if content := destructive.View().Content; !strings.Contains(content, "Delete permanently") {
		t.Fatalf("deletion review omitted its warning: %q", content)
	}
}

func TestWindowSizeKeepsCursorInsideVisibleViewport(t *testing.T) {
	items := make([]scanner.Item, 20)
	for index := range items {
		items[index] = scanner.Item{Kind: fmt.Sprintf("item-%02d", index)}
	}
	model := itemsModel(items, ModeReadOnly)
	next, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 19})
	model = next.(Model)

	for range 10 {
		model = press(model, tea.Key{Code: tea.KeyDown})
	}
	if model.cursor != 10 {
		t.Fatalf("cursor = %d; want 10", model.cursor)
	}
	if model.offset != 8 {
		t.Fatalf("viewport offset = %d; want 8 for three visible rows", model.offset)
	}
	content := model.View().Content
	if !strings.Contains(content, "item-10") || strings.Contains(content, "item-00") {
		t.Fatalf("viewport did not render the expected visible window: %q", content)
	}
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) > 19 {
		t.Fatalf("80x19 view rendered %d lines; want at most 19", len(lines))
	}
	for index, line := range lines {
		if width := lipgloss.Width(line); width > 80 {
			t.Fatalf("line %d occupies %d cells; want at most 80", index+1, width)
		}
	}
}

func TestSelectAllSkipsItemsUnsupportedByMode(t *testing.T) {
	model := itemsModel([]scanner.Item{
		{Path: "/tmp/cache", ResourceType: scanner.ResourceDirectory},
		{Path: "sha256:abc", ResourceType: scanner.ResourceDockerImage},
	}, ModeTrash)

	model = press(model, tea.Key{Code: 'a', Text: "a"})
	if len(model.selected) != 1 {
		t.Fatalf("trash select-all selected %d items; want only the filesystem item", len(model.selected))
	}
	if _, selected := model.selected[0]; !selected {
		t.Fatal("trash select-all did not select the supported directory")
	}

	model = press(model, tea.Key{Code: 'c', Text: "c"})
	if len(model.selected) != 0 {
		t.Fatal("clear-selection key left selected items")
	}
}

func TestTruncateTextUsesTerminalCellWidth(t *testing.T) {
	got := truncateText("cache/very/long/path", 10)
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("truncated value %q has no ellipsis", got)
	}
	if width := lipgloss.Width(got); width > 10 {
		t.Fatalf("truncated value occupies %d cells; want at most 10", width)
	}
}

func TestModeChooserAndTrashRestrictions(t *testing.T) {
	items := []scanner.Item{
		{Path: "/tmp/cache", ResourceType: scanner.ResourceDirectory},
		{Path: "sha256:abc", ResourceType: scanner.ResourceDockerImage},
	}
	model := NewModel(items, ModeTrash)
	if model.screen != screenMode || model.mode != ModeTrash {
		t.Fatalf("initial mode state = (%v, %v); want mode chooser with trash highlighted", model.screen, model.mode)
	}

	model = press(model, tea.Key{Code: tea.KeyEnter})
	if model.screen != screenItems {
		t.Fatal("enter did not continue from mode chooser to items")
	}

	model.cursor = 1
	model = press(model, tea.Key{Code: tea.KeySpace})
	if len(model.selected) != 0 {
		t.Fatal("trash mode allowed a Docker image to be selected")
	}
	if content := model.View().Content; !strings.Contains(content, "[-]") {
		t.Fatalf("trash-incompatible Docker row was not marked unavailable: %q", content)
	}
}

func TestInstructionManualOpensAndReturnsToPreviousScreen(t *testing.T) {
	model := itemsModel([]scanner.Item{{Kind: "cache"}}, ModeReadOnly)

	model = press(model, tea.Key{Code: 'i', Text: "i"})
	if model.screen != screenHelp || model.returnScreen != screenItems {
		t.Fatalf("manual state = (%v, return %v); want help returning to items", model.screen, model.returnScreen)
	}
	content := model.View().Content
	for _, want := range []string{"Instruction manual", "START SAFELY", "Permanent delete", "by Chamudu"} {
		if !strings.Contains(content, want) {
			t.Fatalf("manual did not contain %q: %q", want, content)
		}
	}

	model = press(model, tea.Key{Code: tea.KeyEscape})
	if model.screen != screenItems {
		t.Fatalf("escape returned to screen %v; want items", model.screen)
	}
}

func TestInstructionManualScrollIsBounded(t *testing.T) {
	model := NewModel(nil, ModeReadOnly)
	next, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 15})
	model = next.(Model)
	model = press(model, tea.Key{Code: 'i', Text: "i"})
	model = press(model, tea.Key{Code: tea.KeyEnd})

	want := max(0, len(manualLines())-model.visibleHelpRows())
	if model.helpOffset != want {
		t.Fatalf("end set manual offset to %d; want %d", model.helpOffset, want)
	}
	model = press(model, tea.Key{Code: tea.KeyDown})
	if model.helpOffset != want {
		t.Fatalf("down moved manual beyond final line to %d; want %d", model.helpOffset, want)
	}
	model = press(model, tea.Key{Code: tea.KeyHome})
	if model.helpOffset != 0 {
		t.Fatalf("home set manual offset to %d; want 0", model.helpOffset)
	}
}
