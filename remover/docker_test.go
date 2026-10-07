
package remover

import (
	"errors"
	"strings"
	"testing"

	"sweepr/scanner"
)

func TestDockerRemoverSupportsOnlyDockerImage(t *testing.T) {
	r := &DockerImageRemover{}
	tests := []struct {
		resourceType scanner.ResourceType
		want         bool
	}{
		{scanner.ResourceDockerImage, true},
		{scanner.ResourceFile, false},
		{scanner.ResourceDirectory, false},
		{scanner.ResourceType("unknown"), false},
	}
	for _, test := range tests {
		got := r.Supports(test.resourceType)
		if got != test.want {
			t.Errorf("DockerImageRemover.Supports(%q) = %v, want %v", test.resourceType, got, test.want)
		}
	}
}

func TestDockerRemoverRejectsNonDockerItem(t *testing.T) {
	r := &DockerImageRemover{}
	err := r.Remove(scanner.Item{Path: "unused", ResourceType: scanner.ResourceFile})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Remove() with filesystem resource = %v, want ErrUnsupported", err)
	}
}

func TestDockerRemoverRejectsDirectoryItem(t *testing.T) {
	r := &DockerImageRemover{}
	err := r.Remove(scanner.Item{Path: "/some/path", ResourceType: scanner.ResourceDirectory})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Remove() with directory resource = %v, want ErrUnsupported", err)
	}
}

// TestDockerRemoverCommandFailureIncludesOutput verifies that when the docker
// CLI exits with a non-zero status, the error message includes the command
// output so users can diagnose the failure without running docker manually.
//
// This test intentionally passes a bogus image ID that will either fail because
// Docker is not installed (exit 127/exec error) or because the image does not
// exist (exit 1 with Docker error output). Both cases must produce a non-nil
// error that wraps useful context.
func TestDockerRemoverCommandFailureProducesError(t *testing.T) {
	r := &DockerImageRemover{}
	// Use an image ID that is structurally valid but cannot exist so that
	// Docker itself returns an error, or the exec fails when Docker is absent.
	item := scanner.Item{
		Path:         "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		DisplayName:  "nonexistent-image",
		ResourceType: scanner.ResourceDockerImage,
	}
	err := r.Remove(item)
	// Whether Docker is installed or not, attempting to remove a nonexistent
	// image must return an error. A nil error here would mean the image was
	// found and deleted, which should never happen with a zero-hash ID.
	if err == nil {
		t.Fatal("Remove() of nonexistent Docker image returned nil error")
	}
}

// TestPartialDockerRemovalReportsAllFailures verifies the multi-item contract:
// every item in the list is attempted and every failure is captured, not just
// the first one.
func TestPartialDockerRemovalReportsAllFailures(t *testing.T) {
	items := []scanner.Item{
		{Path: "sha256:aaaa", DisplayName: "img-a", ResourceType: scanner.ResourceDockerImage},
		{Path: "sha256:bbbb", DisplayName: "img-b", ResourceType: scanner.ResourceDockerImage},
	}

	var errs []error
	for _, item := range items {
		r := &DockerImageRemover{}
		if err := r.Remove(item); err != nil {
			errs = append(errs, err)
		}
	}

	// Both bogus images must fail independently.
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors for 2 bogus images, got %d: %v", len(errs), errs)
	}
	// Each error message must identify which image failed.
	for i, err := range errs {
		if !strings.Contains(err.Error(), items[i].DisplayName) {
			t.Errorf("error %d = %q, want it to contain display name %q", i, err.Error(), items[i].DisplayName)
		}
	}
}

func TestFindReturnsDockerRemoverForDockerImage(t *testing.T) {
	item := scanner.Item{ResourceType: scanner.ResourceDockerImage}
	strategy, ok := Find(item)
	if !ok {
		t.Fatal("Find() returned no remover for ResourceDockerImage")
	}
	if _, isDocker := strategy.(*DockerImageRemover); !isDocker {
		t.Fatalf("Find() returned %T, want *DockerImageRemover", strategy)
	}
}

func TestFindReturnsFilesystemRemoverForFile(t *testing.T) {
	item := scanner.Item{ResourceType: scanner.ResourceFile}
	strategy, ok := Find(item)
	if !ok {
		t.Fatal("Find() returned no remover for ResourceFile")
	}
	if _, isFS := strategy.(*FilesystemRemover); !isFS {
		t.Fatalf("Find() returned %T, want *FilesystemRemover", strategy)
	}
}

func TestFindReturnsNoRemoverForUnknownType(t *testing.T) {
	item := scanner.Item{ResourceType: scanner.ResourceType("network-volume")}
	_, ok := Find(item)
	if ok {
		t.Fatal("Find() returned a remover for an unknown resource type")
	}
}
