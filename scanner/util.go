package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// dirStats walks a directory tree recursively and returns:
//   - the total size in bytes of all files found inside it
//   - the most recent file modification time seen during the walk
//   - any error returned by WalkDir itself (individual file errors are swallowed)
//
// Individual file errors (permission denied, broken symlinks) are swallowed
// intentionally — a partial size is more useful than reporting nothing at all.
//
// Note: dirStats only counts the size of regular files. Directory entries
// themselves have no meaningful size on Linux/macOS.
func dirStats(path string) (int64, time.Time, error) {
	var totalSize int64
	var maxMod time.Time

	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		// Skip files/directories we cannot read rather than aborting the whole walk.
		if err != nil {
			return nil
		}

		// Directory entries do not contribute to disk usage — only their contents do.
		if d.IsDir() {
			return nil
		}

		// d.Info() returns cached metadata from the WalkDir call, avoiding an
		// extra syscall compared to os.Stat(p).
		info, err := d.Info()
		if err != nil {
			return nil // skip this file, keep going
		}

		totalSize += info.Size()

		// Track the newest modification time seen across all files. This is used
		// to show when a cache or build folder was last actively used.
		if info.ModTime().After(maxMod) {
			maxMod = info.ModTime()
		}

		return nil
	})

	return totalSize, maxMod, err
}

// fileStats returns the size and modification time of a single file using a
// direct os.Stat call. It is the single-file equivalent of dirStats and is
// used by OSJunkScanner, where the target is an individual file (e.g. .DS_Store)
// rather than an entire directory tree.
func fileStats(path string) (int64, time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, time.Time{}, err
	}
	return info.Size(), info.ModTime(), nil
}

// ShouldSkipGlobalCacheDir returns true if the path points to a top-level hidden tool/cache
// directory in the user's home directory that we should never walk recursively.
// ShouldSkipGlobalCacheDir returns true if the path points to a tool/cache directory
// in the user's home directory that we should never walk recursively.
func ShouldSkipGlobalCacheDir(path string, home string) bool {
	if home == "" {
		return false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Calculate the relative path from HOME to the current directory
	rel, err := filepath.Rel(home, absPath)
	if err != nil {
		return false
	}

	sep := string(filepath.Separator)

	// 1. Skip any direct hidden subdirectories of HOME (e.g., ~/.gradle, ~/.npm, ~/.azure, ~/.vscode)
	// We check if it doesn't contain a path separator and starts with "."
	if !strings.Contains(rel, sep) && strings.HasPrefix(rel, ".") {
		return true
	}

	// 2. Skip Go package cache specifically (e.g., ~/go/pkg)
	if rel == "go"+sep+"pkg" || strings.HasPrefix(rel, "go"+sep+"pkg"+sep) {
		return true
	}

	// 3. Skip other common visible tool folders directly under HOME
	visibleToolsToSkip := []string{
		"google-cloud-sdk",
		"flutter",
		"miniconda",
		"miniconda3",
		"anaconda",
		"anaconda3",
		"snap",
	}
	for _, dir := range visibleToolsToSkip {
		if rel == dir {
			return true
		}
	}

	return false
}
