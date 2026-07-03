package main

import (
	"fmt"
	"sweepr/scanner"
	"time"
	"math"
)

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	// O(1) calculation of the exponent index
	exp := int(math.Log2(float64(bytes)) / 10)
	
	// Edge case check to prevent array index out of bounds for extremely large values
	if exp > 6 { 
		exp = 6
	}

	div := math.Pow(unit, float64(exp))
	return fmt.Sprintf("%.2f %cB", float64(bytes)/div, "KMGTPE"[exp-1])
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	return t.Format("2006-01-02 15:04:05")
}

func main() {
	root := "." // hardcoded for testing

	var totalItems int
	var totalBytes int64

	scanners := scanner.All()

	fmt.Printf("Starting sweepr scan..\n")

	for _, s := range scanners {
		fmt.Printf("\nRunning scanner: %s...\n", s.Name())

		items, err := s.Scan(root)
		if err != nil {
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

	fmt.Printf("Total items: %-12d Total size: %-40s\n",  totalItems,formatSize(totalBytes))
	

}

