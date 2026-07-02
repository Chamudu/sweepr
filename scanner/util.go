package scanner

import "time"

// dirStats walks path and returns the total size of all files inside it,
// plus the most recent modification time seen.
//
// TODO(phase 1):
//   - filepath.WalkDir(path, ...)
//   - for non-dir entries: add info.Size() to a running total
//   - track the max ModTime seen across all entries
//   - treat per-entry errors (permission denied etc.) as skip-not-abort
func dirStats(path string) (int64, time.Time, error) {
	return 0, time.Time{}, nil
}

// fileStats is the single-file equivalent of dirStats, for scanners that
// report individual junk files rather than whole directories.
//
// TODO(phase 2): os.Stat(path), return info.Size() and info.ModTime()
func fileStats(path string) (int64, time.Time, error) {
	return 0, time.Time{}, nil
}
