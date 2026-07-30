package remover

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"sweepr/scanner"
)

func TestFilesystemRemoverRemovesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "junk.tmp")
	if err := os.WriteFile(path, []byte("test data"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := Remove(scanner.Item{Path: path, ResourceType: scanner.ResourceFile})
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("removed file still exists or returned unexpected error: %v", err)
	}
}

func TestFilesystemRemoverRemovesDirectoryTree(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node_modules")
	if err := os.MkdirAll(filepath.Join(path, "package"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "package", "index.js"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	err := Remove(scanner.Item{Path: path, ResourceType: scanner.ResourceDirectory})
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("removed directory still exists or returned unexpected error: %v", err)
	}
}

func TestRemoveRejectsUnsupportedResource(t *testing.T) {
	err := Remove(scanner.Item{Path: "unused", ResourceType: scanner.ResourceType("unknown")})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Remove() error = %v, want ErrUnsupported", err)
	}
}

func TestRegistrySupportsKnownResourceTypes(t *testing.T) {
	tests := []struct {
		resourceType scanner.ResourceType
		want         bool
	}{
		{scanner.ResourceFile, true},
		{scanner.ResourceDirectory, true},
		{scanner.ResourceDockerImage, true},
		{scanner.ResourceType("unknown"), false},
	}

	for _, test := range tests {
		if got := Supports(scanner.Item{ResourceType: test.resourceType}); got != test.want {
			t.Errorf("Supports(%q) = %v, want %v", test.resourceType, got, test.want)
		}
	}
}
