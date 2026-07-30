package dashboard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestWelcomeRequiresExplicitContinue(t *testing.T) {
	model := WelcomeModel{}
	next, command := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = next.(WelcomeModel)
	if !model.continued || command == nil {
		t.Fatal("enter did not continue and request program exit")
	}

	cancelled := WelcomeModel{}
	next, _ = cancelled.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if next.(WelcomeModel).continued {
		t.Fatal("escape incorrectly acknowledged the welcome")
	}
}

func TestWelcomeExplainsSafetyAndCreator(t *testing.T) {
	content := (WelcomeModel{}).View().Content
	for _, want := range []string{"by Chamudu", "Scanning itself is read-only", "Permanent deletion cannot be undone"} {
		if !strings.Contains(content, want) {
			t.Fatalf("welcome did not contain %q: %q", want, content)
		}
	}
}
