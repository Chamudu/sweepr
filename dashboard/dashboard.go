// Package dashboard provides sweepr's interactive terminal interface.
package dashboard

import (
	"fmt"
	"math"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"sweepr/scanner"
)

// Model contains all state needed to render and update the dashboard. Keeping
// state in a plain Go value makes keyboard behavior testable without starting
// a real terminal.
type Model struct {
	items        []scanner.Item
	cursor       int
	selected     map[int]struct{}
	screen       screen
	confirmed    bool
	mode         Mode
	modeCursor   int
	width        int
	height       int
	offset       int
	reviewOffset int
}

// Mode describes the action the user chose for selected resources.
type Mode string

const (
	ModeReadOnly  Mode = "read-only"
	ModeTrash     Mode = "trash"
	ModePermanent Mode = "permanent-delete"
)

var modes = [...]Mode{ModeReadOnly, ModeTrash, ModePermanent}

var (
	accentColor = lipgloss.Color("99")
	cyanColor   = lipgloss.Color("86")
	greenColor  = lipgloss.Color("42")
	yellowColor = lipgloss.Color("214")
	redColor    = lipgloss.Color("203")
	mutedColor  = lipgloss.Color("245")
	borderColor = lipgloss.Color("238")

	appStyle          = lipgloss.NewStyle().Padding(1, 2)
	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(accentColor)
	subtitleStyle     = lipgloss.NewStyle().Foreground(mutedColor)
	sectionTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(cyanColor)
	ruleStyle         = lipgloss.NewStyle().BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(borderColor)
	warningStyle      = lipgloss.NewStyle().Foreground(yellowColor).Bold(true)
	helpStyle         = lipgloss.NewStyle().Foreground(mutedColor)
	emptyStyle        = lipgloss.NewStyle().Foreground(greenColor).Padding(1, 0)
	activeRowStyle    = lipgloss.NewStyle().Foreground(cyanColor).Bold(true)
	selectedStyle     = lipgloss.NewStyle().Foreground(greenColor)
	disabledStyle     = lipgloss.NewStyle().Foreground(mutedColor).Faint(true)
	scrollStyle       = lipgloss.NewStyle().Foreground(mutedColor).Italic(true)
	statusStrongStyle = lipgloss.NewStyle().Bold(true)
	dangerStyle       = lipgloss.NewStyle().Foreground(redColor).Bold(true)
	safeStyle         = lipgloss.NewStyle().Foreground(greenColor).Bold(true)
	infoStyle         = lipgloss.NewStyle().Foreground(cyanColor)
)

func modeCardStyle(width int, active bool) lipgloss.Style {
	color := borderColor
	if active {
		color = accentColor
	}
	return lipgloss.NewStyle().
		Width(max(20, width)).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color)
}

func panelStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(max(20, width-2)).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor)
}

func titleForMode(mode Mode) lipgloss.Style {
	switch mode {
	case ModeTrash:
		return lipgloss.NewStyle().Bold(true).Foreground(greenColor)
	case ModePermanent:
		return lipgloss.NewStyle().Bold(true).Foreground(redColor)
	default:
		return lipgloss.NewStyle().Bold(true).Foreground(cyanColor)
	}
}

func modeBadgeStyle(mode Mode) lipgloss.Style {
	return titleForMode(mode).Padding(0, 1).Reverse(true)
}

// screen identifies which dashboard page currently owns keyboard input.
// Named states are easier to extend and reason about than combinations such as
// reviewing=true, confirming=false, finished=false.
type screen uint8

const (
	screenMode screen = iota
	screenItems
	screenReview
)

// Result is the user's final dashboard decision. Confirmed records intent only;
// the dashboard package never removes resources itself.
type Result struct {
	Items     []scanner.Item
	Confirmed bool
	Mode      Mode
}

// NewModel builds a dashboard with the cursor on the first result and no items
// selected. The items slice is copied so callers cannot reorder it underneath
// the running UI.
func NewModel(items []scanner.Item, initialMode Mode) Model {
	modeCursor := 0
	for index, mode := range modes {
		if mode == initialMode {
			modeCursor = index
			break
		}
	}
	return Model{
		items:      append([]scanner.Item(nil), items...),
		selected:   make(map[int]struct{}),
		mode:       modes[modeCursor],
		modeCursor: modeCursor,
	}
}

// Init has no startup work yet. Bubble Tea commands are useful for asynchronous
// operations; this first version receives already-completed scan results.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update applies one event and returns the next model value. This is the
// Elm-style state transition at the heart of Bubble Tea applications.
func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := message.(tea.WindowSizeMsg); ok {
		m.width = size.Width
		m.height = size.Height
		m.ensureCursorVisible()
		return m, nil
	}

	key, ok := message.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	if key.String() == "q" || key.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.screen == screenMode {
		switch key.String() {
		case "up", "k":
			if m.modeCursor > 0 {
				m.modeCursor--
			}
		case "down", "j":
			if m.modeCursor+1 < len(modes) {
				m.modeCursor++
			}
		case "enter":
			m.mode = modes[m.modeCursor]
			m.dropUnsupportedSelections()
			m.screen = screenItems
		case "esc":
			return m, tea.Quit
		}
		return m, nil
	}

	if m.screen == screenReview {
		switch key.String() {
		case "esc":
			m.screen = screenItems
		case "up", "k":
			if m.reviewOffset > 0 {
				m.reviewOffset--
			}
		case "down", "j":
			if m.reviewOffset+m.visibleItemRows() < len(m.selectedItems()) {
				m.reviewOffset++
			}
		case "enter":
			m.confirmed = true
			return m, tea.Quit
		}
		return m, nil
	}

	switch key.String() {
	case "esc":
		m.screen = screenMode
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		m.ensureCursorVisible()
	case "down", "j":
		if m.cursor+1 < len(m.items) {
			m.cursor++
		}
		m.ensureCursorVisible()
	case "pgup":
		m.cursor -= m.visibleItemRows()
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.ensureCursorVisible()
	case "pgdown":
		m.cursor += m.visibleItemRows()
		if m.cursor >= len(m.items) {
			m.cursor = max(0, len(m.items)-1)
		}
		m.ensureCursorVisible()
	case "g", "home":
		m.cursor = 0
		m.ensureCursorVisible()
	case "G", "end":
		m.cursor = max(0, len(m.items)-1)
		m.ensureCursorVisible()
	case "space":
		m.toggleCurrent()
	case "a":
		m.selectAllSupported()
	case "c":
		clear(m.selected)
	case "d":
		if len(m.selected) > 0 {
			m.reviewOffset = 0
			m.screen = screenReview
		}
	}

	return m, nil
}

func (m Model) visibleItemRows() int {
	if m.height <= 0 {
		return 12
	}
	// Reserve lines for app padding, title, borders, status, description, and
	// help. A conservative budget avoids terminal scrolling at common heights.
	return max(1, m.height-16)
}

func (m *Model) selectAllSupported() {
	for index, item := range m.items {
		if m.canSelect(item) {
			m.selected[index] = struct{}{}
		}
	}
}

func (m *Model) ensureCursorVisible() {
	rows := m.visibleItemRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
	maxOffset := max(0, len(m.items)-rows)
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
}

func (m *Model) toggleCurrent() {
	if len(m.items) == 0 || !m.canSelect(m.items[m.cursor]) {
		return
	}
	if _, exists := m.selected[m.cursor]; exists {
		delete(m.selected, m.cursor)
		return
	}
	m.selected[m.cursor] = struct{}{}
}

func (m Model) canSelect(item scanner.Item) bool {
	switch m.mode {
	case ModeReadOnly:
		return true
	case ModeTrash:
		return item.ResourceType == scanner.ResourceFile || item.ResourceType == scanner.ResourceDirectory
	case ModePermanent:
		return item.ResourceType == scanner.ResourceFile ||
			item.ResourceType == scanner.ResourceDirectory ||
			item.ResourceType == scanner.ResourceDockerImage
	default:
		return false
	}
}

func (m *Model) dropUnsupportedSelections() {
	for index := range m.selected {
		if !m.canSelect(m.items[index]) {
			delete(m.selected, index)
		}
	}
}

// View derives the complete screen from the current state. AltScreen gives the
// dashboard its own temporary terminal surface; quitting restores prior output.
func (m Model) View() tea.View {
	var view strings.Builder
	width := m.contentWidth()
	view.WriteString(titleStyle.Render("SWEEPR"))
	view.WriteString("  ")
	view.WriteString(subtitleStyle.Render("developer cleanup dashboard"))
	view.WriteString("\n")
	view.WriteString(ruleStyle.Width(width).Render(""))
	view.WriteString("\n")

	if m.width > 0 && (m.width < 46 || m.height < 12) {
		view.WriteString(warningStyle.Render("Terminal is small; enlarge it for the best layout."))
		view.WriteString("\n")
	}
	if m.screen == screenMode {
		m.writeModes(&view)
	} else if m.screen == screenReview {
		m.writeReview(&view)
	} else {
		m.writeItems(&view)
	}

	style := appStyle
	if m.width > 0 {
		style = style.MaxWidth(m.width)
	}
	content := style.Render(view.String())
	result := tea.NewView(content)
	result.AltScreen = true
	result.WindowTitle = "sweepr dashboard"
	return result
}

func (m Model) writeModes(view *strings.Builder) {
	view.WriteString("\n")
	view.WriteString(sectionTitleStyle.Render("Choose an action"))
	view.WriteString("\n")
	view.WriteString(subtitleStyle.Render("You will review exact targets before anything changes."))
	view.WriteString("\n\n")
	for index, mode := range modes {
		cursor := "  "
		if index == m.modeCursor {
			cursor = "› "
		}
		label := modeIcon(mode) + "  " + modeTitle(mode)
		body := titleForMode(mode).Render(label) + "\n" + subtitleStyle.Render(modeDescription(mode))
		card := modeCardStyle(m.contentWidth()-4, index == m.modeCursor).Render(body)
		fmt.Fprintf(view, "%s%s\n", cursor, card)
	}
	view.WriteString("\n")
	view.WriteString(helpStyle.Render("↑/k up   ↓/j down   enter choose   q quit"))
}

func modeIcon(mode Mode) string {
	switch mode {
	case ModeReadOnly:
		return "◉"
	case ModeTrash:
		return "♲"
	case ModePermanent:
		return "!"
	default:
		return "?"
	}
}

func modeTitle(mode Mode) string {
	switch mode {
	case ModeReadOnly:
		return "Read only"
	case ModeTrash:
		return "Safe trash"
	case ModePermanent:
		return "Permanent delete"
	default:
		return "Unknown"
	}
}

func modeDescription(mode Mode) string {
	switch mode {
	case ModeReadOnly:
		return "Inspect selections; change nothing."
	case ModeTrash:
		return "Move files/directories to OS trash; Docker is unavailable."
	case ModePermanent:
		return "Permanently remove files, directories, and Docker images."
	default:
		return "Unsupported mode."
	}
}

func (m Model) writeItems(view *strings.Builder) {
	view.WriteString("\n")
	view.WriteString(sectionTitleStyle.Render("Scan results"))
	view.WriteString("  ")
	view.WriteString(modeBadgeStyle(m.mode).Render(modeTitle(m.mode)))
	view.WriteString("\n\n")

	var rows strings.Builder
	if len(m.items) == 0 {
		rows.WriteString(emptyStyle.Render("✓  No junk matched the selected filters."))
	} else {
		start := min(m.offset, len(m.items))
		end := min(len(m.items), start+m.visibleItemRows())
		pathWidth := max(12, m.contentWidth()-47)
		for index := start; index < end; index++ {
			item := m.items[index]
			cursor := " "
			if index == m.cursor {
				cursor = "›"
			}
			checkbox := "[ ]"
			if !m.canSelect(item) {
				checkbox = "[-]"
			} else if _, selected := m.selected[index]; selected {
				checkbox = "[x]"
			}

			row := fmt.Sprintf("%s %s %-19s %10s  %s",
				cursor,
				checkbox,
				item.Kind,
				formatSize(item.SizeBytes),
				truncateText(displayTarget(item), pathWidth),
			)
			if index == m.cursor {
				row = activeRowStyle.Render(row)
			} else if !m.canSelect(item) {
				row = disabledStyle.Render(row)
			} else if _, selected := m.selected[index]; selected {
				row = selectedStyle.Render(row)
			}
			rows.WriteString(row)
			rows.WriteString("\n")
		}
		fmt.Fprintf(&rows, "%s\n", scrollStyle.Render(fmt.Sprintf("Showing %d–%d of %d", start+1, end, len(m.items))))
	}
	view.WriteString(panelStyle(m.contentWidth()).Render(strings.TrimSuffix(rows.String(), "\n")))

	fmt.Fprintf(view, "\n%s   %s\n",
		statusStrongStyle.Render(fmt.Sprintf("%d/%d selected", len(m.selected), len(m.items))),
		statusStrongStyle.Render(formatSize(m.selectedBytes())+" total"))
	if len(m.items) > 0 {
		info := scanner.GetJunkInfo(m.items[m.cursor].Kind)
		view.WriteString(subtitleStyle.Render(truncateText(info.Description, m.contentWidth())))
		view.WriteString("\n")
	}
	view.WriteString(helpStyle.Render("↑↓/jk move   pgup/pgdn page   space toggle   a all   c clear   d review   esc modes   q quit"))
}

func (m Model) writeReview(view *strings.Builder) {
	view.WriteString("\n")
	view.WriteString(sectionTitleStyle.Render("Review selection"))
	view.WriteString("  ")
	view.WriteString(modeBadgeStyle(m.mode).Render(modeTitle(m.mode)))
	view.WriteString("\n")
	view.WriteString(subtitleStyle.Render("Verify every target before confirming."))
	view.WriteString("\n\n")

	selected := m.selectedItems()
	start := min(m.reviewOffset, len(selected))
	end := min(len(selected), start+m.visibleItemRows())
	pathWidth := max(12, m.contentWidth()-39)
	var rows strings.Builder
	for _, item := range selected[start:end] {
		fmt.Fprintf(&rows, "  %-19s %10s  %s\n",
			item.Kind, formatSize(item.SizeBytes), truncateText(displayTarget(item), pathWidth))
	}
	if len(selected) > m.visibleItemRows() {
		fmt.Fprintf(&rows, "%s\n", scrollStyle.Render(fmt.Sprintf("Showing %d–%d of %d", start+1, end, len(selected))))
	}
	view.WriteString(panelStyle(m.contentWidth()).Render(strings.TrimSuffix(rows.String(), "\n")))

	fmt.Fprintf(view, "\n%s\n",
		statusStrongStyle.Render(fmt.Sprintf("%d items • %s total", len(m.selected), formatSize(m.selectedBytes()))))
	message := ""
	help := ""
	switch m.mode {
	case ModePermanent:
		message = dangerStyle.Render("! PERMANENT: these resources cannot be restored.")
		help = "enter permanently delete   esc back   ↑/↓ scroll   q quit"
	case ModeTrash:
		message = safeStyle.Render("♲ Recoverable until the operating-system trash is emptied.")
		help = "enter move to trash   esc back   ↑/↓ scroll   q quit"
	default:
		message = infoStyle.Render("◉ Read-only preview: no resources will change.")
		help = "enter close preview   esc back   ↑/↓ scroll   q quit"
	}
	view.WriteString(message)
	view.WriteString("\n")
	view.WriteString(helpStyle.Render(help))
}

func (m Model) selectedBytes() int64 {
	var total int64
	for index := range m.selected {
		total += m.items[index].SizeBytes
	}
	return total
}

func (m Model) contentWidth() int {
	if m.width <= 0 {
		return 88
	}
	return max(30, min(100, m.width-6))
}

// truncateText limits text by terminal cells rather than bytes. This matters
// for Unicode because a rune may occupy zero, one, or two visible cells.
func truncateText(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}

	var result strings.Builder
	for _, r := range value {
		candidate := result.String() + string(r) + "…"
		if lipgloss.Width(candidate) > width {
			break
		}
		result.WriteRune(r)
	}
	return result.String() + "…"
}

func displayTarget(item scanner.Item) string {
	if item.DisplayName != "" {
		return item.DisplayName
	}
	return item.Path
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	exponent := int(math.Log2(float64(bytes)) / 10)
	if exponent > 6 {
		exponent = 6
	}
	divisor := math.Pow(unit, float64(exponent))
	return fmt.Sprintf("%.2f %cB", float64(bytes)/divisor, "KMGTPE"[exponent-1])
}

func (m Model) selectedItems() []scanner.Item {
	items := make([]scanner.Item, 0, len(m.selected))
	// Iterating over items, rather than over the map, preserves dashboard order.
	for index, item := range m.items {
		if _, selected := m.selected[index]; selected {
			items = append(items, item)
		}
	}
	return items
}

// Run starts the terminal event loop and blocks until the user quits. It
// returns data describing the user's decision but performs no deletion.
func Run(items []scanner.Item, initialMode Mode) (Result, error) {
	final, err := tea.NewProgram(NewModel(items, initialMode)).Run()
	if err != nil {
		return Result{}, err
	}
	model, ok := final.(Model)
	if !ok {
		return Result{}, fmt.Errorf("dashboard returned unexpected model type %T", final)
	}
	return Result{Items: model.selectedItems(), Confirmed: model.confirmed, Mode: model.mode}, nil
}
