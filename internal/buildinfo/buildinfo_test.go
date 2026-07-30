package buildinfo

import (
	"strings"
	"testing"
)

func TestStringIncludesInjectedMetadata(t *testing.T) {
	oldVersion, oldCommit, oldDate := Version, Commit, Date
	t.Cleanup(func() {
		Version, Commit, Date = oldVersion, oldCommit, oldDate
	})

	Version = "v0.1.0"
	Commit = "abc1234"
	Date = "2026-07-30T12:00:00Z"
	got := String()
	for _, want := range []string{"sweepr v0.1.0", "abc1234", "2026-07-30T12:00:00Z"} {
		if !strings.Contains(got, want) {
			t.Errorf("version output %q does not contain %q", got, want)
		}
	}
}
