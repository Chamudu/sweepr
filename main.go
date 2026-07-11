package main

import (
	"flag"
	"fmt"
	"encoding/json"
	"os"
	"math"
	"sort"
	"sweepr/scanner"
	"time"
	"strings"
	"strconv"
	"unicode"
	"bufio"
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

func deleteJunk(filteredItems []scanner.Item) {
	var deletedCount int
	var freedBytes int64
	var failedCount int

	fmt.Println() 

	for _, item := range filteredItems {
		var err error 
		
		if item.IsDir {
			err = os.RemoveAll(item.Path)
		} else {
			err = os.Remove(item.Path)
		}

		if err != nil {
			fmt.Printf("Error deleting %s: %v\n", item.Path, err)
			failedCount++
			continue
		}

		deletedCount++
		freedBytes += item.SizeBytes
		fmt.Printf("Deleted: %s (%s)\n", item.Path, formatSize(item.SizeBytes))
	}

	// Finaly Summary
	fmt.Printf("\nSuccessfully deleted %d items, freed %s of space.\n", deletedCount, formatSize(freedBytes))

	if failedCount > 0 {
		fmt.Printf("Warning: failed to delete %d items.\n", failedCount)
		os.Exit(1)
	}
}

func header(root string, scanners []scanner.Scanner) {
	home, _ := os.UserHomeDir()

	fmt.Printf("\nStarting sweepr scan..\n")
	fmt.Printf("\nTarget Directory: %s\n", root)
	
	// Show scope info
	if root == "." || root == home {
		fmt.Printf("\nScope: \tProject files + Global user caches (%s)\n", home)
	} else {
		fmt.Printf("\nScope: \tSubfolder scan (Global caches ignored for safety)\n")
	}

	// Show active scanners list
	var activeNames []string
	for _, s := range scanners {
		activeNames = append(activeNames, s.Name())
	}
	fmt.Printf("Scanners: \t%s\n\n", strings.Join(activeNames, ", "))

}

func main() {

	// FLAGS (only, skip, minSize, minAge)
	only := flag.String("only", "", "run only this scanner (e.g. dev-junk)")
	skip := flag.String("skip", "", "skip this scanner by name")
	minSize := flag.String("min-size", "", "minimum size of items to report (e.g. 10MB, 500KB)")
	minAge := flag.Int("min-age", 0, "minimum age of items in days to report")
	deleteFlag := flag.Bool("delete", false, "Delete found juck items")
	yesFlag := flag.Bool("yes", false, "skip confirmation prompt (Dangerous!)")
	jsonFlag := flag.Bool("json", false, "format output as JSON")

	flag.Parse()

	if *jsonFlag && *deleteFlag {
		fmt.Fprintln(os.Stderr, "Error: cannot use --json and --delete together")
		os.Exit(1)
	}

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
	// Automatically skip global cache if targeting a specific sub folder
	home, _ := os.UserHomeDir()
	effectiveSkip := *skip

	if root != "." && root != home {
		if effectiveSkip == "" {
			effectiveSkip = "lang-cache"
		} else if !strings.Contains(effectiveSkip, "lang-cache") {
			effectiveSkip += ",lang-cache"
		}
	}

	// totalItems and totalBytes accumulate counts across all scanners so we can
	// print a meaningful summary at the end. They live inside main() (not at the
	// package level) because they belong to a single scan run, not to the program.
	var totalItems int
	var totalBytes int64

	var allItems []scanner.Item
	var filteredItems []scanner.Item

	scanners := filterScanner(scanner.All(), *only, effectiveSkip)

	if !*jsonFlag {
		header(root, scanners)
	}


	for _, s := range scanners {

		if !*jsonFlag {
			fmt.Printf("\nRunning scanner: %s...\n", s.Name())
		}

		items, err := s.Scan(root)
		if err != nil {
			// A scanner error is non-fatal: report it and continue with the rest.
			if !*jsonFlag {
				fmt.Printf("Error running scanner %s: %v\n", s.Name(), err)
			} else {
				fmt.Fprintf(os.Stderr, "Error running scanner %s: %v\n", s.Name(), err)
			}
			continue
		}

		allItems = append(allItems, items...)	
	}

	if !*jsonFlag {
		fmt.Printf("\nScan Completed\n\n")
	}


	for _, item := range allItems {

		// check min-size
		if item.SizeBytes < minSizeBytes {
			continue
		}

		// check min-size
		if *minAge > 0 {
			minAgeDuration := time.Duration(*minAge) * 24 * time.Hour
			if !item.LastMod.IsZero() && time.Since(item.LastMod) < minAgeDuration {
				continue
			}
		}

		filteredItems = append(filteredItems, item)
	}

	sort.Slice(filteredItems, func(i, j int) bool {
		return filteredItems[i].SizeBytes > filteredItems[j].SizeBytes
	})

	if *jsonFlag {
		// Ensure a nil slice serialization to an empty JSON array '[]' instead of `null`
		outputItems := filteredItems

		if outputItems == nil {
			outputItems = []scanner.Item{}
		}

		jsonData, err := json.MarshalIndent(outputItems, "", " ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error serializing JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))
		return
	}

	for _, item := range filteredItems {
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


	// If the user didn't specify, complete the dry-run mode
	if !*deleteFlag {
		return
	}

	// Safety Check: If there are no items matching filters, donts asks for deltetion
	if len(filteredItems) == 0 {
		fmt.Println(("No Junk found to delete."))
		return
	}


	var itemsToDelete []scanner.Item
	deleteAllRemaining := false


	// Promt for confirmation unless --yess is passed
	if *yesFlag {
		// if yes passed, select all items
		itemsToDelete = filteredItems
	} else {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("\nInteractive Deletion Mode")
		fmt.Println("------------------------------")

		for i, item := range filteredItems {
			// If user chose `a` (all) previously, automatically select remaining items
			if deleteAllRemaining {
				itemsToDelete = append(itemsToDelete, item)
				continue
			}

			info := scanner.GetJunkInfo(item.Kind)
			fmt.Printf("\n[%d/%d] Path: %s\n", i+1, len(filteredItems), item.Path)
			fmt.Printf("	Kind: %s (%s)\n", item.Kind, formatSize(item.SizeBytes))
			fmt.Printf("	Description: %s\n", info.Description)

			if info.Warning != "" {
				fmt.Printf("	Warning:	%s\n", info.Warning)
			}

			for {
				fmt.Print("\n      Delete this item? [y (yes) / n (no) / a (all) / q (quit)]: ")
				input, err := reader.ReadString('\n')

				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
					os.Exit(1)
				}

				input = strings.TrimSpace(strings.ToLower(input))

				if input == "y" || input == "yes" {
					itemsToDelete = append(itemsToDelete, item)
					break
				} else if input == "n" || input == "no" || input == "" {
					fmt.Println("      Skipped.")
					break
				} else if input == "a" || input == "all" {
					deleteAllRemaining = true
					itemsToDelete = append(itemsToDelete, item)
					fmt.Println("      Selecting all remaining items for deletion.")
					break
				} else if input == "q" || input == "quit" {
					fmt.Println("\nInteractive deletion aborted.")
					if len(itemsToDelete) > 0 {
						fmt.Printf("Proceeding to delete the %d selected items...\n", len(itemsToDelete))
						deleteJunk(itemsToDelete)
					}
					return
				} else {
					fmt.Println("      Invalid option. Please enter y, n, a, or q.")
				}
			}
		}
	}

	if len(itemsToDelete) == 0 {
		fmt.Println("\nNo items selected for deletion")
		return
	}

	deleteJunk(itemsToDelete)
}
