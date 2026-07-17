package remover

import (
	"fmt"
	"os"

	"sweepr/scanner"
)

// FilesystemRemover deletes ordinary files and directories through the Go
// standard library.
type FilesystemRemover struct{}

func (r *FilesystemRemover) Supports(resourceType scanner.ResourceType) bool {
	return resourceType == scanner.ResourceFile || resourceType == scanner.ResourceDirectory
}

func (r *FilesystemRemover) Remove(item scanner.Item) error {
	switch item.ResourceType {
	case scanner.ResourceFile:
		return os.Remove(item.Path)
	case scanner.ResourceDirectory:
		return os.RemoveAll(item.Path)
	default:
		return fmt.Errorf("%w: %q", ErrUnsupported, item.ResourceType)
	}
}
