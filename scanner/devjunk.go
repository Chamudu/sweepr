package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
)

// Map directory names that are disposable to a friendly "kind" label.
var devJunkNames = map[string]string{
	"node_modules"	: "node_modules",
	"dist"			: "dist",
	"build"			: "build",
	".next"			: "next-cache",
	"target"		: "rust-target",
	"__pycache__"	: "python-cache",
	".venv"			: "python-venv",
	"venv"			: "python-venv",
	".pytest_cache"	: "pytest-cache",
	".poetry"		: "poetry-cache",
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
		// 1 skip file that cannot be read
		if err != nil {
			return nil
		}

		// 2 symlink guard(get raw file info without following links)
		info, lstatErr := os.Lstat(path)
		if lstatErr != nil {
			return nil	
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return nil // skip symlink to prevent loops and unsafe traversal
		}

		// 3 skip non directories	
		if !d.IsDir() {
			return nil	
		}

		// 4. skip .git directories
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