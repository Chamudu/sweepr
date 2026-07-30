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

func TestWelcomeCanChooseVisibleExitAction(t *testing.T) {
	model := WelcomeModel{}
	next, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = next.(WelcomeModel)
	if model.choice != 1 {
		t.Fatalf("down selected choice %d; want exit choice 1", model.choice)
	}
	next, command := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = next.(WelcomeModel)
	if model.continued || command == nil {
		t.Fatal("confirming visible exit action continued or did not quit")
	}
}

func TestWelcomeExplainsSafetyAndCreator(t *testing.T) {
	content := (WelcomeModel{}).View().Content
	for _, want := range []string{"by Chamudu", "Scanning itself is read-only", "Permanent deletion cannot be undone", "Continue to scan setup", "Exit sweepr"} {
		if !strings.Contains(content, want) {
			t.Fatalf("welcome did not contain %q: %q", want, content)
		}
	}
}
