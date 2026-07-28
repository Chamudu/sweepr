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

var _ ProjectScanner = (*DevJunkScanner)(nil)

// Name satisfies the Scanner interface. The returned string is used in output
// headers and future --only/--skip CLI filters.
func (s *DevJunkScanner) Name() string {
	return "dev-junk"
}

// Scan walks root recursively and returns all disposable directories found.
// Walking stops inside any recognised junk directory (filepath.SkipDir) so
// nested junk (e.g. node_modules inside node_modules) is not double-reported.
func (s *DevJunkScanner) Scan(root string, options ScanOptions) ([]Item, error) {
	return ScanProject(root, options, []ProjectScanner{s})
}

// MatchProjectEntry classifies a directory after the shared walker has already
// applied traversal safety rules.
func (s *DevJunkScanner) MatchProjectEntry(root, path string, entry fs.DirEntry) (Item, bool, bool) {
	if !entry.IsDir() {
		return Item{}, false, false
	}

	kind, ok := matchDevJunk(path, root)
	if !ok {
		return Item{}, false, false
	}

	return Item{
		Path:         path,
		Kind:         kind,
		ResourceType: ResourceDirectory,
	}, true, true
}
