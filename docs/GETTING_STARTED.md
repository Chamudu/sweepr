# Getting started with sweepr

This guide is for people who are new to terminals, developer caches, or disk
cleanup tools. You do not need to understand Go to use sweepr.

## What sweepr does

Sweepr finds files that development tools can usually recreate, including
package caches, compiled output, `node_modules`, Python bytecode, OS metadata,
and dangling Docker images.

Finding an item does **not** automatically mean you should remove it. Some
items are cheap to recreate; others require a long build or another download.
Always read the description and review the exact path.

## The safest first run

Open a terminal in a project and run:

```sh
sweepr --tui
```

On the first launch, sweepr shows a safety introduction before scanning. Press
Enter to acknowledge it and continue, or Escape to exit without starting a
scan. The acknowledgement is saved in your operating system's user
configuration directory, never inside the project being scanned.

Choose **Read only**. This mode cannot change files. Move with the arrow keys,
select interesting rows with Space, and press `d` to preview them. Press `q` at
any time to exit.

Before scanning, the setup screen lets you browse for a directory and choose
**Selected folder only**, **Global resources only**, or **Folder + global
resources**. Scanners that do not belong to the selected scope stay visible as
`[-]` and explain why they are unavailable. Use Space to opt compatible
scanners in or out, then choose **Start scan**.

The **Advanced settings** page accepts a minimum result size such as `100MB`,
a minimum age in whole days, and multiple excluded directories. Highlight an
excluded path and press `x` to remove it. Scan settings are remembered for the
next launch, but cleanup mode, selected results, and confirmations are never
remembered.

If `sweepr` is not installed yet, see [Installing](#installing).

## Understanding scan scope

With no directory argument, sweepr scans the current project and global caches
under your user account:

```sh
sweepr
```

Supplying a directory scans project junk only. Global caches are excluded so a
broad path does not unexpectedly include unrelated user data:

```sh
sweepr /path/to/project
```

Add `--include-global` only when you intentionally want both scopes:

```sh
sweepr --include-global /path/to/project
```

## Dashboard modes

### Read only

Reports and previews selections without changing anything. Start here whenever
you are uncertain.

### Safe trash

Moves filesystem items to the operating-system trash or recycle bin. They stay
recoverable until that trash is emptied. Because the bytes still exist in the
trash, disk space is not reclaimed immediately.

Docker images cannot be moved to desktop trash and appear unavailable (`[-]`)
in this mode.

### Permanent delete

Permanently removes selected files, directories, or Docker images. This cannot
be undone through sweepr. Use it only after reviewing every target.

## Dashboard controls

| Key | Action |
|---|---|
| Arrow keys or `j`/`k` | Move between rows |
| Page Up / Page Down | Move one visible page |
| `g` / `G` | Jump to the first / last row |
| Space | Select or deselect the focused row |
| `a` | Select every item supported by the current mode |
| `c` | Clear the selection |
| `d` | Review selected targets |
| `i` | Open or close the instruction manual |
| Esc | Return to the previous screen |
| `q` | Quit without confirming |

## What the scanners mean

| Scanner | Examples | What happens after removal |
|---|---|---|
| `dev-junk` | `node_modules`, `dist`, `target`, Python caches | A package install or build recreates the data |
| `os-junk` | `.DS_Store`, `Thumbs.db` | The operating system may recreate the file |
| `lang-cache` | npm, pip, Go, Gradle caches | Future installs or builds may download/rebuild data |
| `docker` | Dangling images | Docker must rebuild or download the image again |

## Useful safe filters

Scan only one category:

```sh
sweepr --only dev-junk /path/to/projects
```

Ignore Docker:

```sh
sweepr --skip docker
```

Show only items at least 100 MB:

```sh
sweepr --min-size 100MB
```

Exclude a backup or mounted directory:

```sh
sweepr --exclude /path/to/backup /path/to/projects
```

Exclusions can be repeated. Timeshift and `.snapshots` trees are skipped
automatically.

## Installing a packaged release

1. Open the [sweepr releases page](https://github.com/Chamudu/sweepr/releases).
2. Open the newest release and choose the archive matching your computer:
   - `linux_amd64` for most Linux PCs
   - `linux_arm64` for ARM Linux devices
   - `darwin_arm64` for Apple Silicon Macs
   - `darwin_amd64` for Intel Macs
   - `windows_amd64` for most Windows PCs
   - `windows_arm64` for ARM Windows devices
3. Extract the downloaded `.tar.gz` or `.zip` archive.
4. Open a terminal in the extracted directory and check the binary:

Linux or macOS:

```sh
chmod +x sweepr
./sweepr --version
./sweepr --tui
```

Windows PowerShell:

```powershell
.\sweepr.exe --version
.\sweepr.exe --tui
```

The `chmod` command gives a Unix file permission called **execute** to the
binary. It does not run sweepr or grant administrator access.

Before the first packaged release is available, or if you want to learn how
the project is built, use the source instructions below.

## Building from source

1. Install the Go version declared in `go.mod`.
2. Clone the repository.
3. Open a terminal in the repository.
4. Run:

```sh
go build -o sweepr .
./sweepr --version
./sweepr --tui
```

On Windows PowerShell:

```powershell
go build -o sweepr.exe .
.\sweepr.exe --version
.\sweepr.exe --tui
```

## Platform requirements

- Linux safe trash uses `gio`. If it is unavailable, sweepr leaves the item
  untouched and reports an error.
- macOS safe trash uses Finder.
- Windows safe trash uses the Recycle Bin through PowerShell.
- Docker scanning and deletion require a running Docker service and permission
  to access it. Docker errors do not stop filesystem scanners.

Trash may be unavailable on temporary or special mounts such as `/tmp`. Sweepr
fails safely and does not replace a failed trash operation with permanent
deletion.

## Common messages

### `Trashing on system internal mounts is not supported`

The selected filesystem does not provide recoverable trash. The item remains
untouched. Try safe trash on a normal user directory, or choose permanent
deletion only if you truly do not need recovery.

### `permission denied ... docker.sock`

Your user cannot communicate with Docker. Filesystem scanning still works. Fix
Docker access using your operating system's Docker documentation, or run with
`--skip docker`.

### `--tui requires an interactive terminal`

The dashboard needs a real terminal. Do not pipe it to another command or a
file. For machine-readable output use:

```sh
sweepr --json
```

## A safe cleanup habit

1. Scan in read-only mode.
2. Filter large or old items if the list is overwhelming.
3. Read the focused-item description.
4. Prefer safe trash for filesystem items.
5. Review exact paths before confirmation.
6. Use permanent deletion only when recreation and recovery are understood.
