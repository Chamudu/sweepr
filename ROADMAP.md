# sweepr — Build Roadmap

## Phase 0 — Setup
- [x] `go mod init sweepr`
- [x] Write `scanner/scanner.go`: `Item` struct + `Scanner` interface + `All()`

## Phase 1 — One working scanner end-to-end
- [x] Implement `DevJunkScanner` for just `node_modules` (skip the other
      dir names for now — get one working fully before generalizing).
- [x] Implement `dirStats` helper (`filepath.WalkDir`, summing file sizes,
      tracking latest mtime).
- [x] `main.go`: hardcode root to `"."`, call the scanner, print raw Go
      structs with `fmt.Printf("%+v\n", item)`.
- **Test it:** run inside a directory with a few `node_modules` folders,
  confirm sizes look roughly right vs `du -sh`.

## Phase 2 — Generalize + add more scanners
- [x] Expand `DevJunkScanner` to the full `devJunkNames` map (Node, Python, Rust, Go, Yarn, pnpm).
- [x] Add `OSJunkScanner` (`.DS_Store`, etc.) — simpler, no size-summing needed, just `os.Stat` on individual files. Check and skip symbolic links to avoid loops.
- [x] Add `LangCacheScanner` for `$HOME`-relative paths, including Xcode DerivedData and Android Gradle caches.

## Phase 3 — Real CLI
- [x] Replace hardcoded root with the `flag` package: `-root`, `-json`.
- [x] Human-readable table output, sorted by size (`sort.Slice`).
- [x] Byte-to-human formatting (KB/MB/GB) 

## Phase 4 — Filtering
- [x] `--only` / `--skip` (comma-separated kind filters).
- [x] `--min-size` (parse strings like `"10MB"` — you'll want a small parser function; this is a good use of `strconv` + a switch on suffix).
- [x] `--min-age` (days since `LastMod`, using `time.Since`).
- [x] Repeatable `--exclude` paths with traversal-time pruning and automatic
      protection for Timeshift / `.snapshots` backup trees.
- [x] Explicit `--include-global` scope control for fixed `$HOME` caches,
      replacing the ambiguous `"."` versus absolute-root behavior.

## Phase 5 — Deletion & Safety (the dangerous part — go slow)
- [x] `--delete` flag, off by default.
- [x] Confirmation prompt reading from stdin (`bufio.NewReader(os.Stdin)`).
- [x] `--yes` to skip confirmation.
- [x] `os.RemoveAll` for dirs, `os.Remove` for files; track bytes freed and print a summary.
- [x] Implement error handling for locked files: log the issue but proceed to clean remaining files.
- **Test it CAREFULLY:** point `-root` at a scratch directory you don't care about before ever running `--delete` on a real project tree.

## Phase 6 — JSON output
- [x] `--json`: marshal `[]Item` with `encoding/json`, add `json:"..."` struct tags to `Item`.
- [x] **Test it:** `sweepr -json | jq .` and confirm it's clean.

## Phase 7 (stretch) — Docker leftovers
- [x] Add a report-only `DockerScanner` implementing the existing `Scanner`
      interface and querying dangling images through `os/exec`.
- [x] Inspect images through Docker's structured JSON output to report exact
      byte sizes, creation times, and friendly short IDs.
- [x] Treat Docker as an optional integration and skip cleanly when its
      executable is not installed.
- [x] Add explicit resource types so Docker IDs cannot reach filesystem
      deletion (`os.Remove` / `os.RemoveAll`).
- [x] Add Docker-aware deletion through the Docker CLI with the same explicit
      confirmation and failure-reporting guarantees as filesystem deletion.

## Phase 8 (stretch) — Concurrency & Walking Optimizations
- [x] Add per-scanner duration reporting and throttled interactive progress
      without contaminating JSON or redirected output.
- [x] Keep progress responsive during nested directory-size measurement for
      project artifacts and global language caches.
- [x] Replace name-only matching for ambiguous build directories with bounded
      project-marker validation to reduce destructive false positives.
- [x] Run independent scan jobs concurrently with goroutines + `sync.WaitGroup`,
      collecting progress and results through channels while keeping terminal
      output owned by the main goroutine.
- [x] Implement **Single-Pass Walking**: traverse project files once and pass
      safe entries to all enabled project classifiers.

## Phase 9 (stretch) — UI & Safe Trash
- Pick one once the CLI is solid:
  - [x] **TUI:** Bubble Tea v2 — arrow-key navigation, space to toggle
        items for deletion, `d` to delete selected.
    - [x] Add an explicit `--tui` read-only dashboard with navigation,
          selection state, selected-byte totals, and an alternate-screen view.
    - [x] Add a review screen with exact targets, total size, back navigation,
          and explicit confirmation intent without performing deletion.
    - [x] Connect confirmed dashboard selections to the resource-aware remover.
    - [x] Add an in-dashboard mode chooser for read-only, safe trash, and
          permanent deletion, with mode-specific warnings and restrictions.
  - **Local web UI:** `net/http` server exposing `/scan` and `/delete`, with a small HTML/JS frontend.
  - **Native GUI:** `fyne.io/fyne`.
- [x] Add **Safe Trash Support** using native cross-platform adapters: GIO on
      Linux, Finder/AppleScript on macOS, and Recycle Bin APIs through
      PowerShell on Windows. Trash failures never fall back to permanent removal.

## Phase 10 — Portability & Release Confidence
- [x] Add GitHub Actions CI for native Linux, macOS, and Windows tests, vet, and
      builds, plus race detection on Linux.
- [x] Detect Windows-specific global language-cache locations.
- [x] Run opt-in native scratch-file trash integration tests on Linux, macOS,
      and Windows CI runners without touching developer trash during local tests.

## Phase 11 — Dashboard Usability
- [x] Add Lip Gloss visual hierarchy, mode-specific colors, bordered panels,
      contextual descriptions, and clearer safety messaging.
- [x] Add responsive terminal sizing, long-list viewports, Page Up/Page Down,
      first/last navigation, select-all, and clear-selection controls.
- [x] Test resize handling, viewport visibility, cell-aware truncation, and
      mode-safe bulk selection.

## Phase 12 — Release Readiness
- [x] Add `--version` with linker-injected version, commit, build date,
      toolchain, and target-platform information.
- [x] Add a beginner getting-started and safety guide plus a maintainer release
      checklist.
- [x] Add tag-triggered GitHub release automation for Linux, macOS, and Windows
      on amd64/arm64, including SHA-256 checksums and generated release notes.
- [x] Add an MIT license.
- [x] Add a scrollable beginner instruction manual and creator credit to the
      interactive dashboard.
- [ ] Review the beginner guide and publish `v0.1.0`.

## Phase 13 — Complete Interactive Workflow
- [x] Map CLI features to basic, advanced, and intentionally CLI-only UI
      controls.
- [x] Add a first-launch welcome and safety acknowledgement.
- [x] Add target-directory, scope, and scanner selection before scanning.
  - [x] Build a terminal-native, scrollable directory selector with explicit
        confirmation, parent/home navigation, and permission-error reporting.
  - [x] Define local, global, and combined scope compatibility with visible
        notices for unavailable scanners and conflict-free opt-out controls.
- [x] Add advanced exclusions, global-cache scope, minimum-size, and
      minimum-age controls.
- [x] Run scans from the dashboard with live per-scanner progress and
      cancellation before results or cleanup controls appear.
- [x] Store first-launch acknowledgement and safe preferences in the native
      user configuration directory.
- [ ] Test setup validation and the complete workflow on Linux, macOS, and
      Windows before publishing `v0.1.0`.

## Phase 14 — Conservative System Cleanup
- [x] Rename ambiguous UI labels to Folder OS metadata and Global development
      caches.
- [x] Add a global System caches & temporary files scanner using user-owned,
      platform-specific allowlists.
- [x] Cover Linux XDG thumbnails, macOS per-user application caches, Windows
      thumbnail databases, and Windows user-temp entries older than seven days.
- [x] Exclude registries, logs, Downloads, trash, cloud content, symlinks, and
      privileged system directories by design.
- [ ] Verify native findings and safe-trash behavior on Linux, macOS, and
      Windows before publishing `v0.1.0`.
