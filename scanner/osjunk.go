package scanner

import (
	"io/fs"
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

var _ ProjectScanner = (*OSJunkScanner)(nil)

// Name satisfies the Scanner interface.
func (s *OSJunkScanner) Name() string {
	return "os-junk"
}

// Scan walks root recursively and returns all OS-generated junk files found.
// It skips .git directories and symbolic links for the same reasons as
// DevJunkScanner. Because the targets are files (not directories), it uses
// fileStats instead of dirStats.
func (s *OSJunkScanner) Scan(root string, options ScanOptions) ([]Item, error) {
	return ScanProject(root, options, []ProjectScanner{s})
}

// MatchProjectEntry classifies individual OS-generated files. It never asks
// the shared walker to skip a directory because its targets are files.
func (s *OSJunkScanner) MatchProjectEntry(_, path string, entry fs.DirEntry) (Item, bool, bool) {
	if entry.IsDir() || !osJunkFiles[entry.Name()] {
		return Item{}, false, false
	}

	return Item{
		Path:         path,
		Kind:         "os-junk",
		ResourceType: ResourceFile,
	}, true, false
}
