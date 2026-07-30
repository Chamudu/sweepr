// Package buildinfo exposes version metadata embedded in release binaries.
package buildinfo

import (
	"fmt"
	"runtime"
)

// These defaults describe a local development build. Release automation
// replaces them using Go linker -X flags without modifying source files.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// String returns a complete, support-friendly version line. Including the Go
// toolchain and target helps diagnose platform-specific user reports.
func String() string {
	return fmt.Sprintf("sweepr %s (commit %s, built %s, %s, %s/%s)",
		Version, Commit, Date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
