package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLinuxSystemCandidatesUseXDGThumbnailCache(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "cache")
	thumbnails := filepath.Join(cache, "thumbnails")
	if err := os.MkdirAll(thumbnails, 0o700); err != nil {
		t.Fatal(err)
	}
	candidates := systemCandidates("linux", "/unused", func(name string) string {
		if name == "XDG_CACHE_HOME" {
			return cache
		}
		return ""
	}, "", time.Now())
	if len(candidates) != 1 || candidates[0].path != thumbnails || candidates[0].kind != "system-thumbnail-cache" {
		t.Fatalf("Linux candidates = %#v", candidates)
	}
}

func TestMacSystemCandidatesExcludeDevelopmentCaches(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "Library", "Caches")
	for _, name := range []string{"com.example.app", "pip", "CocoaPods"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	candidates := systemCandidates("darwin", home, func(string) string { return "" }, "", time.Now())
	if len(candidates) != 1 || filepath.Base(candidates[0].path) != "com.example.app" {
		t.Fatalf("macOS candidates = %#v", candidates)
	}
}

func TestWindowsSystemCandidatesOnlyIncludeOldTempEntries(t *testing.T) {
	root := t.TempDir()
	local, temp := filepath.Join(root, "local"), filepath.Join(root, "temp")
	explorer := filepath.Join(local, "Microsoft", "Windows", "Explorer")
	if err := os.MkdirAll(explorer, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(temp, 0o700); err != nil {
		t.Fatal(err)
	}
	thumb := filepath.Join(explorer, "thumbcache_256.db")
	old := filepath.Join(temp, "old.tmp")
	fresh := filepath.Join(temp, "fresh.tmp")
	for _, path := range []string{thumb, old, fresh} {
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	if err := os.Chtimes(old, now.Add(-8*24*time.Hour), now.Add(-8*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	candidates := systemCandidates("windows", root, func(name string) string {
		if name == "LOCALAPPDATA" {
			return local
		}
		return ""
	}, temp, now)
	paths := map[string]bool{}
	for _, candidate := range candidates {
		paths[candidate.path] = true
	}
	if !paths[thumb] || !paths[old] || paths[fresh] {
		t.Fatalf("Windows candidates = %#v", candidates)
	}
}

func TestSystemCandidatesNeverIncludeLogsRegistryTrashOrDownloads(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		for _, candidate := range systemCandidates(goos, t.TempDir(), func(string) string { return "" }, t.TempDir(), time.Now()) {
			base := filepath.Base(candidate.path)
			if base == "Logs" || base == "Downloads" || base == "Trash" {
				t.Fatalf("%s unsafe candidate: %#v", goos, candidate)
			}
		}
	}
}

func TestSystemTempRequiresNewestContentToBeOld(t *testing.T) {
	now := time.Now()
	if oldSystemTemp(now.Add(-time.Hour), now) {
		t.Fatal("fresh temporary content was considered old")
	}
	if !oldSystemTemp(now.Add(-8*24*time.Hour), now) {
		t.Fatal("eight-day-old temporary content was not eligible")
	}
}

func TestPathWithinRejectsSiblingPrefix(t *testing.T) {
	base := filepath.Join(t.TempDir(), "user")
	if !pathWithin(base, filepath.Join(base, "temp")) {
		t.Fatal("child path was rejected")
	}
	if pathWithin(base, base+"-other") {
		t.Fatal("sibling sharing a string prefix was accepted")
	}
}
