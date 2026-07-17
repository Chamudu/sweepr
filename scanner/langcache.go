package scanner

import (
	"os"
	"path/filepath"
)

// cacheRelPaths maps paths relative to the user's home directory ($HOME) to a
// short "kind" label. These are fixed, well-known locations where package
// managers store their global download caches.
//
// Unlike devJunkNames, these are NOT discovered by walking a project root —
// we check them directly. If a path does not exist on the current machine
// (e.g., Xcode cache on a Linux box), it is silently skipped. That is
// expected behavior, not an error.
//
// Platform note: these paths are correct for Linux and macOS. Windows uses
// different conventions (%AppData%, %LocalAppData%), which will be handled
// in a future release via Go build constraints (//go:build windows).
var cacheRelPaths = map[string]string{
	".npm":                                "npm-cache",
	".cache/pip":                          "pip-cache",
	".cargo/registry/cache":               "cargo-cache",
	"go/pkg/mod/cache/download":           "go-mod-cache",
	".cache/go-build":                     "go-build-cache",
	".cache/yarn":                         "yarn-cache",
	".local/share/pnpm":                   "pnpm-cache",
	"Library/Developer/Xcode/DerivedData": "xcode-derived-data", // macOS only
	".gradle/caches":                      "gradle-cache",

	// Mobile Development
	".android/avd":                     "android-emulator-snapshots", // Deletes stored states of virtual devices
	"Library/Developer/Xcode/Archives": "xcode-archives",             // macOS only: Past production build history
	"Library/Caches/CocoaPods":         "cocoapods-cache",            // macOS only: iOS Swift/Obj-C package cache

	// Compiler / Language Toolchains
	".cache/clangd":   "clangd-index-cache", // C/C++ language server indexes
	".cache/deno":     "deno-cache",         // Deno runtime and package storage
	".cache/zig":      "zig-cache",          // Zig compiler global artifact store
	".cache/supabase": "supabase-local-dev", // Local database dev caches
	".cache/hardhat":  "hardhat-evm-cache",  // Ethereum/Web3 smart contract dev cache

	// Additional package tools
	".composer/cache": "php-composer-cache", // PHP package dependency cache
	".bower":          "bower-cache",        // Legacy frontend package manager cache
	// Global IDE & CLI Tool caches
	".azure/cliextensions":        "azure-cli-extensions",       // Azure CLI extensions cache
	".vscode/extensions":          "vscode-extensions",          // VS Code downloaded extensions
	".antigravity/extensions":     "antigravity-extensions",     // Antigravity downloaded extensions
	".antigravity-ide/extensions": "antigravity-ide-extensions", // Antigravity IDE extensions
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
func (s *LangCacheScanner) Scan(root string) ([]Item, error) {
	// os.UserHomeDir reads $HOME on Linux/macOS and %USERPROFILE% on Windows.
	// Hardcoding a path like "/home/chamu" would break on other machines and OSes.
	home, err := os.UserHomeDir()
	if err != nil {
		// Home dir is required for this scanner — return the error rather than
		// silently reporting nothing.
		return nil, err
	}

	var items []Item

	for relPath, kind := range cacheRelPaths {
		// filepath.Join correctly handles OS path separators and cleans up any
		// double slashes or trailing separators.
		absPath := filepath.Join(home, filepath.FromSlash(relPath))

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

		size, modTime, _ := dirStats(absPath)
		items = append(items, Item{
			Path:      absPath,
			Kind:      kind,
			SizeBytes: size,
			LastMod:   modTime,
			IsDir:     true,
		})
	}

	return items, nil
}
