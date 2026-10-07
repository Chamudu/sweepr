package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewScanOptionsAcceptsEmptyExcludes verifies that passing no exclusions
// produces valid options without error.
func TestNewScanOptionsAcceptsEmptyExcludes(t *testing.T) {
	root := t.TempDir()
	opts, err := NewScanOptions(root, nil)
	if err != nil {
		t.Fatalf("NewScanOptions(root, nil) = %v", err)
	}
	if opts.ShouldExclude(root) {
		t.Error("empty exclusion list incorrectly excluded the root itself")
	}
}

// TestNewScanOptionsRejectsEmptyExcludePath ensures that a blank string in the
// exclusion list is flagged rather than silently skipped.
func TestNewScanOptionsRejectsEmptyExcludePath(t *testing.T) {
	_, err := NewScanOptions(t.TempDir(), []string{""})
	if err == nil {
		t.Fatal("NewScanOptions with empty exclude path returned nil error")
	}
}

// TestNewScanOptionsRejectsWhitespaceExcludePath checks that an all-whitespace
// entry is also rejected.
func TestNewScanOptionsRejectsWhitespaceExcludePath(t *testing.T) {
	_, err := NewScanOptions(t.TempDir(), []string{"   "})
	if err == nil {
		t.Fatal("NewScanOptions with whitespace-only exclude path returned nil error")
	}
}

// TestShouldExcludeMatchesExactPath confirms that the excluded path itself is
// excluded.
func TestShouldExcludeMatchesExactPath(t *testing.T) {
	root := t.TempDir()
	vendor := filepath.Join(root, "vendor")
	opts, err := NewScanOptions(root, []string{vendor})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.ShouldExclude(vendor) {
		t.Error("ShouldExclude(vendor) = false, want true")
	}
}

// TestShouldExcludeMatchesDescendants confirms that children of an excluded
// path are also excluded.
func TestShouldExcludeMatchesDescendants(t *testing.T) {
	root := t.TempDir()
	vendor := filepath.Join(root, "vendor")
	child := filepath.Join(vendor, "github.com", "some", "dep")
	opts, err := NewScanOptions(root, []string{vendor})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.ShouldExclude(child) {
		t.Error("ShouldExclude(child of vendor) = false, want true")
	}
}

// TestShouldExcludeDoesNotMatchSiblingWithCommonPrefix verifies that excluding
// /projects/app does not also exclude /projects/app-old because the latter
// shares only a string prefix, not a path-component boundary.
func TestShouldExcludeDoesNotMatchSiblingWithCommonPrefix(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "app")
	appOld := filepath.Join(root, "app-old")
	opts, err := NewScanOptions(root, []string{app})
	if err != nil {
		t.Fatal(err)
	}
	if opts.ShouldExclude(appOld) {
		t.Errorf("ShouldExclude(%q) = true, but only %q was excluded — sibling-prefix false-positive", appOld, app)
	}
}

// TestShouldExcludeRelativePathJoinedToRoot confirms that a relative exclusion
// is resolved against the scan root, not the process working directory.
func TestShouldExcludeRelativePathJoinedToRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "vendor"), 0o700); err != nil {
		t.Fatal(err)
	}
	opts, err := NewScanOptions(root, []string{"vendor"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.ShouldExclude(filepath.Join(root, "vendor")) {
		t.Error("relative exclusion 'vendor' was not resolved against the root")
	}
}

// TestShouldExcludeMultipleExclusions confirms that multiple exclusion paths
// each work independently in the same ScanOptions.
func TestShouldExcludeMultipleExclusions(t *testing.T) {
	root := t.TempDir()
	vendor := filepath.Join(root, "vendor")
	cache := filepath.Join(root, ".cache")
	other := filepath.Join(root, "src")
	opts, err := NewScanOptions(root, []string{vendor, cache})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.ShouldExclude(vendor) {
		t.Error("vendor not excluded")
	}
	if !opts.ShouldExclude(cache) {
		t.Error(".cache not excluded")
	}
	if opts.ShouldExclude(other) {
		t.Error("src was incorrectly excluded")
	}
}

// TestIsProtectedSnapshotDirMatchesDotSnapshots checks the .snapshots form.
func TestIsProtectedSnapshotDirMatchesDotSnapshots(t *testing.T) {
	if !IsProtectedSnapshotDir("/mnt/data/.snapshots") {
		t.Error("IsProtectedSnapshotDir('.snapshots') = false, want true")
	}
}

// TestIsProtectedSnapshotDirMatchesTimeshiftSnapshots checks the Timeshift
// directory pair (case-insensitive on the parent).
func TestIsProtectedSnapshotDirMatchesTimeshiftSnapshots(t *testing.T) {
	for _, path := range []string{
		"/run/timeshift/snapshots",
		"/run/Timeshift/snapshots",
		"/run/TIMESHIFT/snapshots",
	} {
		if !IsProtectedSnapshotDir(path) {
			t.Errorf("IsProtectedSnapshotDir(%q) = false, want true", path)
		}
	}
}

// TestIsProtectedSnapshotDirDoesNotMatchGenericSnapshotsDir verifies that a
// "snapshots" directory not under "timeshift" is not protected, preventing
// false positives on project directories named "snapshots".
func TestIsProtectedSnapshotDirDoesNotMatchGenericSnapshotsDir(t *testing.T) {
	for _, path := range []string{
		"/home/user/projects/myapp/snapshots",
		"/var/backups/snapshots",
	} {
		if IsProtectedSnapshotDir(path) {
			t.Errorf("IsProtectedSnapshotDir(%q) = true, want false (generic path)", path)
		}
	}
}

// TestShouldSkipGlobalCacheDirHiddenHomeSubdir verifies that hidden
// directories directly under HOME (e.g., ~/.gradle) are skipped.
func TestShouldSkipGlobalCacheDirHiddenHomeSubdir(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{".gradle", ".npm", ".cache", ".azure"} {
		path := filepath.Join(home, name)
		if !ShouldSkipGlobalCacheDir(path, home) {
			t.Errorf("ShouldSkipGlobalCacheDir(%q) = false, want true", path)
		}
	}
}

// TestShouldSkipGlobalCacheDirGoPkg verifies the Go package cache is skipped.
func TestShouldSkipGlobalCacheDirGoPkg(t *testing.T) {
	home := t.TempDir()
	goPkg := filepath.Join(home, "go", "pkg")
	if !ShouldSkipGlobalCacheDir(goPkg, home) {
		t.Error("ShouldSkipGlobalCacheDir(go/pkg) = false, want true")
	}
	goPkgMod := filepath.Join(home, "go", "pkg", "mod")
	if !ShouldSkipGlobalCacheDir(goPkgMod, home) {
		t.Error("ShouldSkipGlobalCacheDir(go/pkg/mod) = false, want true")
	}
}

// TestShouldSkipGlobalCacheDirKnownVisibleTools verifies well-known visible
// tool directories directly under HOME are skipped.
func TestShouldSkipGlobalCacheDirKnownVisibleTools(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"google-cloud-sdk", "flutter", "miniconda3", "snap"} {
		path := filepath.Join(home, name)
		if !ShouldSkipGlobalCacheDir(path, home) {
			t.Errorf("ShouldSkipGlobalCacheDir(%q) = false, want true", path)
		}
	}
}

// TestShouldSkipGlobalCacheDirDoesNotSkipSourceDirs verifies that ordinary
// project source directories under HOME are not skipped.
func TestShouldSkipGlobalCacheDirDoesNotSkipSourceDirs(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"projects", "Documents", "go"} {
		// go/ itself should not be skipped, only go/pkg
		path := filepath.Join(home, name)
		if name == "go" {
			// go/ is not in the skip list; only go/pkg is.
			if ShouldSkipGlobalCacheDir(path, home) {
				t.Errorf("ShouldSkipGlobalCacheDir(go/) = true, want false (only go/pkg is skipped)")
			}
			continue
		}
		if ShouldSkipGlobalCacheDir(path, home) {
			t.Errorf("ShouldSkipGlobalCacheDir(%q) = true, want false", path)
		}
	}
}

// TestShouldSkipGlobalCacheDirEmptyHomeNeverSkips confirms that passing an
// empty home string never causes a skip (avoids false positives when $HOME is
// unset).
func TestShouldSkipGlobalCacheDirEmptyHomeNeverSkips(t *testing.T) {
	if ShouldSkipGlobalCacheDir("/some/path/.gradle", "") {
		t.Error("ShouldSkipGlobalCacheDir with empty home returned true")
	}
}

// TestDirStatsDoesNotFollowDirectorySymlinks confirms that dirStats does not
// recursively walk through a symlink that points to a directory. filepath.WalkDir
// visits symlink-to-dir as a non-directory entry (type L) and calls Lstat on it
// rather than entering the target, so the measured size reflects only the
// symlink inode itself — not the target's contents.
//
// This is the correct behavior: following directory symlinks would risk infinite
// loops and would count shared subtrees multiple times.
func TestDirStatsDoesNotFollowDirectorySymlinks(t *testing.T) {
	// Layout:
	//   root/real/big.bin   — 1 KiB of content
	//   root/link           — symlink → root/real
	//
	// Walking root/ should count big.bin once (from the real/ entry), and the
	// link entry should contribute only its inode size (a small number), NOT
	// the 1 KiB big.bin inside real/.
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.MkdirAll(real, 0o700); err != nil {
		t.Fatal(err)
	}
	content := make([]byte, 1024)
	if err := os.WriteFile(filepath.Join(real, "big.bin"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlinks not supported on this platform:", err)
	}

	// Walk the whole root. If the symlink were followed, we'd see big.bin twice
	// (once from real/ and once from link/), giving size >= 2048. Correct
	// behavior counts it only once, giving size == 1024 (plus a small inode).
	size, _, err := dirStats(root)
	if err != nil {
		t.Fatalf("dirStats(root) = %v", err)
	}
	// The total must be less than 2*len(content) to confirm no double-counting.
	if size >= int64(2*len(content)) {
		t.Errorf("dirStats size = %d, suggesting symlink-to-dir was followed and content double-counted (threshold %d)", size, 2*len(content))
	}
	// And the real file's 1024 bytes must be present.
	if size < int64(len(content)) {
		t.Errorf("dirStats size = %d, want at least %d (real file content)", size, len(content))
	}
}

// TestDirStatsIgnoresPermissionErrors verifies that dirStats returns a partial
// result rather than an error when it encounters an unreadable subdirectory.
func TestDirStatsIgnoresPermissionErrors(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can read all directories; skipping permission test")
	}
	root := t.TempDir()
	readable := filepath.Join(root, "readable")
	locked := filepath.Join(root, "locked")
	if err := os.MkdirAll(readable, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(readable, "file.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(locked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })

	size, _, err := dirStats(root)
	// dirStats must not return an error — it swallows individual entry errors.
	if err != nil {
		t.Fatalf("dirStats with unreadable subdir = %v", err)
	}
	// At minimum the readable file's bytes must be counted.
	if size < int64(len("hello")) {
		t.Errorf("dirStats size = %d, want at least %d (readable file)", size, len("hello"))
	}
}
