package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SystemCacheScanner finds conservative, user-owned OS caches. It never scans
// registries, system logs, Downloads, trash, or privileged system directories.
type SystemCacheScanner struct{}

func (s *SystemCacheScanner) Name() string { return "system-cache" }

type systemCandidate struct {
	path string
	kind string
}

func systemCandidates(goos, home string, getenv func(string) string, tempDir string, now time.Time) []systemCandidate {
	var candidates []systemCandidate
	appendChildren := func(root, kind string, olderThan time.Duration, excluded map[string]bool) {
		entries, err := os.ReadDir(root)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 || excluded[strings.ToLower(entry.Name())] {
				continue
			}
			if olderThan > 0 {
				info, err := entry.Info()
				if err != nil || now.Sub(info.ModTime()) < olderThan {
					continue
				}
			}
			candidates = append(candidates, systemCandidate{filepath.Join(root, entry.Name()), kind})
		}
	}

	switch goos {
	case "linux":
		cacheHome := getenv("XDG_CACHE_HOME")
		if cacheHome == "" {
			cacheHome = filepath.Join(home, ".cache")
		}
		path := filepath.Join(cacheHome, "thumbnails")
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			candidates = append(candidates, systemCandidate{path, "system-thumbnail-cache"})
		}
	case "darwin":
		// Developer caches already owned by LangCacheScanner are excluded to
		// prevent duplicate results when combined scope is selected.
		appendChildren(filepath.Join(home, "Library", "Caches"), "system-app-cache", 0, map[string]bool{"pip": true, "cocoapods": true})
	case "windows":
		local := getenv("LOCALAPPDATA")
		if local != "" {
			explorer := filepath.Join(local, "Microsoft", "Windows", "Explorer")
			entries, _ := os.ReadDir(explorer)
			for _, entry := range entries {
				name := strings.ToLower(entry.Name())
				if !entry.IsDir() && strings.HasPrefix(name, "thumbcache_") && strings.HasSuffix(name, ".db") {
					candidates = append(candidates, systemCandidate{filepath.Join(explorer, entry.Name()), "system-thumbnail-cache"})
				}
			}
		}
		// Windows documents cleanup of user temporary files. A seven-day floor
		// avoids presenting freshly-created working files by default.
		if pathWithin(home, tempDir) || pathWithin(local, tempDir) {
			appendChildren(tempDir, "system-temp", 7*24*time.Hour, nil)
		}
	}
	return candidates
}

func pathWithin(base, target string) bool {
	if base == "" || target == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(base), filepath.Clean(target))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func oldSystemTemp(lastModified, now time.Time) bool {
	return lastModified.IsZero() || now.Sub(lastModified) >= 7*24*time.Hour
}

func (s *SystemCacheScanner) Scan(_ string, options ScanOptions) ([]Item, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	candidates := systemCandidates(runtime.GOOS, home, os.Getenv, os.TempDir(), time.Now())
	items := make([]Item, 0, len(candidates))
	var entries, bytes int64
	for _, candidate := range candidates {
		info, err := os.Lstat(candidate.path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		item := Item{Path: candidate.path, Kind: candidate.kind, LastMod: info.ModTime(), ResourceType: ResourceFile}
		if info.IsDir() {
			item.ResourceType = ResourceDirectory
			item.SizeBytes, item.LastMod, _, _ = dirStatsWithProgress(candidate.path, func(path string, localEntries, localBytes int64) {
				options.ReportProgress(Progress{Path: path, EntriesScanned: entries + localEntries, ItemsFound: len(items), BytesFound: bytes + localBytes})
			})
		} else {
			item.SizeBytes = info.Size()
		}
		if candidate.kind == "system-temp" && !oldSystemTemp(item.LastMod, time.Now()) {
			continue
		}
		entries++
		bytes += item.SizeBytes
		items = append(items, item)
		options.ReportProgress(Progress{Path: candidate.path, EntriesScanned: entries, ItemsFound: len(items), BytesFound: bytes})
	}
	return items, nil
}
