package scanner

import (
	"os"
	"path/filepath"
	"runtime"
)

// cacheRelPaths maps paths relative to the user's home directory ($HOME) to a
// short "kind" label. These are fixed, well-known locations where package
// managers store their global download caches.
//
// Unlike devJunkNames, these are NOT discovered by walking a project root —
// we check them directly. If a path does not exist on the current machine
// (e.g., Xcode cache on a Linux box), it is silently skipped. That is
// expected behavior, not an error.
var homeCacheRelPaths = map[string]string{
	".cargo/registry/cache":     "cargo-cache",
	"go/pkg/mod/cache/download": "go-mod-cache",
	".gradle/caches":            "gradle-cache",

	// Mobile Development
	".android/avd": "android-emulator-snapshots", // Deletes stored states of virtual devices

	// Compiler / Language Toolchains
	// Additional package tools
	".composer/cache": "php-composer-cache", // PHP package dependency cache
	".bower":          "bower-cache",        // Legacy frontend package manager cache
	// Global IDE & CLI Tool caches
	".azure/cliextensions":        "azure-cli-extensions",       // Azure CLI extensions cache
	".vscode/extensions":          "vscode-extensions",          // VS Code downloaded extensions
	".antigravity/extensions":     "antigravity-extensions",     // Antigravity downloaded extensions
	".antigravity-ide/extensions": "antigravity-ide-extensions", // Antigravity IDE extensions
}

var unixCacheRelPaths = map[string]string{
	".npm":              "npm-cache",
	".cache/pip":        "pip-cache",
	".cache/go-build":   "go-build-cache",
	".cache/yarn":       "yarn-cache",
	".local/share/pnpm": "pnpm-cache",
	".cache/clangd":     "clangd-index-cache",
	".cache/deno":       "deno-cache",
	".cache/zig":        "zig-cache",
	".cache/supabase":   "supabase-local-dev",
	".cache/hardhat":    "hardhat-evm-cache",
}

var macOSCacheRelPaths = map[string]string{
	"Library/Developer/Xcode/DerivedData": "xcode-derived-data",
	"Library/Developer/Xcode/Archives":    "xcode-archives",
	"Library/Caches/CocoaPods":            "cocoapods-cache",
	"Library/Caches/pip":                  "pip-cache",
}

type cacheLocation struct {
	path string
	kind string
}

// cacheLocations translates platform conventions into absolute paths. Passing
// getenv as a function keeps Windows path behavior testable on other systems.
func cacheLocations(goos, home string, getenv func(string) string) []cacheLocation {
	locations := make([]cacheLocation, 0, len(homeCacheRelPaths)+len(unixCacheRelPaths))
	appendHomePaths := func(paths map[string]string) {
		for relPath, kind := range paths {
			locations = append(locations, cacheLocation{
				path: filepath.Join(home, filepath.FromSlash(relPath)),
				kind: kind,
			})
		}
	}
	appendHomePaths(homeCacheRelPaths)

	if goos != "windows" {
		appendHomePaths(unixCacheRelPaths)
		if goos == "darwin" {
			appendHomePaths(macOSCacheRelPaths)
		}
		return locations
	}

	if localAppData := getenv("LOCALAPPDATA"); localAppData != "" {
		windowsPaths := map[string]string{
			"npm-cache":  "npm-cache",
			"pip/Cache":  "pip-cache",
			"go-build":   "go-build-cache",
			"Yarn/Cache": "yarn-cache",
			"pnpm/store": "pnpm-cache",
		}
		for relPath, kind := range windowsPaths {
			locations = append(locations, cacheLocation{
				path: filepath.Join(localAppData, filepath.FromSlash(relPath)),
				kind: kind,
			})
		}
	}
	return locations
}

// LangCacheScanner reports the disk usage of global package-manager caches
// stored under the user's home directory. Unlike DevJunkScanner and
// OSJunkScanner, it does not walk a project root — the 'root' parameter
// passed to Scan is intentionally ignored.
type LangCacheScanner struct{}

// Name satisfies the Scanner interface.
func (s *LangCacheScanner) Name() string {
	return "lang-cache"
}

// Scan checks each path in cacheRelPaths under the user's home directory.
// Paths that do not exist or are not directories are silently skipped —
// this is normal on machines that do not have the corresponding tool installed.
//
// We use os.Stat (follows symlinks) rather than os.Lstat here because some
// package managers (e.g., pnpm) store the real cache elsewhere and create a
// symlink at the well-known path. Following the symlink gives us the true size.
func (s *LangCacheScanner) Scan(root string, options ScanOptions) ([]Item, error) {
	// os.UserHomeDir reads $HOME on Linux/macOS and %USERPROFILE% on Windows.
	// Hardcoding a path like "/home/chamu" would break on other machines and OSes.
	home, err := os.UserHomeDir()
	if err != nil {
		// Home dir is required for this scanner — return the error rather than
		// silently reporting nothing.
		return nil, err
	}

	var items []Item
	var entriesScanned int64
	var bytesFound int64

	for _, location := range cacheLocations(runtime.GOOS, home, os.Getenv) {
		absPath := location.path

		// os.Stat follows symlinks. An error here almost always means the path
		// does not exist on this machine — use continue (not return) to check
		// the remaining paths in the map.
		info, err := os.Stat(absPath)
		if err != nil {
			continue
		}

		// Guard against a non-directory at the expected location (e.g., a file
		// named ".npm" in the home directory). dirStats expects a directory.
		if !info.IsDir() {
			continue
		}

		baseEntries := entriesScanned
		baseBytes := bytesFound
		size, modTime, measuredEntries, _ := dirStatsWithProgress(
			absPath,
			func(currentPath string, localEntries, localBytes int64) {
				options.ReportProgress(Progress{
					Path:           currentPath,
					EntriesScanned: baseEntries + localEntries,
					ItemsFound:     len(items),
					BytesFound:     baseBytes + localBytes,
				})
			},
		)
		entriesScanned += measuredEntries
		bytesFound += size
		items = append(items, Item{
			Path:         absPath,
			Kind:         location.kind,
			SizeBytes:    size,
			LastMod:      modTime,
			ResourceType: ResourceDirectory,
		})
		options.ReportProgress(Progress{
			Path:           absPath,
			EntriesScanned: entriesScanned,
			ItemsFound:     len(items),
			BytesFound:     bytesFound,
		})
	}

	options.ReportProgress(Progress{
		Path:           home,
		EntriesScanned: entriesScanned,
		ItemsFound:     len(items),
		BytesFound:     bytesFound,
	})

	return items, nil
}
