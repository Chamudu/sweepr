// Package scanner defines the plugin-style interface every "junk detector"
// implements, plus the shared Item type used to report what was found.
//
// See docs/ARCHITECTURE.md for the reasoning behind this shape.
package scanner

import "time"

// Item represents one piece of reclaimable disk space: a directory
// (node_modules, __pycache__, ~/.npm) or a single junk file (.DS_Store).
type Item struct {
	Path      string	`json:"path"`
	Kind      string	`json:"kind"`
	SizeBytes int64		`json:"size_bytes"`
	LastMod   time.Time	`json:"last_mod"`
	IsDir     bool		`json:"is_dir"`
}

// Scanner is the interface every junk-finder implements.
type Scanner interface {
	// Name returns a short identifier used in output and --only/--skip filters.
	Name() string

	// Scan walks root (or fixed system paths, for scanners like LangCacheScanner
	// that aren't project-relative) and returns candidate Items.
	Scan(root string) ([]Item, error)
}

// All returns every built-in scanner.
func All() []Scanner {
	return []Scanner{
		&DevJunkScanner{},
		&LangCacheScanner{},
		&OSJunkScanner{},
	}
}
