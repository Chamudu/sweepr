package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
)

// osJunkFiles is used as a set of file names that operating systems scatter
// automatically throughout any directory they interact with. We use
// map[string]bool (not map[string]string) because all entries share the same
// kind label ("os-junk") — the only question per entry is "does this name exist
// in the set?", so a bool value is sufficient.
var osJunkFiles = map[string]bool{
	".DS_Store":   true, // macOS: Finder stores folder view settings here
	"Thumbs.db":   true, // Windows: Explorer stores image thumbnail cache here
	"desktop.ini": true, // Windows: stores folder customisation settings here
}

// OSJunkScanner finds individual junk files scattered throughout a project
// tree. Unlike DevJunkScanner (which targets whole directories), this scanner
// targets single small files and uses fileStats (not dirStats) to measure them.
type OSJunkScanner struct{}

// Name satisfies the Scanner interface.
func (s *OSJunkScanner) Name() string {
	return "os-junk"
}

// Scan walks root recursively and returns all OS-generated junk files found.
// It skips .git directories and symbolic links for the same reasons as
// DevJunkScanner. Because the targets are files (not directories), it uses
// fileStats instead of dirStats.
func (s *OSJunkScanner) Scan(root string) ([]Item, error) {
	var items []Item

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		// Skip any entry we cannot read.
		if err != nil {
			return nil
		}

		// Skip .git before the symlink guard so we do not stat Git-internal paths.
		// We use d.IsDir() here because .git must be a directory to skip properly.
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}

		// Symlink guard: os.Lstat does NOT follow symlinks, so a symlink entry
		// shows up with os.ModeSymlink set rather than as its target type.
		// This prevents walking into unrelated parts of the filesystem.
		info, err := os.Lstat(path)
		if err != nil {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// We only want files — directories cannot be OS junk files.
		if d.IsDir() {
			return nil
		}

		// Check the file name against our set. osJunkFiles[name] returns false
		// if the name is not in the map, so no "comma ok" idiom is needed here.
		if osJunkFiles[d.Name()] {
			size, modTime, _ := fileStats(path) // fileStats: single-file stat, no walk needed
			items = append(items, Item{
				Path:      path,
				Kind:      "os-junk",
				SizeBytes: size,
				LastMod:   modTime,
				IsDir:     false,
			})
		}

		return nil
	})

	return items, err
}
