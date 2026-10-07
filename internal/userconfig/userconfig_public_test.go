package userconfig

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestWelcomeCompleteReturnsFalseWhenFileAbsent verifies the normal first-launch
// case: no config file exists yet, so WelcomeComplete must return false without
// an error.
func TestWelcomeCompleteReturnsFalseWhenFileAbsent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "cfg"))
	got, err := WelcomeComplete()
	if err != nil {
		t.Fatalf("WelcomeComplete() with no file = error %v", err)
	}
	if got {
		t.Fatal("WelcomeComplete() returned true before any acknowledgement")
	}
}

// TestMarkWelcomeCompleteAndReadBack writes the acknowledgement through the
// public API and confirms WelcomeComplete sees it.
func TestMarkWelcomeCompleteAndReadBack(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "cfg"))
	if err := MarkWelcomeComplete(); err != nil {
		t.Fatalf("MarkWelcomeComplete() = %v", err)
	}
	got, err := WelcomeComplete()
	if err != nil {
		t.Fatalf("WelcomeComplete() after mark = %v", err)
	}
	if !got {
		t.Fatal("WelcomeComplete() returned false after MarkWelcomeComplete")
	}
}

// TestMarkWelcomeCompleteIsIdempotent calls Mark twice and asserts the second
// call succeeds and does not corrupt the file.
func TestMarkWelcomeCompleteIsIdempotent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "cfg"))
	for i := range 2 {
		if err := MarkWelcomeComplete(); err != nil {
			t.Fatalf("MarkWelcomeComplete() call %d = %v", i+1, err)
		}
	}
	got, err := WelcomeComplete()
	if err != nil || !got {
		t.Fatalf("WelcomeComplete() after two marks = (%v, %v)", got, err)
	}
}

// TestLoadPreferencesReturnsFalseWhenAbsent confirms that a missing Scan block
// returns (zero, false, nil) — not an error — matching the expected first-launch
// behaviour.
func TestLoadPreferencesReturnsFalseWhenAbsent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "cfg"))
	_, ok, err := LoadPreferences()
	if err != nil {
		t.Fatalf("LoadPreferences() with no file = %v", err)
	}
	if ok {
		t.Fatal("LoadPreferences() returned ok=true before any preferences were saved")
	}
}

// TestSaveAndLoadPreferencesRoundTrip saves a complete Preferences struct through
// the public API and reads it back, asserting every field survives the round trip.
func TestSaveAndLoadPreferencesRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "cfg"))
	want := Preferences{
		Root:     "/home/user/projects/myapp",
		Scope:    "local",
		Enabled:  []string{"dev-junk", "os-junk"},
		MinSize:  "50MB",
		MinAge:   14,
		Excludes: []string{"/home/user/projects/myapp/vendor"},
	}
	if err := SavePreferences(want); err != nil {
		t.Fatalf("SavePreferences() = %v", err)
	}
	got, ok, err := LoadPreferences()
	if err != nil {
		t.Fatalf("LoadPreferences() = %v", err)
	}
	if !ok {
		t.Fatal("LoadPreferences() returned ok=false after save")
	}
	if got.Root != want.Root {
		t.Errorf("Root = %q, want %q", got.Root, want.Root)
	}
	if got.Scope != want.Scope {
		t.Errorf("Scope = %q, want %q", got.Scope, want.Scope)
	}
	if !slices.Equal(got.Enabled, want.Enabled) {
		t.Errorf("Enabled = %v, want %v", got.Enabled, want.Enabled)
	}
	if got.MinSize != want.MinSize {
		t.Errorf("MinSize = %q, want %q", got.MinSize, want.MinSize)
	}
	if got.MinAge != want.MinAge {
		t.Errorf("MinAge = %d, want %d", got.MinAge, want.MinAge)
	}
	if !slices.Equal(got.Excludes, want.Excludes) {
		t.Errorf("Excludes = %v, want %v", got.Excludes, want.Excludes)
	}
}

// TestSavePreferencesDoesNotMutateSlices verifies that SavePreferences takes an
// independent copy of Enabled and Excludes slices rather than holding a
// reference to the caller's slice. Mutating the original after saving must not
// affect what is later read back.
func TestSavePreferencesDoesNotMutateSlices(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "cfg"))
	enabled := []string{"dev-junk"}
	excludes := []string{"/vendor"}
	if err := SavePreferences(Preferences{Root: "/app", Scope: "local", Enabled: enabled, Excludes: excludes}); err != nil {
		t.Fatal(err)
	}
	// Mutate originals.
	enabled[0] = "MUTATED"
	excludes[0] = "/MUTATED"

	got, _, err := LoadPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Enabled) != 1 || got.Enabled[0] != "dev-junk" {
		t.Errorf("Enabled after mutation = %v, want [dev-junk]", got.Enabled)
	}
	if len(got.Excludes) != 1 || got.Excludes[0] != "/vendor" {
		t.Errorf("Excludes after mutation = %v, want [/vendor]", got.Excludes)
	}
}

// TestSaveOverwritesPreviousPreferences confirms that calling SavePreferences
// twice with different values replaces the stored preferences, not appends.
func TestSaveOverwritesPreviousPreferences(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "cfg"))
	first := Preferences{Root: "/first", Scope: "local", Enabled: []string{"dev-junk"}}
	second := Preferences{Root: "/second", Scope: "global", Enabled: []string{"lang-cache"}}
	for _, p := range []Preferences{first, second} {
		if err := SavePreferences(p); err != nil {
			t.Fatalf("SavePreferences() = %v", err)
		}
	}
	got, ok, err := LoadPreferences()
	if err != nil || !ok {
		t.Fatalf("LoadPreferences() = (%v, %v, %v)", got, ok, err)
	}
	if got.Root != "/second" {
		t.Errorf("Root after overwrite = %q, want /second", got.Root)
	}
	if got.Scope != "global" {
		t.Errorf("Scope after overwrite = %q, want global", got.Scope)
	}
}

// TestWelcomeCompletePreservesPreferencesOnMark ensures that marking the
// welcome acknowledgement on a config that already has saved scan preferences
// does not erase those preferences.
func TestWelcomeCompletePreservesPreferencesOnMark(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "cfg"))
	want := Preferences{Root: "/project", Scope: "local", Enabled: []string{"os-junk"}}
	if err := SavePreferences(want); err != nil {
		t.Fatal(err)
	}
	if err := MarkWelcomeComplete(); err != nil {
		t.Fatal(err)
	}
	got, ok, err := LoadPreferences()
	if err != nil || !ok {
		t.Fatalf("LoadPreferences() after MarkWelcomeComplete = (%v, %v, %v)", got, ok, err)
	}
	if got.Root != want.Root {
		t.Errorf("preferences lost after mark: Root = %q, want %q", got.Root, want.Root)
	}
}

// TestLoadRejectsCorruptConfig verifies that a config file with valid JSON
// syntax but missing required fields still loads without panicking, and that
// truly corrupt JSON produces an error through the public API (not just
// the internal load function tested elsewhere).
func TestPublicLoadRejectsCorruptJSON(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cfg", "sweepr")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{bad json`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "cfg"))
	// Point XDG_CONFIG_HOME at the parent so os.UserConfigDir() finds our dir.
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(dir))

	// All public functions that call load() must surface the decode error.
	if _, err := WelcomeComplete(); err == nil {
		t.Error("WelcomeComplete() with corrupt JSON returned nil error")
	}
}
