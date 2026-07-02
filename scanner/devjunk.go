package scanner

import (
	"io/fs"
	"path/filepath"
	
)

// Map directory names that are disposable to a friendly "kind" label.
var devJunkNames = map[string]string{
	"node_modules" : "node_modules",
}

//find disposable dev directories under a project root
type DevJunkScanner struct{}

// returns a short identifier used in output and filters
func (s *DevJunkScanner) Name() string {
	return "dev-junk"
}

// scan walks the root dir recursively and returns all disposable folders
func (s *DevJunkScanner) Scan(root string) ([]Item, error) {
	var items []Item

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() {
			return nil	
		}

		if d.Name() == ".git" {
			return filepath.SkipDir
		}

		if kind, ok := devJunkNames[d.Name()]; ok {
			size, modTime, _ := dirStats(path)

			items = append(items, Item{
				Path:	path,
				Kind: 	kind,
				SizeBytes: size,
				LastMod: modTime,
				IsDir: 	true,
			})

			return filepath.SkipDir
		}

		return nil
	})

	return items, err
}