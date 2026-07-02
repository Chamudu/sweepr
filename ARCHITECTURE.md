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
    Path      string
    Kind      string
    SizeBytes int64
    LastMod   time.Time
    IsDir     bool
}

type Scanner interface {
    Name() string
    Scan(root string) ([]Item, error)
}
```

Every junk-finder (dev dirs, language caches, OS junk files, and later
Docker) implements this. Adding a new junk type later = write one new
file implementing `Scan`, add it to the registry in `scanner.go`. Nothing
else changes.

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
  └─ if --delete:
         confirm (unless --yes)
         for each item: os.RemoveAll or os.Remove
         print freed-space summary
```

## Walking Strategy & Performance
To scan project directories efficiently, the walk implementation can follow one of two strategies:
1. **Multi-Pass Scan (Simpler, sequential):** Each scanner walks the target root independently (`filepath.WalkDir`). While easy to implement, this leads to redundant disk I/O, as the OS has to traverse the same directories multiple times.
2. **Single-Pass Scan (Optimized):** A single master filesystem walker traverses the root directory and evaluates each path against the patterns of all active project-relative scanners in one pass. This significantly reduces disk seek times, especially on HDDs.

## Traversal & Safety Safeguards
- **Symlink Loops:** To prevent infinite directory loops or walking outside the target directory root, symlinks (`os.ModeSymlink`) must not be followed.
- **Lock Files:** Files that are actively locked by running processes (e.g., node servers or IDE builders) should not crash the deletion phase. The deletion loop should log the error and proceed to clean other items.
- **Permission Denied:** Directories requiring elevated privileges (sudo) should be skipped gracefully during walk operations without interrupting the entire scan.

## Concurrency (for later, not v1)
Once the sequential scanning is solid, a natural speedup is to run scans concurrently using `sync.WaitGroup` + goroutines. Since global caches and project directories reside in independent parts of the filesystem, concurrency will leverage modern multi-core NVMe drives effectively.

## Why this supports "add a UI later"
- CLI output and deletion logic in `main.go` only *consumes* `[]Item` — a TUI (`bubbletea`) or a local web server (`net/http` + a JS frontend) would do the same: call `scanner.All()`, get `[]Item`, render it their own way, call the same delete logic on user-selected items.
- The `--json` flag matters for exactly this reason: it's a preview of the data contract a future UI would consume, and forces you to keep `Item` clean and serializable now.
