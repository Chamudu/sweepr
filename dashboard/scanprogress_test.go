package dashboard

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"sweepr/scanner"
)

func TestScanProgressCollectsOnlyCompletedResults(t *testing.T) {
	events := make(chan ScanEvent)
	model := NewScanProgressModel([]string{"project"}, events)
	next, command := model.Update(ScanEvent{Name: "project", Progress: scanner.Progress{EntriesScanned: 12}})
	model = next.(ScanProgressModel)
	if model.completed != 0 || command == nil {
		t.Fatal("progress update completed scanner or stopped listening")
	}

	next, command = model.Update(ScanEvent{Name: "project", Done: true, Items: []scanner.Item{{Kind: "cache"}}})
	model = next.(ScanProgressModel)
	if model.completed != 1 || len(model.items) != 1 || command == nil {
		t.Fatalf("completion state = %#v", model)
	}
}

func TestScanProgressCancellationDiscardsCleanupPath(t *testing.T) {
	model := NewScanProgressModel([]string{"project"}, make(chan ScanEvent))
	next, command := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	model = next.(ScanProgressModel)
	if !model.cancelled || command == nil {
		t.Fatal("escape did not cancel and exit progress")
	}
}

func TestScanProgressViewExplainsReadOnlyWorkAndErrors(t *testing.T) {
	model := NewScanProgressModel([]string{"docker"}, make(chan ScanEvent))
	model.latest["docker"] = ScanEvent{Name: "docker", Done: true, Err: errors.New("daemon unavailable")}
	content := model.View().Content
	for _, want := range []string{"Scanning is read-only", "daemon unavailable", "Cancel scan and exit"} {
		if !strings.Contains(content, want) {
			t.Fatalf("progress view omitted %q: %q", want, content)
		}
	}
}
