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
- [ ] Add Docker-aware deletion through the Docker CLI with the same explicit
      confirmation and failure-reporting guarantees as filesystem deletion.

## Phase 8 (stretch) — Concurrency & Walking Optimizations
- [ ] Run all scanners concurrently with goroutines + `sync.WaitGroup`, collect results via a channel or a mutex-protected slice.
- [ ] Implement **Single-Pass Walking**: refactor the walkers to do a single directory traversal, passing paths to a matcher routine to avoid redundant disk I/O.

## Phase 9 (stretch) — UI & Safe Trash
- Pick one once the CLI is solid:
  - **TUI:** `github.com/charmbracelet/bubbletea` + `lipgloss` — arrow-key navigation, space to toggle items for deletion, `d` to delete selected.
  - **Local web UI:** `net/http` server exposing `/scan` and `/delete`, with a small HTML/JS frontend.
  - **Native GUI:** `fyne.io/fyne`.
- [ ] Add **Safe Trash Support**: Integrate trash libraries (like `github.com/electron/trash` equivalents, or native AppleScript/gio shell-outs) to move items to system trash instead of calling `RemoveAll`.
