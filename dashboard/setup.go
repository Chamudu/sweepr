package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// ScanScope describes whether a scan targets project-relative or machine-wide
// resources. Keeping scope explicit prevents surprising global cache scans.
type ScanScope uint8

const (
	ScopeLocal ScanScope = iota
	ScopeGlobal
	ScopeCombined
)

var scannerNames = [...]string{"dev-junk", "os-junk", "lang-cache", "docker"}

// ScanSetup is the validated configuration returned to the scanner pipeline.
type ScanSetup struct {
	Root     string
	Scope    ScanScope
	Enabled  map[string]bool
	Advanced bool
}

func DefaultScanSetup(root string) ScanSetup {
	return ScanSetup{Root: root, Scope: ScopeLocal, Enabled: map[string]bool{
		"dev-junk": true, "os-junk": true,
	}}
}

// EnabledScannerNames returns stable CLI-compatible scanner identifiers.
func (s ScanSetup) EnabledScannerNames() []string {
	result := make([]string, 0, len(scannerNames))
	for _, name := range scannerNames {
		if s.Enabled[name] && scannerAllowed(s.Scope, name) {
			result = append(result, name)
		}
	}
	return result
}

type setupAction uint8

const (
	setupNone setupAction = iota
	setupBrowse
	setupStart
	setupExit
)

type SetupModel struct {
	config ScanSetup
	cursor int
	width  int
	height int
	notice string
	action setupAction
}

func NewSetupModel(config ScanSetup) SetupModel {
	if config.Enabled == nil {
		config.Enabled = make(map[string]bool)
	}
	return SetupModel{config: config}
}

func (m SetupModel) Init() tea.Cmd { return nil }

func (m SetupModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := message.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
		return m, nil
	}
	key, ok := message.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "q", "esc", "ctrl+c":
		m.action = setupExit
		return m, tea.Quit
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(8, m.cursor+1)
	case "left", "h":
		if m.cursor == 1 {
			m.changeScope(-1)
		}
	case "right", "l":
		if m.cursor == 1 {
			m.changeScope(1)
		}
	case "space":
		m.activate(false)
	case "enter":
		if m.activate(true) {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *SetupModel) activate(enter bool) bool {
	m.notice = ""
	switch {
	case m.cursor == 0 && enter:
		m.action = setupBrowse
		return true
	case m.cursor == 1:
		m.changeScope(1)
	case m.cursor >= 2 && m.cursor <= 5:
		name := scannerNames[m.cursor-2]
		if !scannerAllowed(m.config.Scope, name) {
			m.notice = scannerUnavailableReason(name)
			return false
		}
		m.config.Enabled[name] = !m.config.Enabled[name]
	case m.cursor == 6:
		m.notice = "Advanced filters will open here in the next implementation step."
	case m.cursor == 7 && enter:
		if err := validateSetup(m.config); err != nil {
			m.notice = err.Error()
			return false
		}
		m.action = setupStart
		return true
	case m.cursor == 8 && enter:
		m.action = setupExit
		return true
	}
	return false
}

func (m *SetupModel) changeScope(delta int) {
	next := int(m.config.Scope) + delta
	if next < int(ScopeLocal) {
		next = int(ScopeCombined)
	}
	if next > int(ScopeCombined) {
		next = int(ScopeLocal)
	}
	m.config.Scope = ScanScope(next)
	removed := []string{}
	for name, enabled := range m.config.Enabled {
		if enabled && !scannerAllowed(m.config.Scope, name) {
			m.config.Enabled[name] = false
			removed = append(removed, name)
		}
	}
	// Give a new scope useful safe defaults when it would otherwise be empty.
	if len(m.config.EnabledScannerNames()) == 0 {
		for _, name := range scannerNames {
			if scannerAllowed(m.config.Scope, name) {
				m.config.Enabled[name] = true
			}
		}
	}
	if len(removed) > 0 {
		m.notice = "Disabled incompatible scanners: " + strings.Join(removed, ", ")
	}
}

func scannerAllowed(scope ScanScope, name string) bool {
	project := name == "dev-junk" || name == "os-junk"
	switch scope {
	case ScopeLocal:
		return project
	case ScopeGlobal:
		return !project
	case ScopeCombined:
		return true
	default:
		return false
	}
}

func scannerUnavailableReason(name string) string {
	if name == "docker" {
		return "Docker images belong to the Docker engine, not the selected folder. Choose a global scope."
	}
	if name == "lang-cache" {
		return "Language caches are shared across projects. Choose a global scope to include them."
	}
	return "This scanner needs a selected-folder scope."
}

func validateSetup(config ScanSetup) error {
	info, err := os.Stat(config.Root)
	if err != nil {
		return fmt.Errorf("Target directory is unavailable: %v", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("Target must be a directory")
	}
	if len(config.EnabledScannerNames()) == 0 {
		return fmt.Errorf("Select at least one available scanner")
	}
	return nil
}

func scopeTitle(scope ScanScope) string {
	return [...]string{"Selected folder only", "Global resources only", "Folder + global resources"}[scope]
}

func scannerTitle(name string) string {
	return map[string]string{"dev-junk": "Developer junk", "os-junk": "OS junk", "lang-cache": "Language caches", "docker": "Docker images"}[name]
}

func (m SetupModel) View() tea.View {
	var view strings.Builder
	view.WriteString(titleStyle.Render("SWEEPR"))
	view.WriteString("  ")
	view.WriteString(renderBrandSubtitle("scan setup"))
	view.WriteString("\n")
	view.WriteString(ruleStyle.Width(m.setupWidth()).Render(""))
	view.WriteString("\n\n")
	view.WriteString(sectionTitleStyle.Render("Configure scan"))
	view.WriteString("\n")
	view.WriteString(subtitleStyle.Render("Choose exactly where sweepr may look. Scanning does not remove anything."))
	view.WriteString("\n\n")

	rows := []string{
		"Browse folder       " + truncateText(m.config.Root, max(12, m.setupWidth()-25)),
		"Scan scope          ◀ " + scopeTitle(m.config.Scope) + " ▶",
	}
	for _, name := range scannerNames {
		mark := "[ ]"
		if !scannerAllowed(m.config.Scope, name) {
			mark = "[-]"
		} else if m.config.Enabled[name] {
			mark = "[x]"
		}
		rows = append(rows, fmt.Sprintf("%-3s %s", mark, scannerTitle(name)))
	}
	rows = append(rows, "Advanced settings", "Start scan", "Exit sweepr")
	var panel strings.Builder
	for index, row := range rows {
		prefix := "  "
		style := lipglossStyleForSetupRow(index, m)
		if index == m.cursor {
			prefix = "› "
		}
		panel.WriteString(prefix + style.Render(row) + "\n")
	}
	view.WriteString(panelStyle(m.setupWidth()).Render(strings.TrimSuffix(panel.String(), "\n")))
	view.WriteString("\n")
	if m.notice != "" {
		view.WriteString(warningStyle.Render(m.notice))
		view.WriteString("\n")
	}
	view.WriteString(renderShortcuts(shortcut{"↑/↓", "Move"}, shortcut{"←/→", "Change scope"}, shortcut{"Space", "Toggle"}, shortcut{"Enter", "Open"}, shortcut{"Esc/Q", "Exit"}))

	style := appStyle
	if m.width > 0 {
		style = style.MaxWidth(m.width)
	}
	result := tea.NewView(style.Render(view.String()))
	result.AltScreen = true
	result.WindowTitle = "sweepr scan setup"
	return result
}

func lipglossStyleForSetupRow(index int, m SetupModel) interface{ Render(...string) string } {
	if index >= 2 && index <= 5 && !scannerAllowed(m.config.Scope, scannerNames[index-2]) {
		return disabledStyle
	}
	if index == m.cursor {
		return activeRowStyle
	}
	return subtitleStyle
}

func (m SetupModel) setupWidth() int {
	if m.width <= 0 {
		return 88
	}
	return max(30, min(100, m.width-6))
}

// RunScanSetup coordinates the setup page and reusable directory browser.
func RunScanSetup(initial ScanSetup) (ScanSetup, bool, error) {
	config := initial
	for {
		final, err := tea.NewProgram(NewSetupModel(config)).Run()
		if err != nil {
			return ScanSetup{}, false, err
		}
		model, ok := final.(SetupModel)
		if !ok {
			return ScanSetup{}, false, fmt.Errorf("scan setup returned unexpected model type %T", final)
		}
		config = model.config
		switch model.action {
		case setupBrowse:
			path, selected, err := ChooseDirectory(config.Root)
			if err != nil {
				return ScanSetup{}, false, err
			}
			if selected {
				config.Root = filepath.Clean(path)
			}
		case setupStart:
			return config, true, nil
		default:
			return ScanSetup{}, false, nil
		}
	}
}
