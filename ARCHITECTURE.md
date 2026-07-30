# sweepr — Architecture

## Why this shape
The core idea: **separate "finding junk" from "showing/deleting junk."**
That split is what lets you add a TUI or GUI later by writing a new
front-end that calls the same scanning code — instead of rewriting
everything.

```
sweepr/
├── go.mod
├── main.go              # CLI entrypoint: flags, wiring, output
├── dashboard/           # Bubble Tea mode, selection, and review state machine
├── remover/             # resource-specific filesystem and Docker deletion
├── trash/               # cross-platform recoverable filesystem removal
├── scanner/
│   ├── scanner.go        # Item struct + Scanner interface + registry
│   ├── util.go           # shared helpers (dirStats, fileStats)
│   ├── devjunk.go         # node_modules / build dirs
│   ├── langcache.go        # ~/.npm, ~/.cache/pip, etc.
│   └── osjunk.go            # .DS_Store, Thumbs.db, etc.
└── docs/
    ├── SPEC.md
    ├── ARCHITECTURE.md    (this file)
    └── ROADMAP.md
```

`main.go` (or, later, a TUI) never knows *how* junk is found — it just
calls `scanner.All()`, runs `.Scan(root)` on each, and gets back a slice
of `Item`. This is the same pattern Go tools like `golangci-lint` use for
their linters, and Kubernetes uses for its admission controllers — a
registry of things implementing one small interface.

## The core interface

```go
type Item struct {
    Path         string
    DisplayName  string
    Kind         string
    SizeBytes    int64
    LastMod      time.Time
    ResourceType ResourceType
}

type Scanner interface {
    Name() string
    Scan(root string, options ScanOptions) ([]Item, error)
}

type ProjectScanner interface {
    Scanner
    MatchProjectEntry(root, path string, entry fs.DirEntry) (Item, bool, bool)
}
```

`ResourceType` distinguishes files, directories, and non-filesystem resources
such as Docker images. This prevents a Docker ID from being passed to
`os.Remove` merely because it is not a directory. `DisplayName` is optional
human-readable text for resources whose stable identifier is not useful in a
report.

`ScanOptions` carries normalized exclusion paths shared by project-relative
walkers. Walk callbacks return `filepath.SkipDir` as soon as they reach an
excluded or protected snapshot directory. Pruning at traversal time avoids the
I/O, false positives, and deletion risk of scanning first and filtering later.

`ScanOptions` also carries an optional `ProgressFunc`. Walkers publish
point-in-time entry, finding, byte, and path counters without knowing anything
about terminal presentation. The CLI throttles rendering to ten updates per
second on interactive stderr, clears the temporary line before permanent
output, and disables progress for JSON or redirected streams. A percentage is
not reported because determining the total entry count would require a second
full filesystem traversal.

Directory measurement is observable through an optional callback in
`dirStatsWithProgress`. This keeps progress moving while nested size walks
measure large dependency and cache directories. The shared project walker owns
measurement and merges nested counts into its totals; classifiers only identify
resource type and kind.

Scanner scope is based on user intent, not path spelling. Omitting the root
includes global language caches; supplying an explicit root excludes them by
default; `--include-global` opts them back into a mixed scan. An explicit
`--only lang-cache` selection also runs because it is not an accidental global
side effect.

`DevJunkScanner` and `OSJunkScanner` additionally implement `ProjectScanner`.
The CLI groups selected project scanners into one `ScanProject` traversal and
offers each safe entry to their classifiers. Standalone `--only` modes use the
same engine with one classifier, so traversal policy does not fork.

Developer-junk patterns carry a confidence policy. Ecosystem-specific names
such as `node_modules` are direct matches, while ambiguous names (`build`,
`dist`, and `target`) require a nearby project marker. Marker search is bounded
to three ancestors and never crosses the selected scan root, preventing an
unrelated marker high in a broad tree from validating false positives.

Every junk-finder (dev dirs, development caches, folder OS metadata, system
caches, and Docker)
implements this. Adding a new junk type later = write one new
file implementing `Scan`, add it to the registry in `scanner.go`. Nothing
else changes.

Deletion uses a parallel registry in `remover/`. Each remover declares which
`ResourceType` values it supports and owns the corresponding removal mechanism.
Filesystem resources use `os.Remove` / `os.RemoveAll`, while Docker images use
`docker image rm`. This keeps non-filesystem identifiers away from filesystem
deletion and prevents `main.go` from accumulating resource-specific commands.

Recoverable filesystem removal is separate in `trash/`. It selects a native OS
adapter at runtime: GIO on Linux, Finder automation on macOS, and the Recycle
Bin API through PowerShell on Windows. It supports only files and directories,
fails closed when unavailable, and never falls back to permanent removal.

The Bubble Tea dashboard models action choice as one `Mode` value: read-only,
safe trash, or permanent deletion. Its state machine moves through mode choice,
item selection, and exact-target review before returning a confirmed `Result`
to `main`. The dashboard owns interaction but not side effects.

## Data flow

```
main.go
  │
  ├─ parse flags (root, --delete, --only, --min-size, ...)
  │
  ├─ for each Scanner in scanner.All():
  │      items := scanner.Scan(root)
  │      allItems = append(allItems, items...)
  │
  ├─ filter allItems by flags (--only/--skip/--min-size/--min-age)
  ├─ sort allItems by SizeBytes desc
  │
  ├─ print report (table or --json)
  │
  └─ choose output path:
       ├─ table / JSON
       ├─ classic --delete confirmation
       └─ TUI: choose mode → select → review → confirm
            ├─ read-only: no side effect
            ├─ trash: native OS trash for filesystem items
            └─ permanent: remover selected by ResourceType
```

## Walking Strategy & Performance
Project-relative scanners use a single master `filepath.WalkDir` traversal.
Traversal mechanics (pruning, symlinks, permissions, and progress) are applied
once, while enabled classifiers independently evaluate each entry. Global cache
and Docker scanners remain independent because they do not walk the project root.

## Traversal & Safety Safeguards
- **Symlink Loops:** To prevent infinite directory loops or walking outside the target directory root, symlinks (`os.ModeSymlink`) must not be followed.
- **Lock Files:** Files that are actively locked by running processes (e.g., node servers or IDE builders) should not crash the deletion phase. The deletion loop should log the error and proceed to clean other items.
- **Permission Denied:** Directories requiring elevated privileges (sudo) should be skipped gracefully during walk operations without interrupting the entire scan.
- **Snapshot & User Exclusions:** Timeshift snapshot roots and `.snapshots`
  directories are pruned automatically. Repeatable `--exclude` paths are
  normalized relative to the scan root and matched on path-component boundaries.

## Concurrency
The shared project walk, global-cache scanner, and Docker scanner run as
independent jobs using goroutines and `sync.WaitGroup`. Workers send progress
and final results over channels; the main goroutine alone aggregates results
and writes terminal output, preventing slice races and interleaved rendering.

## Why this supports "add a UI later"
- CLI output and deletion logic in `main.go` only *consumes* `[]Item` — a TUI (`bubbletea`) or a local web server (`net/http` + a JS frontend) would do the same: call `scanner.All()`, get `[]Item`, render it their own way, call the same delete logic on user-selected items.
- The `--json` flag matters for exactly this reason: it's a preview of the data contract a future UI would consume, and forces you to keep `Item` clean and serializable now.
