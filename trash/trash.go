// Package trash moves filesystem resources to the operating system's native
// trash or recycle bin. It never falls back to permanent deletion.
package trash

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"sweepr/scanner"
)

var ErrUnsupported = errors.New("trash is unsupported for this resource")

type command struct {
	name string
	args []string
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
	output, err := exec.Command(cmd.name, cmd.args...).CombinedOutput()
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
tell application "Finder" to delete POSIX file (item 1 of argv)
end run`
		return command{name: "osascript", args: []string{"-e", script, item.Path}}, nil
	case "windows":
		const script = `Add-Type -AssemblyName Microsoft.VisualBasic; $p=$args[0]; $kind=$args[1]; if ($kind -eq 'directory') { [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteDirectory($p, 'OnlyErrorDialogs', 'SendToRecycleBin') } else { [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteFile($p, 'OnlyErrorDialogs', 'SendToRecycleBin') }`
		return command{
			name: "powershell.exe",
			args: []string{"-NoProfile", "-NonInteractive", "-Command", script, item.Path, string(item.ResourceType)},
		}, nil
	default:
		return command{}, fmt.Errorf("%w on %s", ErrUnsupported, goos)
	}
}
