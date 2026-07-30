// Package dashboard provides sweepr's interactive terminal interface.
package dashboard

import (
	"fmt"
	"math"
	"strings"

	tea "charm.land/bubbletea/v2"

	"sweepr/scanner"
)

// Model contains all state needed to render and update the dashboard. Keeping
// state in a plain Go value makes keyboard behavior testable without starting
// a real terminal.
type Model struct {
	items      []scanner.Item
	cursor     int
	selected   map[int]struct{}
	screen     screen
	confirmed  bool
	mode       Mode
	modeCursor int
}

// Mode describes the action the user chose for selected resources.
type Mode string

const (
	ModeReadOnly  Mode = "read-only"
	ModeTrash     Mode = "trash"
	ModePermanent Mode = "permanent-delete"
)

var modes = [...]Mode{ModeReadOnly, ModeTrash, ModePermanent}

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
	case "down", "j":
		if m.cursor+1 < len(m.items) {
			m.cursor++
		}
	case "space":
		m.toggleCurrent()
	case "d":
		if len(m.selected) > 0 {
			m.screen = screenReview
		}
	}

	return m, nil
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
	view.WriteString("sweepr dashboard\n")
	view.WriteString("────────────────────────────────────────────────────────────────────\n")
	if m.screen == screenMode {
		m.writeModes(&view)
	} else if m.screen == screenReview {
		m.writeReview(&view)
	} else {
		m.writeItems(&view)
	}

	result := tea.NewView(view.String())
	result.AltScreen = true
	result.WindowTitle = "sweepr dashboard"
	return result
}

func (m Model) writeModes(view *strings.Builder) {
	view.WriteString("\nChoose how selected resources should be handled:\n\n")
	for index, mode := range modes {
		cursor := " "
		if index == m.modeCursor {
			cursor = ">"
		}
		fmt.Fprintf(view, "%s %-18s %s\n", cursor, modeTitle(mode), modeDescription(mode))
	}
	view.WriteString("\n↑/k up  ↓/j down  enter choose  q quit")
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
	if len(m.items) == 0 {
		view.WriteString("\nNo junk matched the selected filters.\n")
	} else {
		for index, item := range m.items {
			cursor := " "
			if index == m.cursor {
				cursor = ">"
			}
			checkbox := "[ ]"
			if !m.canSelect(item) {
				checkbox = "[-]"
			} else if _, selected := m.selected[index]; selected {
				checkbox = "[x]"
			}

			fmt.Fprintf(view, "%s %s %-20s %10s  %s\n",
				cursor,
				checkbox,
				item.Kind,
				formatSize(item.SizeBytes),
				displayTarget(item),
			)
		}
	}

	fmt.Fprintf(view, "\nSelected: %d/%d  Reclaimable: %s\n",
		len(m.selected), len(m.items), formatSize(m.selectedBytes()))
	fmt.Fprintf(view, "Mode: %s\n", modeTitle(m.mode))
	view.WriteString("↑/k up  ↓/j down  space toggle  d review  esc modes  q quit")
}

func (m Model) writeReview(view *strings.Builder) {
	view.WriteString("\nReview selection\n\n")
	for index, item := range m.items {
		if _, selected := m.selected[index]; !selected {
			continue
		}
		fmt.Fprintf(view, "  %-20s %10s  %s\n",
			item.Kind, formatSize(item.SizeBytes), displayTarget(item))
	}

	fmt.Fprintf(view, "\n%d items selected • %s reclaimable\n",
		len(m.selected), formatSize(m.selectedBytes()))
	switch m.mode {
	case ModePermanent:
		view.WriteString("\nWARNING: Enter will permanently delete these resources.\n")
		view.WriteString("enter DELETE selected  esc back  q quit")
	case ModeTrash:
		view.WriteString("\nItems will move to OS trash and remain recoverable until trash is emptied.\n")
		view.WriteString("enter MOVE TO TRASH  esc back  q quit")
	default:
		view.WriteString("\nRead-only mode: no resources will be deleted.\n")
		view.WriteString("enter confirm preview  esc back  q quit")
	}
}

func (m Model) selectedBytes() int64 {
	var total int64
	for index := range m.selected {
		total += m.items[index].SizeBytes
	}
	return total
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
