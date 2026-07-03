package scanner

import (
	"os"
	"path/filepath"
)
var cacheRelPaths = map[string]string{
	".npm":                              "npm-cache",
	".cache/pip":                        "pip-cache",
	".cargo/registry/cache":             "cargo-cache",
	"go/pkg/mod/cache/download":         "go-mod-cache",
	".cache/go-build":                   "go-build-cache",
	".cache/yarn":                       "yarn-cache",
	".local/share/pnpm":                 "pnpm-cache",
	"Library/Developer/Xcode/DerivedData": "xcode-derived-data",  // macOS only
	".gradle/caches":                    "gradle-cache",
}

type LangCacheScanner struct{}

func (s *LangCacheScanner) Name() string {
	return "lang-cache"
}

func (s *LangCacheScanner) Scan(root string) ([]Item, error) {
	home, err := os.UserHomeDir()

	if err != nil {
		return nil, err
	}

	var items []Item

	for relPath, kind := range cacheRelPaths {
		absPath := filepath.Join(home, relPath)

		info, err := os.Stat(absPath)

		if err != nil {
			continue
		}

		if !info.IsDir() {
			continue
		}

		size, modTime, _ := dirStats(absPath)
		items = append(items, Item{
			Path: absPath,
			Kind: kind,
			SizeBytes: size,
			LastMod: modTime,
			IsDir: true,
		})

	}

	return items, nil
}
