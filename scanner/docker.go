package scanner

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// dockerInspectResult mirrors only the fields we consume from
// `docker image inspect`. Keeping this boundary type small prevents Docker's
// much larger JSON response from leaking into the rest of the scanner package.
type dockerInspectResult struct {
	ID       string    `json:"Id"`
	RepoTags []string  `json:"RepoTags"`
	Size     int64     `json:"Size"`
	Created  time.Time `json:"Created"`
}

// dockerImage is sweepr's internal representation of an inspected image. It
// separates the stable ID used by Docker from the friendly name shown to users.
type dockerImage struct {
	ID          string
	DisplayName string
	SizeBytes   int64
	CreatedAt   time.Time
}

// DockerScanner finds dangling Docker images. Unlike the filesystem scanners,
// it queries the Docker daemon and does not walk the requested root directory.
type DockerScanner struct{}

// Compile-time assertion that DockerScanner implements Scanner. This does not
// allocate or run a scanner; it only makes a missing method a compiler error.
var _ Scanner = (*DockerScanner)(nil)

// Name returns the identifier accepted by the --only and --skip flags.
func (s *DockerScanner) Name() string {
	return "docker"
}

// Scan finds dangling Docker images. Docker resources are global, so the
// filesystem root required by the Scanner interface is intentionally unused.
func (s *DockerScanner) Scan(_ string, _ ScanOptions) ([]Item, error) {
	// Docker is a stretch integration, so its executable is optional. A machine
	// without Docker should still be able to run every filesystem scanner.
	if _, err := exec.LookPath("docker"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return []Item{}, nil
		}
		return nil, fmt.Errorf("locate Docker executable: %w", err)
	}

	ids, err := listDanglingImageIDs()
	if err != nil {
		return nil, err
	}

	results, err := inspectDockerImages(ids)
	if err != nil {
		return nil, err
	}

	// One inspection result becomes one Item, so reserve enough capacity up
	// front to avoid growing and reallocating the slice during append.
	items := make([]Item, 0, len(results))
	for _, result := range results {
		image := dockerImageFromInspect(result)
		items = append(items, Item{
			// Path temporarily carries the Docker ID because Item was originally
			// filesystem-only. ResourceType prevents it from reaching filesystem removal.
			Path:         image.ID,
			DisplayName:  image.DisplayName,
			Kind:         "docker-image",
			SizeBytes:    image.SizeBytes,
			LastMod:      image.CreatedAt,
			ResourceType: ResourceDockerImage,
		})
	}

	return items, nil
}

// listDanglingImageIDs asks Docker for untagged image IDs. Empty command output
// is a successful scan with no findings, not an error.
func listDanglingImageIDs() ([]string, error) {
	cmd := exec.Command(
		"docker", "image", "ls", "--filter", "dangling=true", "--quiet",
	)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return nil, fmt.Errorf(
			"docker image listing failed: %w: %s", err, strings.TrimSpace(string(output)),
		)
	}

	ids := strings.Fields(string(output))

	return ids, nil
}

// inspectDockerImages fetches exact byte sizes and creation times for all IDs
// in one Docker invocation. Structured inspect JSON is more reliable than
// parsing the human-oriented columns printed by `docker image ls`.
func inspectDockerImages(ids []string) ([]dockerInspectResult, error) {
	// Avoid invoking `docker image inspect` without a required image argument.
	if len(ids) == 0 {
		return []dockerInspectResult{}, nil
	}

	args := []string{"image", "inspect"}
	args = append(args, ids...)
	cmd := exec.Command("docker", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"docker image inspection failed: %w: %s", err, strings.TrimSpace(string(output)),
		)
	}

	var results []dockerInspectResult
	err = json.Unmarshal(output, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to parse docker inspect output: %w", err)
	}

	return results, nil
}

// shortDockerID returns the familiar 12-character form used in Docker output.
// The length check prevents a slice-bounds panic for unexpected short input.
func shortDockerID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// dockerImageFromInspect converts Docker's external JSON model into the small
// internal model used to create scanner Items.
func dockerImageFromInspect(result dockerInspectResult) dockerImage {
	// Dangling images normally have no repository tags, so start with a useful
	// fallback and replace it only when Docker supplies one or more tags.
	displayName := "<untagged>"
	if len(result.RepoTags) > 0 {
		displayName = strings.Join(result.RepoTags, ", ")
	}

	// Include a short ID so multiple untagged images remain distinguishable.
	displayName = fmt.Sprintf("%s (%s)", displayName, shortDockerID(result.ID))

	return dockerImage{
		ID:          result.ID,
		DisplayName: displayName,
		SizeBytes:   result.Size,
		CreatedAt:   result.Created,
	}
}
