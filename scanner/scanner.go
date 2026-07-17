// Package scanner defines the plugin-style interface every "junk detector"
// implements, plus the shared Item type used to report what was found.
//
// See docs/ARCHITECTURE.md for the reasoning behind this shape.
package scanner

import "time"

// ResourceType identifies what an Item represents and therefore which removal
// mechanism, if any, is safe to use for it.
type ResourceType string

const (
	ResourceFile        ResourceType = "file"
	ResourceDirectory   ResourceType = "directory"
	ResourceDockerImage ResourceType = "docker-image"
)

// Item represents one reclaimable resource found by a scanner. Path is the
// filesystem location for file-backed items; newer non-filesystem scanners may
// temporarily use it as a stable resource identifier. DisplayName is optional
// user-facing text for resources whose identifier is not meaningful to people.
type Item struct {
	Path         string       `json:"path"`
	DisplayName  string       `json:"display_name,omitempty"`
	Kind         string       `json:"kind"`
	SizeBytes    int64        `json:"size_bytes"`
	LastMod      time.Time    `json:"last_mod"`
	ResourceType ResourceType `json:"resource_type"`
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
		&DockerScanner{},
	}
}

// JunkInfo holds descriptive explanation and safety warnings for a given junk kind.
type JunkInfo struct {
	Description string `json:"description"`
	Warning     string `json:"warning"`
}

// GetJunkInfo returns the Description and Warning for a specific junk kind label.
func GetJunkInfo(kind string) JunkInfo {
	switch kind {
	case "node_modules":
		return JunkInfo{
			Description: "Node.js external package dependencies directory.",
			Warning:     "Requires running 'npm install' or equivalent to restore dependencies.",
		}
	case "dist", "build":
		return JunkInfo{
			Description: "Generic compiler build and distribution output folder.",
			Warning:     "Recompilation may take longer next time you run a build.",
		}
	case "next-cache":
		return JunkInfo{
			Description: "Next.js compiler and build cache.",
			Warning:     "Safe to delete. First startup/rebuild will compile from scratch.",
		}
	case "rust-target":
		return JunkInfo{
			Description: "Rust cargo compilation artifacts folder.",
			Warning:     "Requires compiling the project from scratch ('cargo build') next time.",
		}
	case "python-cache":
		return JunkInfo{
			Description: "CPython compiled bytecode cache directories.",
			Warning:     "Safe to delete. Python automatically regenerates these when running code.",
		}
	case "python-venv":
		return JunkInfo{
			Description: "Python virtual environment containing dependencies.",
			Warning:     "Deletes installed packages. You will need to recreate it and run 'pip install'.",
		}
	case "pytest-cache":
		return JunkInfo{
			Description: "Pytest test result and state cache.",
			Warning:     "Safe to delete. Pytest will regenerate this on the next run.",
		}
	case "poetry-cache":
		return JunkInfo{
			Description: "Poetry dependency management global downloads.",
			Warning:     "Subsequent installs will fetch packages from the internet.",
		}
	case "os-junk":
		return JunkInfo{
			Description: "Operating system desktop and folder layout files (e.g. .DS_Store, Thumbs.db).",
			Warning:     "Safe to delete. Discards local visual preferences (icon placement, scroll positions).",
		}
	case "npm-cache", "pip-cache", "cargo-cache", "yarn-cache", "pnpm-cache", "cocoapods-cache", "php-composer-cache", "bower-cache":
		return JunkInfo{
			Description: "Global package manager cache.",
			Warning:     "Deletes downloaded copies of packages. Next install will download from the internet.",
		}
	case "xcode-derived-data", "xcode-archives":
		return JunkInfo{
			Description: "macOS Developer build cache and archive history.",
			Warning:     "Deletes cached indexes and old production builds. Rebuilds will take longer.",
		}
	case "gradle-cache":
		return JunkInfo{
			Description: "Android gradle compilation and dependency cache.",
			Warning:     "First build of projects will take longer as gradle downloads dependencies.",
		}
	case "android-emulator-snapshots":
		return JunkInfo{
			Description: "Android Virtual Device saved boot states.",
			Warning:     "The emulator will perform a slow cold-boot next time it starts.",
		}
	case "docker-image":
		return JunkInfo{
			Description: "Dangling Docker image no longer referenced by a repository tag.",
			Warning:     "Report only: Docker image deletion is not implemented yet.",
		}
	case "clangd-index-cache", "deno-cache", "zig-cache", "supabase-local-dev", "hardhat-evm-cache":
		return JunkInfo{
			Description: "Compiler or framework local development storage.",
			Warning:     "Subsequent launches will re-download packages or re-index files.",
		}
	case "nuxt-cache", "sveltekit-cache", "docusaurus-cache", "turborepo-cache", "jupyter-snapshots", "tox-virtualenv", "mypy-type-cache", "python-coverage", "zig-local-cache", "cmake-debug", "cmake-release", "pnpm-local-store", "terraform-plugins", "serverless-framework", "vagrant-vm-state":
		return JunkInfo{
			Description: "Framework, environment, or IaC tracking files.",
			Warning:     "Recompilation, cloud state syncs, or VM boots will take longer next time.",
		}
	default:
		return JunkInfo{
			Description: "Disposable cache or artifact folder.",
			Warning:     "Recreation or redownload may occur on next run.",
		}
	}
}
