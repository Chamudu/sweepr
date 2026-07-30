# Releasing sweepr

This is a maintainer checklist. Creating and pushing a version tag publishes a
GitHub release, so complete every verification step first.

## Before tagging

1. Confirm the working tree is clean and `master` matches `origin/master`.
2. Confirm CI is green on Linux, macOS, and Windows.
3. Review `README.md`, `docs/GETTING_STARTED.md`, and safety wording.
4. Confirm the repository license is present and correct.
5. Run locally:

```sh
go test -race ./...
go vet ./...
go build -o /tmp/sweepr-release-check .
/tmp/sweepr-release-check --version
```

## Tagging

Use semantic versions such as `v0.1.0`. Create an annotated tag so its purpose
is recorded in Git history:

```sh
git tag -a v0.1.0 -m "sweepr v0.1.0"
git push origin v0.1.0
```

The release workflow tests the tagged commit, embeds the tag/commit/build date,
builds six platform archives, writes SHA-256 checksums, and creates the GitHub
release with generated notes.

## After publishing

1. Download one archive and verify its checksum.
2. Run `sweepr --version` from the downloaded archive.
3. Open the dashboard and perform a read-only scan.
4. Confirm the release page contains all six archives and `checksums.txt`.
