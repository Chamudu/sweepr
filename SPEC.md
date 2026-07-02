# sweepr — Project Spec

A cross-platform (Linux/macOS) CLI that finds and removes disposable
developer/OS junk: dead `node_modules`, build outputs, language caches,
and OS clutter files — safely.

## Goals
- Ship a fast, safe, and useful CLI tool that you'd actually run on your own machine.
- Start CLI-only; leave room to bolt on a TUI or GUI later without a rewrite.


## Non-goals (v1)
- Not a general-purpose disk analyzer (that's `ncdu`/`du`, don't reinvent it).
- Not networked / no telemetry.
- No auto-scheduling (cron) — that's a v2 idea, not v1.

## Functional Requirements

### 1. Scanning
The tool scans a given root path (default: current directory, or `$HOME`
for system-wide caches) and finds candidate junk:

| Category      | Examples                                                              | Scope                |
|----------------|------------------------------------------------------------------------|-----------------------|
| Dev junk       | `node_modules`, `dist`, `build`, `.next`, `target`, `__pycache__`, `.venv`, `.pytest_cache`, `.poetry` | walked from project root |
| Language caches| `~/.npm`, `~/.cache/pip`, `~/.cargo/registry`, `~/go/pkg/mod/cache`, `~/.cache/yarn`, `~/.local/share/pnpm` | fixed, under `$HOME`  |
| OS junk        | `.DS_Store`, `Thumbs.db`, `desktop.ini`                                 | walked from project root |
| macOS/iOS Dev  | `~/Library/Developer/Xcode/DerivedData`                                | fixed, under `$HOME`  |
| Android Dev    | `~/.gradle/caches`                                                     | fixed, under `$HOME`  |
| (stretch) Docker | dangling images, stopped containers, unused volumes                  | via `docker` CLI      |

For each candidate item, report:
- absolute path
- kind/category label
- size on disk (bytes)
- last-modified time (most recent file inside, for directories)

### 2. Safety rules (non-negotiable)
- **Never delete anything on first run.** Default mode is always a dry-run
  report.
- Deleting requires an explicit `--delete` flag.
- Even with `--delete`, prompt for confirmation per-item or with a
  "confirm all" summary — unless `--yes` is also passed.
- Never descend *into* a directory you've already flagged as junk (e.g.
  don't report nested `node_modules` inside a `node_modules` you're about
  to delete).
- Skip `.git` directories entirely (don't walk them, don't flag them).
- **Symlink Handling:** Skip following symbolic links during traversal to avoid infinite loops and deleting files outside the scanning root.
- **Lock & Permission Handling:** Handle permission errors by skipping the entry, not crashing the scan. Handle locked files during deletion by logging the error, tracking it, and continuing to clean remaining items.

### 3. Output
- Human-readable table sorted by size (largest first): path, kind, size
  (human units: KB/MB/GB), last modified, age in days.
- Print a total: "X items found, Y GB reclaimable."
- `--json` flag: same data as machine-readable JSON (useful later for a
  UI to consume).

### 4. Filtering flags
- `--only <kind,kind>` — restrict to specific categories.
- `--skip <kind,kind>` — exclude specific categories.
- `--min-size <e.g. 10MB>` — ignore anything smaller.
- `--min-age <days>` — ignore anything modified more recently than this
  (protects active projects you're still working in).

### 5. Deletion
- `--delete` performs the removal (directories: `os.RemoveAll`, files:
  `os.Remove`).
- Log what was deleted and how much space was freed, at the end.
- Exit non-zero if any deletion failed, but continue attempting the rest.

## Prior Art & Similar Programs

How `sweepr` compares to other tools in this space:

| Tool | Focus | Strengths | Weaknesses |
|------|-------|-----------|------------|
| **npkill** | `node_modules` only | Great interactive TUI | Requires Node; limited to JS ecosystem |
| **clean-dev-dirs** | Multi-language build dirs | Fast, Rust-based | CLI-only; doesn't clean OS/global caches |
| **dua-cli / dust** | General disk usage | Parallel scanning; beautiful TUIs | Shows everything; no safety filters for junk |
| **bleachbit** | System/browser cleanup | Deep cleaning of system caches | Heavy GUI focus; not dev-workflow optimized |

*The `sweepr` advantage:* A compiled Go binary with **zero startup latency** and **no runtime dependencies** that is specifically designed to scan for developer/OS junk safely.

## Future (v2+, not now)
- **Interactive TUI:** Select items with arrow keys / space to toggle, `d` to delete (using `bubbletea` + `lipgloss`).
- **Safe Trash/Recycle Bin:** Move deleted files to the system trash (e.g. via AppleScript on macOS or `gio` on Linux) instead of immediate permanent deletion.
- **Git Awareness:** Verify if directories are ignored in Git (e.g. via `.gitignore`) before flagging them as safe to delete.
- **Config file:** `.sweeprrc` or `sweepr.toml` for custom exclude paths, custom junk patterns, and ignore lists.
- **Docker/VM leftover cleanup:** cleanup via the `docker` CLI or SDK.
- Simple web UI or a native GUI.
