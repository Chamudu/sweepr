package main

import (
	"testing"
	"time"

	"sweepr/dashboard"
	"sweepr/scanner"
)

// TestRunScanJobsStartsJobsConcurrently uses synchronization instead of timing
// comparisons. Both jobs announce that they started, then wait at the same
// gate. A sequential implementation could never get both jobs to that gate.
func TestRunScanJobsStartsJobsConcurrently(t *testing.T) {
	started := make(chan string, 2)
	release := make(chan struct{})

	newBlockingJob := func(name string) scanJob {
		return scanJob{
			name: name,
			run: func(scanner.ProgressFunc) ([]scanner.Item, error) {
				started <- name
				<-release
				return []scanner.Item{{Kind: name}}, nil
			},
		}
	}

	done := make(chan []scanner.Item, 1)
	go func() {
		done <- runScanJobs(
			[]scanJob{newBlockingJob("first"), newBlockingJob("second")},
			false,
			true,
		)
	}()

	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			close(release)
			t.Fatal("both scan jobs did not start concurrently")
		}
	}

	close(release)
	items := <-done
	if len(items) != 2 {
		t.Fatalf("runScanJobs returned %d items; want 2", len(items))
	}
}

func TestDeleteConfirmedSelectionOnlyForwardsConfirmedItems(t *testing.T) {
	selected := []scanner.Item{{Path: "/tmp/selected", Kind: "test"}}
	var received []scanner.Item
	spyDelete := func(items []scanner.Item) {
		received = append(received, items...)
	}

	deleted := deleteConfirmedSelection(dashboard.Result{Items: selected}, spyDelete)
	if deleted || len(received) != 0 {
		t.Fatal("unconfirmed dashboard result reached deletion")
	}

	deleted = deleteConfirmedSelection(dashboard.Result{
		Items:     selected,
		Confirmed: true,
	}, spyDelete)
	if !deleted || len(received) != 1 || received[0].Path != "/tmp/selected" {
		t.Fatalf("confirmed deletion received %#v; want only /tmp/selected", received)
	}
}

func TestSystemCacheRequiresExplicitCLISelection(t *testing.T) {
	if hasScanner(filterScanner(scanner.All(), "", ""), "system-cache") {
		t.Fatal("default CLI scanner set unexpectedly includes system cleanup")
	}
	explicit := filterScanner(scanner.All(), "system-cache", "")
	if len(explicit) != 1 || explicit[0].Name() != "system-cache" {
		t.Fatalf("explicit system-cache selection returned %#v", explicit)
	}
}
