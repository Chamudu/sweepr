package scanner

// osJunkFiles are files the OS scatters through every folder it touches.
//
// TODO(phase 2): fill in {".DS_Store": true, "Thumbs.db": true, "desktop.ini": true}
var osJunkFiles = map[string]bool{}

// OSJunkScanner finds individual junk files (not directories) scattered
// throughout a project tree.
type OSJunkScanner struct{}

func (s *OSJunkScanner) Name() string {
	return ""
}

func (s *OSJunkScanner) Scan(root string) ([]Item, error) {
	// TODO(phase 2):
	//   - filepath.WalkDir(root, ...)
	//   - skip .git
	//   - if !d.IsDir() && osJunkFiles[d.Name()]: fileStats() + append Item
	return nil, nil
}
