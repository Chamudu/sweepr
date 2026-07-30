package dashboard

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// WelcomeModel owns the first-launch safety dialogue. It is deliberately a
// separate model because no scan should begin until the user continues.
type WelcomeModel struct {
	width     int
	height    int
	continued bool
}

func (m WelcomeModel) Init() tea.Cmd { return nil }

func (m WelcomeModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = message.Width, message.Height
	case tea.KeyPressMsg:
		switch message.String() {
		case "enter":
			m.continued = true
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m WelcomeModel) View() tea.View {
	var view strings.Builder
	view.WriteString(titleStyle.Render("WELCOME TO SWEEPR"))
	view.WriteString("  ")
	view.WriteString(subtitleStyle.Render("by Chamudu"))
	view.WriteString("\n")
	view.WriteString(ruleStyle.Width(m.welcomeWidth()).Render(""))
	view.WriteString("\n\n")
	view.WriteString(sectionTitleStyle.Render("A safe start"))
	view.WriteString("\n\n")
	view.WriteString("Sweepr finds development files and caches that may be recreated.\n")
	view.WriteString("A result is a candidate—not an instruction to delete it.\n\n")
	view.WriteString(safeStyle.Render("✓ Scanning itself is read-only."))
	view.WriteString("\n")
	view.WriteString(infoStyle.Render("◉ Read only is the default cleanup mode."))
	view.WriteString("\n")
	view.WriteString(warningStyle.Render("! Safe trash keeps data until your OS trash is emptied."))
	view.WriteString("\n")
	view.WriteString(dangerStyle.Render("! Permanent deletion cannot be undone through sweepr."))
	view.WriteString("\n\n")
	view.WriteString("Global caches can affect builds outside the selected project.\n")
	view.WriteString("You will review exact paths before any cleanup action.\n\n")
	view.WriteString(helpStyle.Render("enter continue safely   esc/q exit without scanning"))

	style := appStyle
	if m.width > 0 {
		style = style.MaxWidth(m.width)
	}
	result := tea.NewView(style.Render(view.String()))
	result.AltScreen = true
	result.WindowTitle = "Welcome to sweepr"
	return result
}

func (m WelcomeModel) welcomeWidth() int {
	if m.width <= 0 {
		return 76
	}
	return max(30, min(76, m.width-6))
}

// RunWelcome blocks until the user continues or cancels. False means no scan
// should start; cancellation never changes files or preferences.
func RunWelcome() (bool, error) {
	final, err := tea.NewProgram(WelcomeModel{}).Run()
	if err != nil {
		return false, err
	}
	model, ok := final.(WelcomeModel)
	if !ok {
		return false, fmt.Errorf("welcome returned unexpected model type %T", final)
	}
	return model.continued, nil
}
