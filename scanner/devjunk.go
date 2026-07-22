package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// MatchConfidence describes how much evidence is required before a directory
// name can be treated as disposable developer output.
type MatchConfidence string

const (
	// ConfidenceHigh is reserved for names whose ecosystem meaning is specific
	// enough to report directly, such as node_modules and __pycache__.
	ConfidenceHigh MatchConfidence = "high"
	// ConfidenceProject requires a nearby project marker because generic names
	// such as build and dist also occur in source trees and SDK metadata.
	ConfidenceProject MatchConfidence = "project-marker-required"
)

// devJunkPattern describes both the result label and the evidence required to
// accept a directory-name match.
type devJunkPattern struct {
	Kind       string
	Confidence MatchConfidence
	Markers    []string
}

var commonBuildMarkers = []string{
	"package.json",
	"pyproject.toml",
	"setup.py",
	"build.gradle",
	"build.gradle.kts",
	"settings.gradle",
	"settings.gradle.kts",
	"CMakeLists.txt",
	"pom.xml",
}

// devJunkPatterns maps well-known disposable directory names to matching
// metadata. A map keeps name lookup O(1); the pattern value carries the safety
// policy that a plain map[string]string could not express.
//
// Design notes:
//   - Multiple directory names can map to the same kind label. For example,
//     "venv" and ".venv" both map to "python-venv" because they serve the same
//     purpose (Python virtual environments) and users should be able to filter
//     them together with --only python-venv.
//   - Ambiguous names are accepted only with nearby ecosystem evidence.
var devJunkPatterns = map[string]devJunkPattern{
	// JavaScript / TypeScript
	"node_modules": {Kind: "node_modules", Confidence: ConfidenceHigh},
	"dist":         {Kind: "dist", Confidence: ConfidenceProject, Markers: commonBuildMarkers},
	"build":        {Kind: "build", Confidence: ConfidenceProject, Markers: commonBuildMarkers},
	".next":        {Kind: "next-cache", Confidence: ConfidenceHigh},

	// Rust
	"target": {Kind: "rust-target", Confidence: ConfidenceProject, Markers: []string{"Cargo.toml"}},

	// Python
	"__pycache__":   {Kind: "python-cache", Confidence: ConfidenceHigh},
	".venv":         {Kind: "python-venv", Confidence: ConfidenceHigh},
	"venv":          {Kind: "python-venv", Confidence: ConfidenceHigh},
	".pytest_cache": {Kind: "pytest-cache", Confidence: ConfidenceHigh},
	".poetry":       {Kind: "poetry-cache", Confidence: ConfidenceHigh},

	// Modern Frontend Frameworks
	".nuxt":       {Kind: "nuxt-cache", Confidence: ConfidenceHigh},
	".svelte-kit": {Kind: "sveltekit-cache", Confidence: ConfidenceHigh},
	".docusaurus": {Kind: "docusaurus-cache", Confidence: ConfidenceHigh},
	".turbo":      {Kind: "turborepo-cache", Confidence: ConfidenceHigh},

	// Python & Data Science
	".ipynb_checkpoints": {Kind: "jupyter-snapshots", Confidence: ConfidenceHigh},
	".tox":               {Kind: "tox-virtualenv", Confidence: ConfidenceHigh},
	".mypy_cache":        {Kind: "mypy-type-cache", Confidence: ConfidenceHigh},
	"htmlcov":            {Kind: "python-coverage", Confidence: ConfidenceHigh},

	// Native Compiled Environments
	".zig-cache":          {Kind: "zig-local-cache", Confidence: ConfidenceHigh},
	"cmake-build-debug":   {Kind: "cmake-debug", Confidence: ConfidenceHigh},
	"cmake-build-release": {Kind: "cmake-release", Confidence: ConfidenceHigh},
	".pnpm-store":         {Kind: "pnpm-local-store", Confidence: ConfidenceHigh},

	// Infrastructure & Infrastructure as Code (IaC)
	".terraform":  {Kind: "terraform-plugins", Confidence: ConfidenceHigh},
	".serverless": {Kind: "serverless-framework", Confidence: ConfidenceHigh},
	".vagrant":    {Kind: "vagrant-vm-state", Confidence: ConfidenceHigh},
}

const projectMarkerSearchDepth = 3

// matchDevJunk applies the evidence policy for a directory-name candidate.
func matchDevJunk(path, scanRoot string) (string, bool) {
	pattern, ok := devJunkPatterns[filepath.Base(path)]
	if !ok {
		return "", false
	}
	if pattern.Confidence == ConfidenceHigh {
		return pattern.Kind, true
	}
	if hasNearbyProjectMarker(filepath.Dir(path), scanRoot, pattern.Markers) {
		return pattern.Kind, true
	}
	return "", false
}

// hasNearbyProjectMarker searches the candidate's parent and a small number of
// ancestors, stopping at the scan root. Limiting depth prevents an unrelated
// marker high in a broad tree from validating every generic build directory.
func hasNearbyProjectMarker(start, scanRoot string, markers []string) bool {
	current, err := filepath.Abs(start)
	if err != nil {
		return false
	}
	absRoot, err := filepath.Abs(scanRoot)
	if err != nil {
		return false
	}

	for depth := 0; depth <= projectMarkerSearchDepth; depth++ {
		for _, marker := range markers {
			info, err := os.Stat(filepath.Join(current, marker))
			if err == nil && !info.IsDir() {
				return true
			}
		}

		if current == absRoot {
			break
		}
		parent := filepath.Dir(current)
		if parent == current || !pathWithinRoot(parent, absRoot) {
			break
		}
		current = parent
	}
	return false
}

func pathWithinRoot(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator))
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
func (s *DevJunkScanner) Scan(root string, options ScanOptions) ([]Item, error) {
	var items []Item
	var entriesScanned int64
	var bytesFound int64

	home, _ := os.UserHomeDir()      // Fetch home dir once
	absRoot, _ := filepath.Abs(root) // Gets absolute path of the scan root

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		// Skip any entry we cannot read (permission denied, broken symlink target, etc.).
		// Returning nil continues the walk instead of aborting it.
		if err != nil {
			return nil
		}

		entriesScanned++
		// Reporting every entry would make time formatting and terminal rendering
		// part of the hot filesystem loop. Periodic snapshots stay responsive with
		// substantially less overhead.
		if entriesScanned%256 == 0 {
			options.ReportProgress(Progress{
				Path:           path,
				EntriesScanned: entriesScanned,
				ItemsFound:     len(items),
				BytesFound:     bytesFound,
			})
		}

		// Prune excluded and protected snapshot trees before inspecting their
		// contents. SkipDir prevents both false reports and unnecessary disk I/O.
		if d.IsDir() && (options.ShouldExclude(path) || IsProtectedSnapshotDir(path)) {
			return filepath.SkipDir
		}

		// skip top-level global tool caches inside the home folder to avoid leaks
		// Never skip root dir if user target it
		absPath, _ := filepath.Abs(path)
		if d.IsDir() && absPath != absRoot && ShouldSkipGlobalCacheDir(path, home) {
			return filepath.SkipDir
		}

		// Symlink guard: Check the file type bits directly in memory using the fs.DirEntry.
		if d.Type()&fs.ModeSymlink != 0 {
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

		if kind, ok := matchDevJunk(path, root); ok {
			size, modTime, _ := dirStats(path)
			bytesFound += size

			items = append(items, Item{
				Path:         path,
				Kind:         kind,
				SizeBytes:    size,
				LastMod:      modTime,
				ResourceType: ResourceDirectory,
			})
			options.ReportProgress(Progress{
				Path:           path,
				EntriesScanned: entriesScanned,
				ItemsFound:     len(items),
				BytesFound:     bytesFound,
			})

			// Skip descending into this directory — we've already counted it as a
			// whole, and we do not want to report junk nested inside junk.
			return filepath.SkipDir
		}

		return nil
	})

	options.ReportProgress(Progress{
		Path:           root,
		EntriesScanned: entriesScanned,
		ItemsFound:     len(items),
		BytesFound:     bytesFound,
	})

	return items, err
}
