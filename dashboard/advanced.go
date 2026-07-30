package dashboard

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

type advancedAction uint8

const (
	advancedNone advancedAction = iota
	advancedBrowseExclude
	advancedApply
	advancedCancel
)

type AdvancedModel struct {
	config  ScanSetup
	cursor  int
	editing bool
	input   string
	notice  string
	action  advancedAction
	width   int
}

func NewAdvancedModel(config ScanSetup) AdvancedModel {
	return AdvancedModel{config: cloneSetup(config)}
}
func (m AdvancedModel) Init() tea.Cmd { return nil }

func (m AdvancedModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := message.(tea.WindowSizeMsg); ok {
		m.width = size.Width
		return m, nil
	}
	key, ok := message.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	if m.editing {
		switch key.String() {
		case "enter":
			m.finishEdit()
		case "esc":
			m.editing, m.input = false, ""
		case "backspace":
			if len(m.input) > 0 {
				runes := []rune(m.input)
				m.input = string(runes[:len(runes)-1])
			}
		default:
			if key.Text != "" {
				m.input += key.Text
			}
		}
		return m, nil
	}
	maxCursor := 4 + len(m.config.Excludes)
	switch key.String() {
	case "q", "esc", "ctrl+c":
		m.action = advancedCancel
		return m, tea.Quit
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(maxCursor, m.cursor+1)
	case "x", "delete":
		m.removeFocusedExclude()
	case "enter":
		switch {
		case m.cursor <= 1:
			m.editing = true
			if m.cursor == 0 {
				m.input = m.config.MinSize
			} else if m.config.MinAge > 0 {
				m.input = strconv.Itoa(m.config.MinAge)
			}
		case m.cursor == 2:
			m.action = advancedBrowseExclude
			return m, tea.Quit
		case m.cursor == 3+len(m.config.Excludes):
			m.config.MinSize, m.config.MinAge, m.config.Excludes = "", 0, nil
			m.notice = "Advanced settings reset to defaults."
		case m.cursor == 4+len(m.config.Excludes):
			if err := validateAdvanced(m.config); err != nil {
				m.notice = err.Error()
				break
			}
			m.action = advancedApply
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *AdvancedModel) finishEdit() {
	value := strings.TrimSpace(m.input)
	if m.cursor == 0 {
		if value != "" {
			if _, err := parseSizeValue(value); err != nil {
				m.notice = err.Error()
				return
			}
		}
		m.config.MinSize = value
	} else {
		if value == "" {
			m.config.MinAge = 0
		} else {
			age, err := strconv.Atoi(value)
			if err != nil || age < 0 {
				m.notice = "Minimum age must be zero or a positive whole number."
				return
			}
			m.config.MinAge = age
		}
	}
	m.editing, m.input, m.notice = false, "", ""
}

func (m *AdvancedModel) removeFocusedExclude() {
	index := m.cursor - 3
	if index < 0 || index >= len(m.config.Excludes) {
		return
	}
	m.config.Excludes = append(m.config.Excludes[:index], m.config.Excludes[index+1:]...)
	m.cursor = min(m.cursor, 4+len(m.config.Excludes))
	m.notice = "Excluded path removed."
}

func validateAdvanced(config ScanSetup) error {
	if config.MinSize != "" {
		if _, err := parseSizeValue(config.MinSize); err != nil {
			return err
		}
	}
	if config.MinAge < 0 {
		return fmt.Errorf("minimum age cannot be negative")
	}
	return nil
}

func parseSizeValue(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	cut := len(value)
	for i, r := range value {
		if !unicode.IsDigit(r) && r != '.' {
			cut = i
			break
		}
	}
	number, unit := strings.TrimSpace(value[:cut]), strings.ToLower(strings.TrimSpace(value[cut:]))
	amount, err := strconv.ParseFloat(number, 64)
	if err != nil || amount < 0 {
		return 0, fmt.Errorf("Use a size such as 100MB or 1GB.")
	}
	multipliers := map[string]float64{"": 1, "b": 1, "k": 1024, "kb": 1024, "m": 1024 * 1024, "mb": 1024 * 1024, "g": 1024 * 1024 * 1024, "gb": 1024 * 1024 * 1024, "t": 1024 * 1024 * 1024 * 1024, "tb": 1024 * 1024 * 1024 * 1024}
	multiplier, ok := multipliers[unit]
	if !ok {
		return 0, fmt.Errorf("Unknown size unit %q; use B, KB, MB, GB, or TB.", unit)
	}
	return int64(amount * multiplier), nil
}

func cloneSetup(config ScanSetup) ScanSetup {
	enabled := config.Enabled
	config.Enabled = map[string]bool{}
	for key, value := range enabled {
		config.Enabled[key] = value
	}
	config.Excludes = append([]string(nil), config.Excludes...)
	return config
}

func (m AdvancedModel) View() tea.View {
	var view strings.Builder
	view.WriteString(titleStyle.Render("SWEEPR"))
	view.WriteString("  ")
	view.WriteString(renderBrandSubtitle("advanced scan settings"))
	view.WriteString("\n")
	view.WriteString(ruleStyle.Width(m.advancedWidth()).Render(""))
	view.WriteString("\n\n")
	view.WriteString(sectionTitleStyle.Render("Advanced settings"))
	view.WriteString("\n")
	view.WriteString(subtitleStyle.Render("Filters affect reported results only; exclusions prevent directory traversal."))
	view.WriteString("\n\n")
	size, age := defaultText(m.config.MinSize, "No minimum"), "No minimum"
	if m.config.MinAge > 0 {
		age = fmt.Sprintf("%d days", m.config.MinAge)
	}
	if m.editing && m.cursor == 0 {
		size = m.input + "▏"
	}
	if m.editing && m.cursor == 1 {
		age = m.input + "▏"
	}
	rows := []string{"Minimum size       " + size, "Minimum age        " + age, "Add excluded folder"}
	for _, path := range m.config.Excludes {
		rows = append(rows, "Exclude             "+truncateText(path, max(12, m.advancedWidth()-24)))
	}
	rows = append(rows, "Reset advanced settings", "Apply and go back")
	var panel strings.Builder
	for index, row := range rows {
		prefix, style := "  ", subtitleStyle
		if index == m.cursor {
			prefix, style = "› ", activeRowStyle
		}
		panel.WriteString(prefix + style.Render(row) + "\n")
	}
	view.WriteString(panelStyle(m.advancedWidth()).Render(strings.TrimSuffix(panel.String(), "\n")))
	view.WriteString("\n")
	if m.notice != "" {
		view.WriteString(warningStyle.Render(m.notice))
		view.WriteString("\n")
	}
	if m.editing {
		view.WriteString(renderShortcuts(shortcut{"Type", "Enter value"}, shortcut{"Enter", "Apply field"}, shortcut{"Esc", "Cancel edit"}))
	} else {
		view.WriteString(renderShortcuts(shortcut{"↑/↓", "Move"}, shortcut{"Enter", "Edit or open"}, shortcut{"X", "Remove exclusion"}, shortcut{"Esc", "Go back"}))
	}
	style := appStyle
	if m.width > 0 {
		style = style.MaxWidth(m.width)
	}
	result := tea.NewView(style.Render(view.String()))
	result.AltScreen = true
	result.WindowTitle = "sweepr advanced settings"
	return result
}

func defaultText(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
func (m AdvancedModel) advancedWidth() int {
	if m.width <= 0 {
		return 88
	}
	return max(30, min(100, m.width-6))
}

func RunAdvancedSetup(initial ScanSetup) (ScanSetup, bool, error) {
	config := cloneSetup(initial)
	for {
		final, err := tea.NewProgram(NewAdvancedModel(config)).Run()
		if err != nil {
			return initial, false, err
		}
		model, ok := final.(AdvancedModel)
		if !ok {
			return initial, false, fmt.Errorf("advanced setup returned unexpected model type %T", final)
		}
		config = model.config
		switch model.action {
		case advancedBrowseExclude:
			path, selected, err := ChooseDirectory(config.Root)
			if err != nil {
				return initial, false, err
			}
			if selected {
				clean := filepath.Clean(path)
				duplicate := false
				for _, existing := range config.Excludes {
					if existing == clean {
						duplicate = true
					}
				}
				if !duplicate {
					config.Excludes = append(config.Excludes, clean)
				}
			}
		case advancedApply:
			return config, true, nil
		default:
			return initial, false, nil
		}
	}
}
