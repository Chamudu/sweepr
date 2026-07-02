package scanner

// cacheRelPaths are paths relative to $HOME for well-known package-manager
// caches.
//
// TODO(phase 2): fill in entries like:
//   ".npm":                     "npm-cache"
//   ".cache/pip":                "pip-cache"
//   ".cargo/registry/cache":     "cargo-cache"
//   "go/pkg/mod/cache/download": "go-mod-cache"
//   ".cache/go-build":           "go-build-cache"
// See docs/SPEC.md for the full list.
var cacheRelPaths = map[string]string{}

// LangCacheScanner looks at global cache locations under $HOME. Unlike
// DevJunkScanner these are NOT tied to a project root.
type LangCacheScanner struct{}

func (s *LangCacheScanner) Name() string {
	return ""
}

func (s *LangCacheScanner) Scan(root string) ([]Item, error) {
	// TODO(phase 2):
	//   - os.UserHomeDir()
	//   - for each rel path: os.Stat(filepath.Join(home, rel))
	//   - if it doesn't exist or isn't a dir, skip (not an error — normal)
	//   - otherwise dirStats() it and append an Item
	return nil, nil
}
