package scanner

// cacheRelPaths maps paths relative to the user's home directory ($HOME) to a
// friendly "kind" label. Unlike devJunkNames, these are NOT found by walking a
// project root — they are fixed, well-known paths where package managers store
// their global download caches.
//
// These paths vary by OS:
//   - Linux uses XDG dirs like ~/.cache/pip and ~/.local/share/pnpm
//   - macOS uses ~/Library/Developer/Xcode/DerivedData for Xcode
//   - Gradle/Android caches exist on both Linux and macOS
//
// If a path doesn't exist on the current machine, we skip it silently — that is
// EXPECTED behavior, not an error.
var cacheRelPaths = map[string]string{
	// TODO(phase 2): Uncomment these entries:
	// ".npm":                              "npm-cache",
	// ".cache/pip":                        "pip-cache",
	// ".cargo/registry/cache":             "cargo-cache",
	// "go/pkg/mod/cache/download":         "go-mod-cache",
	// ".cache/go-build":                   "go-build-cache",
	// ".cache/yarn":                       "yarn-cache",
	// ".local/share/pnpm":                 "pnpm-cache",
	// "Library/Developer/Xcode/DerivedData": "xcode-derived-data",  // macOS only
	// ".gradle/caches":                    "gradle-cache",
}

// LangCacheScanner looks at global cache locations under $HOME.
// Unlike DevJunkScanner and OSJunkScanner, this scanner does NOT walk a
// project directory. It checks fixed paths under the user's home directory.
// The 'root' parameter passed to Scan() is intentionally ignored.
type LangCacheScanner struct{}

func (s *LangCacheScanner) Name() string {
	// TODO(phase 2): Return "lang-cache"
	return ""
}

func (s *LangCacheScanner) Scan(root string) ([]Item, error) {
	// TODO(phase 2): Implement this function following these steps:
	//
	// 1. Get the user's home directory:
	//    home, err := os.UserHomeDir()
	//    if err != nil { return nil, err }
	//    WHY: We cannot hardcode "/home/username" or "/Users/username" because
	//    it differs per machine. Go's os.UserHomeDir() handles this universally.
	//
	// 2. Declare: var items []Item
	//
	// 3. Loop through the map:
	//    for relPath, kind := range cacheRelPaths {
	//
	// 4. Build the full absolute path:
	//    absPath := filepath.Join(home, relPath)
	//    WHY filepath.Join and not string concatenation (+)?
	//    filepath.Join handles OS-specific path separators (/ on Linux/macOS,
	//    \ on Windows) and cleans up double slashes.
	//
	// 5. Check if the path exists and is a directory:
	//    info, err := os.Stat(absPath)
	//    if err != nil { continue }      // Path doesn't exist on this machine — skip it
	//    if !info.IsDir() { continue }   // Exists but is a file, not a directory — skip
	//    WHY os.Stat here but os.Lstat in osjunk.go?
	//    For caches under $HOME, we WANT to follow symlinks. Some package managers
	//    (like pnpm) store their actual cache elsewhere and put a symlink at the
	//    well-known path. Following the symlink gives us the real size.
	//
	// 6. Calculate size:
	//    size, modTime, _ := dirStats(absPath)
	//
	// 7. Append item:
	//    items = append(items, Item{
	//        Path: absPath, Kind: kind,
	//        SizeBytes: size, LastMod: modTime, IsDir: true,
	//    })
	//    }  // end for loop
	//
	// 8. return items, nil
	return nil, nil
}
