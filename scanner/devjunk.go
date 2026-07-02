package scanner

// devJunkNames maps a directory name we consider disposable to a friendly
// "kind" label.
//
// TODO(phase 1): start with just {"node_modules": "node_modules"} and get
// that fully working before filling in the rest.
// TODO(phase 2): add dist, build, .next, target, __pycache__, .pytest_cache,
// .venv, venv — see docs/SPEC.md for the full list.
var devJunkNames = map[string]string{}

// DevJunkScanner finds disposable dev directories under a project root
// (node_modules, build outputs, language-specific build/cache dirs).
type DevJunkScanner struct{}

func (s *DevJunkScanner) Name() string {
	// TODO: return an identifier, e.g. "dev-junk"
	return ""
}

func (s *DevJunkScanner) Scan(root string) ([]Item, error) {
	// TODO(phase 1):
	//   - filepath.WalkDir(root, ...)
	//   - skip .git entirely (return filepath.SkipDir)
	//   - if d.IsDir() && devJunkNames[d.Name()] exists:
	//       compute size + last-mod (see util.go TODO)
	//       append an Item
	//       return filepath.SkipDir so you don't recurse into it
	//   - handle walk errors by skipping the entry, not aborting
	return nil, nil
}
