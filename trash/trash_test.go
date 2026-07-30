package trash

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"sweepr/scanner"
)

func TestCommandForKeepsPathInSeparateArgument(t *testing.T) {
	item := scanner.Item{
		Path:         `/tmp/cache with spaces;$(touch unsafe)`,
		ResourceType: scanner.ResourceDirectory,
	}

	tests := []struct {
		goos string
		name string
	}{
		{goos: "linux", name: "gio"},
		{goos: "darwin", name: "osascript"},
		{goos: "windows", name: "powershell.exe"},
	}
	for _, test := range tests {
		t.Run(test.goos, func(t *testing.T) {
			cmd, err := commandFor(test.goos, item)
			if err != nil {
				t.Fatal(err)
			}
			if cmd.name != test.name {
				t.Fatalf("command name = %q; want %q", cmd.name, test.name)
			}
			pathIsArgument := slices.Contains(cmd.args, item.Path)
			pathIsEnvironment := slices.Contains(cmd.env, "SWEEPR_TRASH_PATH="+item.Path)
			if !pathIsArgument && !pathIsEnvironment {
				t.Fatalf("path was not passed separately from program text: args=%#v env=%#v", cmd.args, cmd.env)
			}
		})
	}
}

func TestNativeTrashIntegration(t *testing.T) {
	if os.Getenv("SWEEPR_TRASH_INTEGRATION") != "1" {
		t.Skip("set SWEEPR_TRASH_INTEGRATION=1 to modify the current user's trash")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	scratch, err := os.MkdirTemp(home, ".sweepr-trash-integration-")
	if err != nil {
		t.Fatal(err)
	}
	// Cleanup is only a fallback when trashing fails. On success, the original
	// path no longer exists and the ephemeral CI runner later discards its trash.
	t.Cleanup(func() { _ = os.RemoveAll(scratch) })

	marker := filepath.Join(scratch, "marker.txt")
	if err := os.WriteFile(marker, []byte("sweepr trash integration test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	item := scanner.Item{Path: scratch, ResourceType: scanner.ResourceDirectory}
	if err := Move(item); err != nil {
		t.Fatalf("native trash operation failed: %v", err)
	}
	if _, err := os.Stat(scratch); !os.IsNotExist(err) {
		t.Fatalf("scratch path still exists after trash operation: %v", err)
	}
}

func TestTrashRejectsDockerAndUnknownOperatingSystems(t *testing.T) {
	docker := scanner.Item{ResourceType: scanner.ResourceDockerImage}
	if Supports(docker) {
		t.Fatal("Docker image must not be supported by filesystem trash")
	}

	_, err := commandFor("plan9", scanner.Item{ResourceType: scanner.ResourceFile})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("unsupported OS error = %v; want ErrUnsupported", err)
	}
}
