// Package remover defines resource-specific deletion behavior. Keeping removal
// separate from scanning lets scanners report resources without also owning
// confirmation prompts or CLI presentation.
package remover

import (
	"errors"
	"fmt"

	"sweepr/scanner"
)

// ErrUnsupported is returned when no registered remover owns an item's
// resource type.
var ErrUnsupported = errors.New("unsupported resource type")

// Remover deletes one or more resource types. Implementations must decide
// support without performing deletion so callers can filter confirmation lists.
type Remover interface {
	Supports(scanner.ResourceType) bool
	Remove(scanner.Item) error
}

// All returns every built-in removal strategy. More-specific removers can be
// added without expanding a central deletion switch in main.go.
func All() []Remover {
	return []Remover{
		&FilesystemRemover{},
		&DockerImageRemover{},
	}
}

// Find returns the remover responsible for an item, if one is registered.
func Find(item scanner.Item) (Remover, bool) {
	for _, candidate := range All() {
		if candidate.Supports(item.ResourceType) {
			return candidate, true
		}
	}
	return nil, false
}

// Supports reports whether a registered remover can safely handle the item.
func Supports(item scanner.Item) bool {
	_, ok := Find(item)
	return ok
}

// Remove delegates deletion to the strategy that owns the item's resource
// type. The unsupported check remains a final safety guard even when callers
// filter items with Supports before presenting confirmation prompts.
func Remove(item scanner.Item) error {
	strategy, ok := Find(item)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnsupported, item.ResourceType)
	}
	return strategy.Remove(item)
}
