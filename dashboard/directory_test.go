package dashboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestDirectorySelectorListsFoldersAndNavigates(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"Zulu", "alpha"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "ignored.txt"), []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}

	model := NewDirectoryModel(root)
	if len(model.entries) != 2 || model.entries[0] != "alpha" || model.entries[1] != "Zulu" {
		t.Fatalf("directory entries = %#v; want folders in case-insensitive order", model.entries)
	}
	next, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = next.(DirectoryModel)
	if model.path != filepath.Join(root, "alpha") {
		t.Fatalf("enter opened %q; want alpha", model.path)
	}
}

func TestDirectorySelectorRequiresExplicitSelection(t *testing.T) {
	root := t.TempDir()
	model := NewDirectoryModel(root)
	next, command := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeySpace}))
	model = next.(DirectoryModel)
	if model.selected != root || command == nil {
		t.Fatal("space did not explicitly select the current directory and exit")
	}

	cancelled := NewDirectoryModel(root)
	next, _ = cancelled.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if !next.(DirectoryModel).cancelled {
		t.Fatal("escape did not cancel directory selection")
	}
}

func TestDirectorySelectorViewExplainsControls(t *testing.T) {
	content := NewDirectoryModel(t.TempDir()).View().Content
	for _, want := range []string{"Directory selector", "Select this folder", "by Chamudu"} {
		if !strings.Contains(content, want) {
			t.Fatalf("selector did not contain %q: %q", want, content)
		}
	}
}
