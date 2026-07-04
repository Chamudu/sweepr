package main

import (
	"flag"
	"fmt"
	"os"
	"math"
	"sort"
	"sweepr/scanner"
	"time"
	"strings"
	"strconv"
	"unicode"
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

func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}


func filterScanner(all []scanner.Scanner, only, skip string) []scanner.Scanner {

	if only == "" && skip == "" {
		return  all
	}

	var result []scanner.Scanner

	var onlyList []string
	if only != "" {
		onlyList = strings.Split(only, ",")
	}

	var skipList []string
	if skip != "" {
		skipList = strings.Split(skip, ",")
	}

	for _, s := range all {
		if only != "" && !contains(onlyList, s.Name()) {
			continue
		}

		if skip != "" && contains(skipList, s.Name()) {
			continue
		}

		result = append(result, s)
	}

	return result
}

func parseSize(s string)(int64, error){
	// clean anyleading spaces
	s = strings.TrimSpace(s)

	var digitPart, unitPart string

	for i, r := range s {
		if !unicode.IsDigit(r) && r != '.' {
			digitPart = s[:i]
			unitPart = strings.TrimSpace(s[i:])
			break
		}
	}

	if digitPart == ""{
		digitPart = s
	}

	val, err := strconv.ParseFloat(digitPart, 64) 

	if err != nil {
		return 0, fmt.Errorf("Invalid size format: %s", s)
	}

	var multiplier float64 = 1

	switch strings.ToLower(unitPart) {
	case "", "b":
		multiplier = 1
	case "kb", "k":
		multiplier = 1024
	case "mb", "m":
		multiplier = 1024*1024
	case "gb", "g":
		multiplier = 1024*1024*1024
	case "tb", "t":
		multiplier = 1024*1024*1024*1024
	default:
		return 0, fmt.Errorf("Unknown unit %q in %s", unitPart, s)
	}


	return int64(val * multiplier), nil
}

func main() {
	only := flag.String("only", "", "run only this scanner (e.g. dev-junk)")
	skip := flag.String("skip", "", "skip this scanner by name")
	minSize := flag.String("min-size", "", "minimum size of items to report (e.g. 10MB, 500KB)")
	minAge := flag.Int("min-age", 0, "minimum age of items in days to report")

	flag.Parse()

	var minSizeBytes int64
	if *minSize != "" {
		var err error
		minSizeBytes, err = parseSize(*minSize)

		if err != nil {
			fmt.Printf("Error: invalid min-size format: %v\n", err)
			os.Exit(1)
		}
	}


	// root is the directory to scan for project-level junk (dev-junk, os-junk).
	// LangCacheScanner ignores this value and always checks $HOME.
	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	// totalItems and totalBytes accumulate counts across all scanners so we can
	// print a meaningful summary at the end. They live inside main() (not at the
	// package level) because they belong to a single scan run, not to the program.
	var totalItems int
	var totalBytes int64

	var allItems []scanner.Item

	scanners := filterScanner(scanner.All(), *only, *skip)

	fmt.Printf("Starting sweepr scan..\n")

	for _, s := range scanners {
		fmt.Printf("\nRunning scanner: %s...\n", s.Name())

		items, err := s.Scan(root)
		if err != nil {
			// A scanner error is non-fatal: report it and continue with the rest.
			fmt.Printf("Error running scanner %s: %v\n", s.Name(), err)
			continue
		}

		allItems = append(allItems, items...)	
	}
	fmt.Printf("\nScan Completed\n\n")

	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].SizeBytes > allItems[j].SizeBytes
	})

	for _, item := range allItems {
		if item.SizeBytes < minSizeBytes {
			continue
		}


		if *minAge > 0 {
			minAgeDuration := time.Duration(*minAge) * 24 * time.Hour
			if !item.LastMod.IsZero() && time.Since(item.LastMod) < minAgeDuration {
				continue
			}
		}

		fmt.Printf("\t%-22s %-40s %10s Last Mod: %s\n",
			item.Kind,
			item.Path,
			formatSize(item.SizeBytes),
			formatTime(item.LastMod),
		)
		totalBytes += item.SizeBytes
		totalItems++
	}
	

	fmt.Printf("\nTotal items: %-12d Total size: %s\n", totalItems, formatSize(totalBytes))
}
