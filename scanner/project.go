package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
)

// ProjectScanner classifies entries supplied by the shared project walker.
// skipDir is true when a matched directory has been measured as one item and
// its descendants must not be visited or double-counted.
type ProjectScanner interface {
	Scanner
	MatchProjectEntry(root, path string, entry fs.DirEntry) (item Item, matched, skipDir bool)
}

// ScanProject walks a project root once and offers each readable entry to all
// enabled project scanners. Traversal policy lives here so exclusions, .git,
// symlinks, global-cache pruning, and progress cannot diverge between scanners.
func ScanProject(root string, options ScanOptions, scanners []ProjectScanner) ([]Item, error) {
	var items []Item
	var entriesScanned int64
	var bytesFound int64

	home, _ := os.UserHomeDir()
	absRoot, _ := filepath.Abs(root)

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		// Permission and per-entry metadata failures are non-fatal. A partial scan
		// is more useful than abandoning every result collected so far.
		if walkErr != nil {
			return nil
		}

		entriesScanned++
		if entriesScanned%256 == 0 {
			reportProjectProgress(options, path, entriesScanned, items, bytesFound)
		}

		if entry.IsDir() && (options.ShouldExclude(path) || IsProtectedSnapshotDir(path)) {
			return filepath.SkipDir
		}

		absPath, _ := filepath.Abs(path)
		if entry.IsDir() && absPath != absRoot && ShouldSkipGlobalCacheDir(path, home) {
			return filepath.SkipDir
		}

		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		for _, candidate := range scanners {
			item, matched, skipDir := candidate.MatchProjectEntry(root, path, entry)
			if !matched {
				continue
			}

			// Classifiers identify resources; the shared walker owns filesystem
			// measurement so nested progress and entry accounting stay consistent.
			if item.ResourceType == ResourceDirectory {
				baseEntries := entriesScanned
				baseBytes := bytesFound
				size, modTime, measuredEntries, _ := dirStatsWithProgress(
					item.Path,
					func(currentPath string, localEntries, localBytes int64) {
						nestedEntries := localEntries - 1 // outer walk already counted the directory root
						if nestedEntries < 0 {
							nestedEntries = 0
						}
						options.ReportProgress(Progress{
							Path:           currentPath,
							EntriesScanned: baseEntries + nestedEntries,
							ItemsFound:     len(items),
							BytesFound:     baseBytes + localBytes,
						})
					},
				)
				item.SizeBytes = size
				item.LastMod = modTime
				if measuredEntries > 0 {
					entriesScanned += measuredEntries - 1
				}
			} else if item.ResourceType == ResourceFile {
				item.SizeBytes, item.LastMod, _ = fileStats(item.Path)
			}

			items = append(items, item)
			bytesFound += item.SizeBytes
			reportProjectProgress(options, path, entriesScanned, items, bytesFound)

			if skipDir && entry.IsDir() {
				return filepath.SkipDir
			}
		}

		return nil
	})

	reportProjectProgress(options, root, entriesScanned, items, bytesFound)
	return items, err
}

func reportProjectProgress(options ScanOptions, path string, entriesScanned int64, items []Item, bytesFound int64) {
	options.ReportProgress(Progress{
		Path:           path,
		EntriesScanned: entriesScanned,
		ItemsFound:     len(items),
		BytesFound:     bytesFound,
	})
}
