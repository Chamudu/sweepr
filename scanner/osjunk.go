package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
)

var osJunkFiles = map[string]bool{
	".DS_Store":   true,  // macOS: Finder metadata file (created in every folder)
	"Thumbs.db":   true,  // Windows: image thumbnail cache
	"desktop.ini": true,  // Windows: folder customization settings
}


type OSJunkScanner struct{}

func (s *OSJunkScanner) Name() string {
	return "os-junk"

}

func (s *OSJunkScanner) Scan(root string) ([]Item, error) {

	var items []Item

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}

		info, err := os.Lstat(path)
		if err != nil {
			return nil
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		
		if d.IsDir() {
			return nil
		}

		if osJunkFiles[d.Name()] {
			size, modTime, _ := fileStats(path)
			items = append(items, Item{
				Path: path,
				Kind: "os-junk",
				SizeBytes: size,
				LastMod: modTime,
				IsDir: false,
			})
		}

		return nil
	})
	
	return items, err
}
