# Interactive UI feature plan

This document maps sweepr's command-line capabilities to the interactive UI.
It prevents a beginner-focused dashboard from accidentally hiding important
scan or safety controls.

## User flow

```text
first launch → welcome and safety → scan setup → scanning → results → review
later launch ─────────────────────→ scan setup → scanning → results → review
```

The first-launch acknowledgement is stored in the operating system's user
configuration directory. It is not stored inside the scanned project.

## Basic scan setup

| UI control | CLI equivalent | Default |
|---|---|---|
| Target directory | positional directory argument | Current directory |
| Developer junk | `--only` / `--skip dev-junk` | Enabled |
| OS junk | `--only` / `--skip os-junk` | Enabled |
| Language caches | `--only` / `--skip lang-cache` | Enabled only for the default scope |
| Docker images | `--only` / `--skip docker` | Enabled |
| System caches and temporary files | `--only` / `--skip system-cache` | Enabled in global scopes |
| Global user caches | `--include-global` | Off for an explicitly selected directory |

### Scope and scanner compatibility

The UI uses one explicit scope rather than making users combine `--only`,
`--skip`, and `--include-global` correctly.

| Scanner | Selected folder | Global resources | Folder + global |
|---|---:|---:|---:|
| Developer junk | Available | Disabled | Available |
| Folder OS metadata | Available | Disabled | Available |
| Global development caches | Disabled | Available | Available |
| System caches & temporary files | Disabled | Available | Available |
| Docker images | Disabled | Available | Available |

Disabled rows remain visible and explain why they cannot run in the selected
scope. For example, Docker images belong to the current Docker engine rather
than the chosen folder. Changing scope automatically removes incompatible
scanner selections and shows a notice; it never runs them silently.

System cleanup uses a conservative allowlist of documented, user-owned,
recreatable caches. It never edits the Windows registry or selects system logs,
Downloads, trash, cloud content, or privileged OS directories.

Within the interactive UI, scanner checkboxes produce one canonical enabled
set. This prevents contradictory CLI-style combinations such as selecting and
skipping the same scanner. CLI flags keep their existing behavior for scripts
and experienced users.

The target uses a terminal-native directory selector with folder-only rows,
parent and home navigation, scrolling, and explicit confirmation. This works
consistently on Linux, macOS, Windows, remote terminals, and SSH sessions
without adding a different native file-dialog dependency for every desktop
platform.

## Advanced scan setup

| UI control | CLI equivalent | Meaning |
|---|---|---|
| Excluded paths | repeatable `--exclude` | Never descend into these paths |
| Minimum size | `--min-size` | Hide smaller results |
| Minimum age | `--min-age` | Hide newer results |

## Result actions

Read only, safe trash, permanent deletion, target selection, and confirmation
already exist in the dashboard. Permanent deletion must never gain an
equivalent of CLI `--yes`; interactive users always review and confirm.

## Features that stay outside the dashboard

| CLI feature | Reason |
|---|---|
| `--json` | Intended for scripts and redirected output, not people |
| `--no-progress` | The dashboard owns and renders its own progress view |
| `--yes` | Bypassing confirmation conflicts with interactive safety |
| `--version` | Available before the UI starts and displayed in help/about |

## First-launch dialogue

The welcome screen must explain:

1. Scanning is read-only.
2. Finding an item does not mean it must be removed.
3. Safe trash is recoverable until trash is emptied and does not immediately
   reclaim space.
4. Permanent deletion cannot be undone through sweepr.
5. Global caches affect projects outside the selected directory.

The default action is **Continue safely**, and the first selected cleanup mode
remains **Read only**.

## Validation rules

- The target must exist and be a directory before scanning begins.
- At least one scanner must be enabled.
- Minimum size uses the same parser as the CLI.
- Minimum age cannot be negative.
- Invalid exclusions are shown before scanning, not silently ignored.
- Escape always returns to the previous screen; quitting never starts cleanup.
