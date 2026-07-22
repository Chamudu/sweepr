package scanner

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ScanOptions contains behavior shared by project-relative scanners. Excluded
// paths are normalized once before scanning so every walker applies identical
// subtree-boundary checks.
type ScanOptions struct {
	excludedPaths  []string
	reportProgress ProgressFunc
}

// Progress is a point-in-time snapshot of a scanner's work. A total percentage
// is intentionally absent because discovering the total filesystem entry count
// would require an additional full traversal.
type Progress struct {
	Path           string
	EntriesScanned int64
	ItemsFound     int
	BytesFound     int64
}

// ProgressFunc receives progress snapshots from scanners. Callers may leave it
// nil when progress output is disabled.
type ProgressFunc func(Progress)

// WithProgress returns a copy configured with a reporter for one scan run.
func (o ScanOptions) WithProgress(reporter ProgressFunc) ScanOptions {
	o.reportProgress = reporter
	return o
}

// ReportProgress safely emits a snapshot when a reporter is configured.
func (o ScanOptions) ReportProgress(progress Progress) {
	if o.reportProgress != nil {
		o.reportProgress(progress)
	}
}

// NewScanOptions resolves relative exclusions against the scan root and stores
// absolute, cleaned paths. A missing exclusion is allowed because users may use
// the same command across machines with different directory layouts.
func NewScanOptions(root string, excludes []string) (ScanOptions, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return ScanOptions{}, fmt.Errorf("resolve scan root: %w", err)
	}

	options := ScanOptions{
		excludedPaths: make([]string, 0, len(excludes)),
	}
	for _, exclude := range excludes {
		exclude = strings.TrimSpace(exclude)
		if exclude == "" {
			return ScanOptions{}, fmt.Errorf("exclude path cannot be empty")
		}

		if !filepath.IsAbs(exclude) {
			exclude = filepath.Join(absRoot, exclude)
		}
		absExclude, err := filepath.Abs(exclude)
		if err != nil {
			return ScanOptions{}, fmt.Errorf("resolve exclude path %q: %w", exclude, err)
		}
		options.excludedPaths = append(options.excludedPaths, filepath.Clean(absExclude))
	}

	return options, nil
}

// ShouldExclude reports whether path is an excluded path or one of its
// descendants. filepath.Rel provides component-aware matching, avoiding false
// matches such as excluding /projects/app also excluding /projects/app-old.
func (o ScanOptions) ShouldExclude(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, excluded := range o.excludedPaths {
		rel, err := filepath.Rel(excluded, absPath)
		if err != nil {
			continue
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			return true
		}
	}
	return false
}

// IsProtectedSnapshotDir identifies well-known snapshot roots that broad scans
// should never enter by default. Matching the Timeshift directory pair avoids
// skipping an unrelated project directory merely named "snapshots".
func IsProtectedSnapshotDir(path string) bool {
	cleanPath := filepath.Clean(path)
	if filepath.Base(cleanPath) == ".snapshots" {
		return true
	}
	return filepath.Base(cleanPath) == "snapshots" &&
		strings.EqualFold(filepath.Base(filepath.Dir(cleanPath)), "timeshift")
}
