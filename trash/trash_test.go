package trash

import (
	"errors"
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
			if !slices.Contains(cmd.args, item.Path) {
				t.Fatalf("path was not passed as a separate argument: %#v", cmd.args)
			}
		})
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
