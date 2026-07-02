// Command sweepr finds and (optionally) deletes disposable dev/OS junk.
// See docs/SPEC.md, docs/ARCHITECTURE.md, docs/ROADMAP.md.
package main

func main() {
	// TODO(phase 1): hardcode root := "." and call scanner.All(), print
	// raw structs with fmt.Printf("%+v\n", item) to sanity-check scanning
	// works before building any real CLI/output around it.

	// TODO(phase 3): replace hardcoded root with flag.String, add -json.
	// Sort results by SizeBytes (sort.Slice) and print a human table with
	// human-readable byte sizes (write your own KB/MB/GB formatter).

	// TODO(phase 4): --only, --skip, --min-size, --min-age filtering.

	// TODO(phase 5): --delete + confirmation prompt (bufio.NewReader(os.Stdin))
	// + --yes to skip it. os.RemoveAll for dirs, os.Remove for files. Track
	// and print bytes freed. Keep going on individual failures.

	// TODO(phase 6): --json output via encoding/json.
}
