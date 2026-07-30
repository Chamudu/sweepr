// Package trash moves filesystem resources to the operating system's native
// trash or recycle bin. It never falls back to permanent deletion.
package trash

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"sweepr/scanner"
)

var ErrUnsupported = errors.New("trash is unsupported for this resource")

type command struct {
	name string
	args []string
	env  []string
}

// Supports reports whether an item is meaningful to an OS trash service.
// Docker images live in a daemon, not a filesystem, so they cannot be trashed.
func Supports(item scanner.Item) bool {
	return item.ResourceType == scanner.ResourceFile ||
		item.ResourceType == scanner.ResourceDirectory
}

// Move sends one filesystem item to the native trash implementation for the
// current OS. A missing native command returns an error and leaves the item in
// place; there is deliberately no os.Remove fallback.
func Move(item scanner.Item) error {
	if !Supports(item) {
		return fmt.Errorf("%w: %q", ErrUnsupported, item.ResourceType)
	}

	cmd, err := commandFor(runtime.GOOS, item)
	if err != nil {
		return err
	}
	process := exec.Command(cmd.name, cmd.args...)
	if len(cmd.env) > 0 {
		process.Env = append(os.Environ(), cmd.env...)
	}
	output, err := process.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return fmt.Errorf("move %q to trash using %s: %w", item.Path, cmd.name, err)
		}
		return fmt.Errorf("move %q to trash using %s: %w: %s", item.Path, cmd.name, err, message)
	}
	return nil
}

func commandFor(goos string, item scanner.Item) (command, error) {
	switch goos {
	case "linux":
		return command{name: "gio", args: []string{"trash", item.Path}}, nil
	case "darwin":
		const script = `on run argv
set targetItem to POSIX file (item 1 of argv) as alias
tell application "Finder" to delete targetItem
end run`
		return command{name: "osascript", args: []string{"-e", script, item.Path}}, nil
	case "windows":
		const script = `Add-Type -AssemblyName Microsoft.VisualBasic; $p=$env:SWEEPR_TRASH_PATH; $kind=$env:SWEEPR_TRASH_KIND; if ($kind -eq 'directory') { [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteDirectory($p, 'OnlyErrorDialogs', 'SendToRecycleBin') } else { [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteFile($p, 'OnlyErrorDialogs', 'SendToRecycleBin') }`
		return command{
			name: "powershell.exe",
			args: []string{"-NoProfile", "-NonInteractive", "-Command", script},
			env: []string{
				"SWEEPR_TRASH_PATH=" + item.Path,
				"SWEEPR_TRASH_KIND=" + string(item.ResourceType),
			},
		}, nil
	default:
		return command{}, fmt.Errorf("%w on %s", ErrUnsupported, goos)
	}
}
