# sweepr

A fast command-line tool to find and report reclaimable disk space on developer machines — build artifacts, OS-generated clutter, and global package-manager caches.

```
Running scanner: dev-junk...
        node_modules  ./my-app/node_modules          312.44 MB  Last Mod: 2026-06-30
        python-cache  ./scripts/__pycache__            2.10 MB  Last Mod: 2026-07-01

Running scanner: lang-cache...
        npm-cache     /home/user/.npm                  2.76 GB  Last Mod: 2026-07-03
        gradle-cache  /home/user/.gradle/caches        3.39 GB  Last Mod: 2026-06-30

Total items: 4            Total size: 6.75 GB
```

## Usage

```bash
# Build the binary
go build -o sweepr

# Scan the current directory
./sweepr

# Scan a specific directory
./sweepr /path/to/projects

# Run only a specific scanner
./sweepr --only dev-junk

# Skip a scanner
./sweepr --skip lang-cache
```

## What It Scans

| Scanner | Name | What It Finds |
|---|---|---|
| **Dev Junk** | `dev-junk` | Build artifacts inside project trees: `node_modules`, `dist`, `build`, `.next`, `target`, `__pycache__`, `.venv`, `venv`, `.pytest_cache`, `.poetry` |
| **OS Junk** | `os-junk` | Files the OS drops into every directory: `.DS_Store` (macOS), `Thumbs.db`, `desktop.ini` (Windows) |
| **Lang Cache** | `lang-cache` | Global package manager caches under `$HOME`: npm, pip, cargo, Go modules, Go build cache, yarn, pnpm, Gradle |

## Platform Support

| Platform | Status |
|---|---|
| Linux | ✅ Fully supported |
| macOS | ✅ Fully supported |
| Windows | ⚠️ Partial — `dev-junk` and `os-junk` scanners work, but `lang-cache` does not yet detect Windows cache locations (`%AppData%\npm-cache` etc.) |

Windows cache path support is planned for a future release via Go build constraints.

## Architecture

The tool uses a registry-based plugin pattern:

- **`scanner.Scanner`** — Interface every scanner implements (`Name()` + `Scan(root)`).
- **`scanner.Item`** — Common struct all scanners return (path, kind, size, last-modified).
- **`scanner.All()`** — Registry that returns all built-in scanners. Adding a new scanner only requires implementing the interface and registering it here.
- **Symlink safety** — All filesystem walkers use `os.Lstat` (not `os.Stat`) to detect and skip symbolic links, preventing infinite loops and unsafe directory traversal.

## Project Status

This project is in active development. See [docs/ROADMAP.md](docs/ROADMAP.md) for planned phases.

Current: **Phase 4** — Advanced filtering (`--only`, `--skip`, `--min-size`, `--min-age`, size sorting).  
Next: **Phase 5** — Deletion & Safety Rules (`--delete`, confirmation prompts).
