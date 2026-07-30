# Contributing to sweepr

Thank you for helping improve sweepr. Bug reports, documentation fixes, tests,
and focused code changes are welcome.

## Before changing code

1. Read `SPEC.md` for behavior and safety boundaries.
2. Read `ARCHITECTURE.md` for package responsibilities.
3. For cleanup behavior, preserve the rule that scanning is read-only and
   deletion always requires explicit user intent.
4. Keep platform-specific behavior testable without touching real user data.

## Local checks

Format changed Go files and run the same core checks used for releases:

```sh
gofmt -w path/to/changed.go
go test ./...
go test -race ./...
go vet ./...
```

Add or update tests for changed behavior. Never write a test that deletes a
developer's real cache, trash, Docker image, or project directory.

## Pull requests

Keep each pull request focused, explain the user-visible behavior and safety
impact, and mention which operating systems were tested. By contributing, you
agree that your contribution is licensed under GPL-3.0-or-later.
