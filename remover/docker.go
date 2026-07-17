package remover

import (
	"fmt"
	"os/exec"
	"strings"

	"sweepr/scanner"
)

// DockerImageRemover removes images through Docker instead of treating their
// stable IDs as filesystem paths.
type DockerImageRemover struct{}

func (r *DockerImageRemover) Supports(resourceType scanner.ResourceType) bool {
	return resourceType == scanner.ResourceDockerImage
}

func (r *DockerImageRemover) Remove(item scanner.Item) error {
	if item.ResourceType != scanner.ResourceDockerImage {
		return fmt.Errorf("%w: %q", ErrUnsupported, item.ResourceType)
	}

	output, err := exec.Command("docker", "image", "rm", item.Path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove Docker image %s: %w: %s",
			item.DisplayName, err, strings.TrimSpace(string(output)))
	}
	return nil
}
