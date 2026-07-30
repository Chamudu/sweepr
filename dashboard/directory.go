package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// DirectoryModel is a terminal-native folder browser. It lists directories
// only because files can never be valid scan roots.
type DirectoryModel struct {
	path      string
	entries   []string
	cursor    int
	offset    int
	width     int
	height    int
	selected  string
	cancelled bool
	err       error
}

func NewDirectoryModel(initial string) DirectoryModel {
	abs, err := filepath.Abs(initial)
	if err != nil {
		return DirectoryModel{path: initial, err: err}
	}
	model := DirectoryModel{path: filepath.Clean(abs)}
	model.reload()
	return model
}

func (m DirectoryModel) Init() tea.Cmd { return nil }

func (m DirectoryModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := message.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
		m.keepVisible()
		return m, nil
	}
	key, ok := message.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "q", "esc", "ctrl+c":
		m.cancelled = true
		return m, tea.Quit
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(max(0, len(m.entries)-1), m.cursor+1)
	case "pgup":
		m.cursor = max(0, m.cursor-m.visibleRows())
	case "pgdown":
		m.cursor = min(max(0, len(m.entries)-1), m.cursor+m.visibleRows())
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = max(0, len(m.entries)-1)
	case "backspace", "left", "h":
		m.open(filepath.Dir(m.path))
	case "~":
		if home, err := os.UserHomeDir(); err == nil {
			m.open(home)
		} else {
			m.err = err
		}
	case "enter", "right", "l":
		if len(m.entries) > 0 {
			m.open(filepath.Join(m.path, m.entries[m.cursor]))
		}
	case "s", "space":
		if m.err == nil {
			m.selected = m.path
			return m, tea.Quit
		}
	}
	m.keepVisible()
	return m, nil
}

func (m *DirectoryModel) open(path string) {
	previous := m.path
	m.path = filepath.Clean(path)
	if !m.reload() {
		m.path = previous
	}
}

func (m *DirectoryModel) reload() bool {
	entries, err := os.ReadDir(m.path)
	if err != nil {
		m.err = err
		return false
	}
	m.err = nil
	m.entries = m.entries[:0]
	for _, entry := range entries {
		// Do not follow directory symlinks from the browser. This matches the
		// scanner's symlink safety and avoids surprising jumps or loops.
		if entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			m.entries = append(m.entries, entry.Name())
		}
	}
	sort.Slice(m.entries, func(i, j int) bool {
		return strings.ToLower(m.entries[i]) < strings.ToLower(m.entries[j])
	})
	m.cursor, m.offset = 0, 0
	return true
}

func (m DirectoryModel) visibleRows() int {
	if m.height <= 0 {
		return 14
	}
	return max(3, m.height-13)
}

func (m *DirectoryModel) keepVisible() {
	rows := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
	m.offset = min(m.offset, max(0, len(m.entries)-rows))
}

func (m DirectoryModel) View() tea.View {
	var view strings.Builder
	view.WriteString(titleStyle.Render("SWEEPR"))
	view.WriteString("  ")
	view.WriteString(renderBrandSubtitle("choose a scan directory"))
	view.WriteString("\n")
	view.WriteString(ruleStyle.Width(m.directoryWidth()).Render(""))
	view.WriteString("\n\n")
	view.WriteString(sectionTitleStyle.Render("Directory selector"))
	view.WriteString("\n")
	view.WriteString(infoStyle.Render(truncateText(m.path, m.directoryWidth())))
	view.WriteString("\n\n")

	var rows strings.Builder
	start := min(m.offset, len(m.entries))
	end := min(len(m.entries), start+m.visibleRows())
	if len(m.entries) == 0 {
		rows.WriteString(emptyStyle.Render("This directory has no subdirectories."))
	}
	for index := start; index < end; index++ {
		prefix := "  📁 "
		line := prefix + m.entries[index]
		if index == m.cursor {
			line = activeRowStyle.Render("› 📁 " + m.entries[index])
		}
		rows.WriteString(line)
		rows.WriteString("\n")
	}
	if len(m.entries) > m.visibleRows() {
		fmt.Fprintf(&rows, "%s\n", scrollStyle.Render(fmt.Sprintf("Showing %d–%d of %d folders", start+1, end, len(m.entries))))
	}
	view.WriteString(panelStyle(m.directoryWidth()).Render(strings.TrimSuffix(rows.String(), "\n")))
	view.WriteString("\n")
	if m.err != nil {
		view.WriteString(dangerStyle.Render("Cannot open folder: " + m.err.Error()))
		view.WriteString("\n")
	}
	view.WriteString(renderShortcuts(
		shortcut{"Enter/→", "Open folder"}, shortcut{"←/Backspace", "Parent folder"},
		shortcut{"~", "Home"}, shortcut{"Space/S", "Select this folder"},
		shortcut{"Esc", "Cancel"},
	))

	style := appStyle
	if m.width > 0 {
		style = style.MaxWidth(m.width)
	}
	result := tea.NewView(style.Render(view.String()))
	result.AltScreen = true
	result.WindowTitle = "sweepr directory selector"
	return result
}

func (m DirectoryModel) directoryWidth() int {
	if m.width <= 0 {
		return 88
	}
	return max(30, min(100, m.width-6))
}

// ChooseDirectory opens the terminal browser. The boolean is false when the
// user cancels, allowing callers to return to scan setup without starting work.
func ChooseDirectory(initial string) (string, bool, error) {
	final, err := tea.NewProgram(NewDirectoryModel(initial)).Run()
	if err != nil {
		return "", false, err
	}
	model, ok := final.(DirectoryModel)
	if !ok {
		return "", false, fmt.Errorf("directory selector returned unexpected model type %T", final)
	}
	if model.cancelled || model.selected == "" {
		return "", false, nil
	}
	return model.selected, true, nil
}
