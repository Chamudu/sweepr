package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"sweepr/scanner"
)

const progressRefreshInterval = 100 * time.Millisecond

var progressFrames = [...]string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// progressRenderer owns the temporary one-line terminal display for one
// scanner. Scanner callbacks only provide data; terminal presentation remains
// in the CLI layer.
type progressRenderer struct {
	enabled    bool
	name       string
	start      time.Time
	lastRender time.Time
	frame      int
	latest     scanner.Progress
}

func newProgressRenderer(name string, enabled bool) *progressRenderer {
	return &progressRenderer{
		enabled: enabled,
		name:    name,
		start:   time.Now(),
	}
}

// Update stores every snapshot but redraws at most ten times per second. This
// prevents terminal I/O from becoming a significant part of scan time.
func (r *progressRenderer) Update(progress scanner.Progress) {
	r.latest = progress
	if !r.enabled {
		return
	}

	now := time.Now()
	if !r.lastRender.IsZero() && now.Sub(r.lastRender) < progressRefreshInterval {
		return
	}
	r.render(now)
}

func (r *progressRenderer) render(now time.Time) {
	frame := progressFrames[r.frame%len(progressFrames)]
	r.frame++
	r.lastRender = now

	fmt.Fprintf(os.Stderr, "\r\033[2K%s %-10s | %d entries | %d found | %s | %s | %s",
		frame,
		r.name,
		r.latest.EntriesScanned,
		r.latest.ItemsFound,
		formatSize(r.latest.BytesFound),
		now.Sub(r.start).Round(100*time.Millisecond),
		shortProgressPath(r.latest.Path, 64),
	)
}

// Finish removes the temporary progress line before permanent report output is
// printed. The per-scanner completion line is produced by main.go.
func (r *progressRenderer) Finish() {
	if r.enabled {
		fmt.Fprint(os.Stderr, "\r\033[2K")
	}
}

// shortProgressPath keeps the most useful end of deeply nested paths visible
// and prevents normal terminal widths from wrapping the progress line.
func shortProgressPath(path string, max int) string {
	runes := []rune(path)
	if len(runes) <= max {
		return path
	}
	return "…" + string(runes[len(runes)-(max-1):])
}

// isInteractiveTerminal avoids ANSI control sequences when stderr is piped to
// a file or another command. It uses only the standard library.
func isInteractiveTerminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
