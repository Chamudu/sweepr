package userconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadWelcomeAcknowledgement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	want := config{Version: configVersion, WelcomeComplete: true}
	if err := save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("loaded config = %#v; want %#v", got, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("config permissions = %o; want no group/other permissions", info.Mode().Perm())
	}
}

func TestLoadRejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := load(path); err == nil {
		t.Fatal("invalid configuration JSON returned no error")
	}
}

func TestConfigRoundTripIncludesOnlyScanPreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	want := config{Version: configVersion, WelcomeComplete: true, Scan: &Preferences{
		Root: "/projects/app", Scope: "local", Enabled: []string{"dev-junk"},
		MinSize: "100MB", MinAge: 30, Excludes: []string{"/projects/app/vendor"},
	}}
	if err := save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Scan == nil || got.Scan.Root != want.Scan.Root || got.Scan.MinAge != 30 {
		t.Fatalf("loaded preferences = %#v; want %#v", got.Scan, want.Scan)
	}
}
