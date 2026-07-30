# Releasing sweepr

This is a maintainer checklist. Creating and pushing a version tag publishes a
GitHub release, so complete every verification step first.

## Before tagging

1. Confirm the working tree is clean and `master` matches `origin/master`.
2. Confirm CI is green on Linux, macOS, and Windows.
3. Review `README.md`, `docs/GETTING_STARTED.md`, and safety wording.
4. Confirm `LICENSE`, `COPYRIGHT`, `SOURCE.md`, and
   `THIRD_PARTY_NOTICES.md` are present and correct.
5. Run locally:

```sh
go test -race ./...
go vet ./...
go build -o /tmp/sweepr-release-check .
/tmp/sweepr-release-check --version
/tmp/sweepr-release-check --license
```

## Tagging

Use semantic versions such as `v0.1.0`. Create an annotated tag so its purpose
is recorded in Git history:

```sh
git tag -a v0.1.0 -m "sweepr v0.1.0"
git push origin v0.1.0
```

The release workflow tests the tagged commit, embeds the tag/commit/build date,
builds six platform archives plus an exact tagged-source archive, writes
SHA-256 checksums, and creates the GitHub release with generated notes. Every
binary archive also includes the license, copyright notice, source directions,
third-party notices, README, and beginner guide.

## After publishing

1. Download one archive and verify its checksum.
2. Inspect the archive for its documentation and run `sweepr --version` and
   `sweepr --license`.
3. Open the dashboard and perform a read-only scan.
4. Confirm the release page contains all six binary archives, the tagged-source
   archive, and `checksums.txt`.

## Interactive workflow check

Before the first release, verify on Linux, macOS, and Windows that:

1. The first-run safety screen appears with an isolated/empty user config.
2. Directory browsing, parent navigation, and folder confirmation work.
3. Local, global, and combined scopes enable the documented scanners.
4. Advanced size, age, and exclusion filters affect the results.
5. Cancelling progress exits before the results and cleanup screens.
6. A second launch restores scan settings but starts cleanup in Read only.
