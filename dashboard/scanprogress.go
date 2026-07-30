package dashboard

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"sweepr/scanner"
)

// ScanEvent is emitted by the scan coordinator. Progress events are
// best-effort; completion events contain the authoritative results.
type ScanEvent struct {
	Name     string
	Progress scanner.Progress
	Items    []scanner.Item
	Err      error
	Duration time.Duration
	Done     bool
}

type scanChannelClosed struct{}

type ScanProgressModel struct {
	names     []string
	events    <-chan ScanEvent
	latest    map[string]ScanEvent
	items     []scanner.Item
	completed int
	cancelled bool
	width     int
	height    int
}

func NewScanProgressModel(names []string, events <-chan ScanEvent) ScanProgressModel {
	return ScanProgressModel{names: append([]string(nil), names...), events: events, latest: make(map[string]ScanEvent)}
}

func waitForScanEvent(events <-chan ScanEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			return scanChannelClosed{}
		}
		return event
	}
}

func (m ScanProgressModel) Init() tea.Cmd { return waitForScanEvent(m.events) }

func (m ScanProgressModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = message.Width, message.Height
	case tea.KeyPressMsg:
		if message.String() == "q" || message.String() == "esc" || message.String() == "ctrl+c" {
			m.cancelled = true
			return m, tea.Quit
		}
	case ScanEvent:
		m.latest[message.Name] = message
		if message.Done {
			m.completed++
			m.items = append(m.items, message.Items...)
		}
		if m.completed >= len(m.names) {
			return m, tea.Quit
		}
		return m, waitForScanEvent(m.events)
	case scanChannelClosed:
		return m, tea.Quit
	}
	return m, nil
}

func (m ScanProgressModel) View() tea.View {
	var view strings.Builder
	view.WriteString(titleStyle.Render("SWEEPR"))
	view.WriteString("  ")
	view.WriteString(renderBrandSubtitle("scanning"))
	view.WriteString("\n")
	view.WriteString(ruleStyle.Width(m.progressWidth()).Render(""))
	view.WriteString("\n\n")
	view.WriteString(sectionTitleStyle.Render("Scanning safely"))
	view.WriteString("\n")
	view.WriteString(subtitleStyle.Render("Scanning is read-only. No files or images are being removed."))
	view.WriteString("\n\n")
	var rows strings.Builder
	for _, name := range m.names {
		event, exists := m.latest[name]
		status := "Waiting…"
		if exists && event.Done && event.Err != nil {
			status = dangerStyle.Render("Failed: " + event.Err.Error())
		} else if exists && event.Done {
			status = safeStyle.Render(fmt.Sprintf("Complete • %d items • %s", len(event.Items), event.Duration.Round(time.Millisecond)))
		} else if exists {
			status = fmt.Sprintf("%d entries • %d found • %s", event.Progress.EntriesScanned, event.Progress.ItemsFound, truncateText(event.Progress.Path, max(12, m.progressWidth()-42)))
		}
		fmt.Fprintf(&rows, "%-28s %s\n", name, status)
	}
	view.WriteString(panelStyle(m.progressWidth()).Render(strings.TrimSuffix(rows.String(), "\n")))
	view.WriteString("\n")
	view.WriteString(statusStrongStyle.Render(fmt.Sprintf("%d/%d scanners complete", m.completed, len(m.names))))
	view.WriteString("\n")
	view.WriteString(renderShortcuts(shortcut{"Esc/Q", "Cancel scan and exit"}))
	style := appStyle
	if m.width > 0 {
		style = style.MaxWidth(m.width)
	}
	result := tea.NewView(style.Render(view.String()))
	result.AltScreen = true
	result.WindowTitle = "sweepr scanning"
	return result
}

func (m ScanProgressModel) progressWidth() int {
	if m.width <= 0 {
		return 88
	}
	return max(30, min(100, m.width-6))
}

func RunScanProgress(names []string, events <-chan ScanEvent) ([]scanner.Item, bool, error) {
	final, err := tea.NewProgram(NewScanProgressModel(names, events)).Run()
	if err != nil {
		return nil, false, err
	}
	model, ok := final.(ScanProgressModel)
	if !ok {
		return nil, false, fmt.Errorf("scan progress returned unexpected model type %T", final)
	}
	return model.items, model.cancelled, nil
}
