package main

import (
	"fmt"
	"math"
	"sweepr/scanner"
	"time"
)

// formatSize converts a raw byte count into a human-readable string with the
// appropriate unit (B, KB, MB, GB, TB, PB, EB). Uses binary units (1024-based),
// which is the standard for disk space.
//
// Examples:
//
//	formatSize(512)        → "512 B"
//	formatSize(1536)       → "1.50 KB"
//	formatSize(1073741824) → "1.00 GB"
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	// Determine the unit exponent (1=KB, 2=MB, 3=GB, ...) using log base 1024.
	// math.Log2(bytes)/10 gives the same result because log2(1024^n) = 10n.
	exp := int(math.Log2(float64(bytes)) / 10)

	// Cap at EB (10^6) to prevent the "KMGTPE" slice from going out of bounds
	// for astronomically large values.
	if exp > 6 {
		exp = 6
	}

	div := math.Pow(unit, float64(exp))
	return fmt.Sprintf("%.2f %cB", float64(bytes)/div, "KMGTPE"[exp-1])
}

// formatTime returns the time as "YYYY-MM-DD HH:MM:SS", or "Never" for the
// zero value. The zero value occurs when a scanned directory was empty, meaning
// dirStats never found any files with a modification time to track.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	return t.Format("2006-01-02 15:04:05")
}

func main() {
	// root is the directory to scan for project-level junk (dev-junk, os-junk).
	// LangCacheScanner ignores this value and always checks $HOME.
	// Phase 3 will replace this hardcoded value with a CLI argument.
	root := "."

	// totalItems and totalBytes accumulate counts across all scanners so we can
	// print a meaningful summary at the end. They live inside main() (not at the
	// package level) because they belong to a single scan run, not to the program.
	var totalItems int
	var totalBytes int64

	scanners := scanner.All()

	fmt.Printf("Starting sweepr scan..\n")

	for _, s := range scanners {
		fmt.Printf("\nRunning scanner: %s...\n", s.Name())

		items, err := s.Scan(root)
		if err != nil {
			// A scanner error is non-fatal: report it and continue with the rest.
			fmt.Printf("Error running scanner %s: %v\n", s.Name(), err)
			continue
		}

		for _, item := range items {
			fmt.Printf("\t%-22s %-40s %10s Last Mod: %s\n",
				item.Kind,
				item.Path,
				formatSize(item.SizeBytes),
				formatTime(item.LastMod),
			)

			totalBytes += item.SizeBytes
			totalItems++
		}
	}

	fmt.Printf("\nTotal items: %-12d Total size: %s\n", totalItems, formatSize(totalBytes))
}
