package scanner

import (
	"path/filepath"
	"testing"
)

func TestWindowsCacheLocationsUseLocalAppData(t *testing.T) {
	home := filepath.Join("C:", "Users", "Tada")
	localAppData := filepath.Join(home, "AppData", "Local")
	getenv := func(key string) string {
		if key == "LOCALAPPDATA" {
			return localAppData
		}
		return ""
	}

	locations := cacheLocations("windows", home, getenv)
	want := map[string]string{
		filepath.Join(localAppData, "npm-cache"):     "npm-cache",
		filepath.Join(localAppData, "pip", "Cache"):  "pip-cache",
		filepath.Join(localAppData, "go-build"):      "go-build-cache",
		filepath.Join(localAppData, "Yarn", "Cache"): "yarn-cache",
		filepath.Join(localAppData, "pnpm", "store"): "pnpm-cache",
		filepath.Join(home, ".gradle", "caches"):     "gradle-cache",
	}

	for path, kind := range want {
		if !hasCacheLocation(locations, path, kind) {
			t.Errorf("Windows locations missing %s at %s", kind, path)
		}
	}
	if hasCacheKind(locations, "xcode-derived-data") {
		t.Error("Windows locations unexpectedly contain an Xcode cache")
	}
}

func TestWindowsCacheLocationsOmitLocalAppDataPathsWhenUnset(t *testing.T) {
	locations := cacheLocations("windows", `C:\Users\Tada`, func(string) string { return "" })
	for _, kind := range []string{"npm-cache", "pip-cache", "go-build-cache", "yarn-cache", "pnpm-cache"} {
		if hasCacheKind(locations, kind) {
			t.Errorf("Windows locations contain %s without LOCALAPPDATA", kind)
		}
	}
}

func TestPlatformSpecificCachesDoNotLeakBetweenSystems(t *testing.T) {
	linux := cacheLocations("linux", "/home/tada", func(string) string { return "" })
	macOS := cacheLocations("darwin", "/Users/tada", func(string) string { return "" })
	if !hasCacheLocation(linux, filepath.Join("/home/tada", ".npm"), "npm-cache") {
		t.Error("Linux locations missing ~/.npm")
	}
	if hasCacheKind(linux, "xcode-derived-data") {
		t.Error("Linux locations unexpectedly contain Xcode DerivedData")
	}
	if !hasCacheKind(macOS, "xcode-derived-data") {
		t.Error("macOS locations missing Xcode DerivedData")
	}
}

func hasCacheLocation(locations []cacheLocation, path, kind string) bool {
	for _, location := range locations {
		if location.path == path && location.kind == kind {
			return true
		}
	}
	return false
}

func hasCacheKind(locations []cacheLocation, kind string) bool {
	for _, location := range locations {
		if location.kind == kind {
			return true
		}
	}
	return false
}
