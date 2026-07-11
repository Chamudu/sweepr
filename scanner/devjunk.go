package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
)

// devJunkNames maps well-known disposable directory names to a short "kind" label.
// The kind label is used in output formatting and future --only/--skip CLI filters.
//
// Design notes:
//   - Multiple directory names can map to the same kind label. For example,
//     "venv" and ".venv" both map to "python-venv" because they serve the same
//     purpose (Python virtual environments) and users should be able to filter
//     them together with --only python-venv.
//   - We use map[string]string (not a slice) so that each directory name lookup
//     is O(1) regardless of how many entries are in the map.
var devJunkNames = map[string]string{
	// JavaScript / TypeScript
	"node_modules": "node_modules", // npm / yarn / pnpm dependencies
	"dist":         "dist",         // generic compiled output
	"build":        "build",        // generic compiled output
	".next":        "next-cache",   // Next.js server-side build cache

	// Rust
	"target": "rust-target", // Cargo build artefacts (can reach 10+ GB)

	// Python
	"__pycache__":   "python-cache",  // CPython bytecode cache
	".venv":         "python-venv",   // Virtual environment (created by venv / poetry)
	"venv":          "python-venv",   // Virtual environment (alternate naming convention)
	".pytest_cache": "pytest-cache",  // pytest result cache
	".poetry":       "poetry-cache",  // Poetry package cache

	// Modern Frontend Frameworks
	".nuxt":        "nuxt-cache",       // Nuxt.js framework tracking state
	".svelte-kit":  "sveltekit-cache",   // SvelteKit compilation framework state
	".docusaurus":  "docusaurus-cache", // Static site documentation build folders
	".turbo":       "turborepo-cache",  // Turborepo local workspace caching records

	// Python & Data Science
	".ipynb_checkpoints": "jupyter-snapshots", // Jupyter Notebook automatic local backups
	".tox":               "tox-virtualenv",      // Python automated testing environments
	".mypy_cache":        "mypy-type-cache",     // Python strict type checking logs
	"htmlcov":            "python-coverage",     // Code coverage reports generated from tests

	// Native Compiled Environments
	".zig-cache":          "zig-local-cache",   // Zig project build logs
	"cmake-build-debug":   "cmake-debug",       // C/C++ CLion & CMake compilation folders
	"cmake-build-release": "cmake-release",     // C/C++ Production optimization assets
	".pnpm-store":         "pnpm-local-store",  // Soft-linked project engine items for pnpm

	// Infrastructure & Infrastructure as Code (IaC)
	".terraform":  "terraform-plugins",   // Cloned cloud providers & remote state configurations
	".serverless": "serverless-framework", // AWS Lambda / Cloud provider local deployment packaging
	".vagrant":    "vagrant-vm-state",    // Virtual box metadata tracking local environments
}


// DevJunkScanner finds disposable build-output and dependency directories
// inside a project tree. It skips .git directories entirely and stops
// descending into a directory once it recognises it as junk (so it does not
// double-count nested node_modules, for example).
type DevJunkScanner struct{}

// Name satisfies the Scanner interface. The returned string is used in output
// headers and future --only/--skip CLI filters.
func (s *DevJunkScanner) Name() string {
	return "dev-junk"
}

// Scan walks root recursively and returns all disposable directories found.
// Walking stops inside any recognised junk directory (filepath.SkipDir) so
// nested junk (e.g. node_modules inside node_modules) is not double-reported.
func (s *DevJunkScanner) Scan(root string) ([]Item, error) {
	var items []Item

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		// Skip any entry we cannot read (permission denied, broken symlink target, etc.).
		// Returning nil continues the walk instead of aborting it.
		if err != nil {
			return nil
		}

		// Symlink guard: use os.Lstat (does NOT follow symlinks) to get the raw
		// file mode. If the entry is a symlink we skip it entirely. This prevents:
		//   1. Infinite loops when a project has a symlink back to a parent dir.
		//   2. Accidentally walking into system directories via a symlink.
		info, lstatErr := os.Lstat(path)
		if lstatErr != nil {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// We only care about directories — individual files cannot be junk targets.
		if !d.IsDir() {
			return nil
		}

		// Skip .git entirely. We never want to scan or report Git internals.
		if d.Name() == ".git" {
			return filepath.SkipDir
		}

		// "Comma ok" map lookup: if d.Name() is in devJunkNames, ok is true and
		// kind holds the mapped label. If not found, ok is false and we continue.
		if kind, ok := devJunkNames[d.Name()]; ok {
			size, modTime, _ := dirStats(path)

			items = append(items, Item{
				Path:      path,
				Kind:      kind,
				SizeBytes: size,
				LastMod:   modTime,
				IsDir:     true,
			})

			// Skip descending into this directory — we've already counted it as a
			// whole, and we do not want to report junk nested inside junk.
			return filepath.SkipDir
		}

		return nil
	})

	return items, err
}