package display

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/howmanysmall/relay/src/internal/core"
	"golang.org/x/term"
)

// Dashboard provides a live updating terminal interface.
type Dashboard struct {
	renderer     *ProgressRenderer
	engine       *core.SyncEngine
	refreshRate  time.Duration
	colorEnabled bool
	termWidth    int
	termHeight   int
	lastLines    int
}

// NewDashboard creates a new dashboard for the sync engine.
func NewDashboard(engine *core.SyncEngine, refreshRate time.Duration) *Dashboard {
	colorEnabled := term.IsTerminal(int(os.Stdout.Fd()))

	termWidth, termHeight, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termWidth, termHeight = 80, 24
	}

	return &Dashboard{
		renderer:     NewProgressRenderer(colorEnabled, termWidth),
		engine:       engine,
		refreshRate:  refreshRate,
		colorEnabled: colorEnabled,
		termWidth:    termWidth,
		termHeight:   termHeight,
	}
}

// Run starts the dashboard and updates it until the context is cancelled.
func (d *Dashboard) Run(ctx context.Context) {
	ticker := time.NewTicker(d.refreshRate)
	defer ticker.Stop()

	// Hide cursor
	if d.colorEnabled {
		fmt.Print("\033[?25l")
		defer fmt.Print("\033[?25h")
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.update()
		}
	}
}

// update refreshes the dashboard display.
func (d *Dashboard) update() {
	// Clear previous lines
	d.clearLines()

	// Get current state
	progress := d.engine.GetProgress()
	stats := d.engine.GetStats()
	errorSummary := d.engine.GetErrorSummary()

	var lines []string

	// Header
	header := d.formatHeader()
	if header != "" {
		lines = append(lines, header)
		lines = append(lines, "")
	}

	// Progress section
	progressLines := d.renderer.RenderProgress(progress, stats)
	if progressLines != "" {
		lines = append(lines, progressLines)
		lines = append(lines, "")
	}

	// Statistics section
	statsLines := d.renderer.RenderStats(stats)
	if statsLines != "" {
		lines = append(lines, statsLines)
		lines = append(lines, "")
	}

	// Errors section
	errorLines := d.renderer.RenderErrors(errorSummary)
	if errorLines != "" {
		lines = append(lines, errorLines)
		lines = append(lines, "")
	}

	// Print all lines
	output := strings.Join(lines, "\n")
	fmt.Print(output)

	// Track number of lines for clearing
	d.lastLines = strings.Count(output, "\n") + 1
}

// ShowCompletion displays a completion summary.
func (d *Dashboard) ShowCompletion(stats *core.SyncStats) {
	d.clearLines()

	var lines []string

	// Success header
	if stats.ErrorsEncountered == 0 {
		lines = append(lines, d.formatMessage("🎉 Synchronization completed successfully!", color.FgGreen))
	} else {
		lines = append(lines, d.formatMessage("⚠️  Synchronization completed with errors", color.FgYellow))
	}

	lines = append(lines, "")

	// Final statistics
	statsLines := d.renderer.RenderStats(stats)
	if statsLines != "" {
		lines = append(lines, statsLines)
		lines = append(lines, "")
	}

	// Error summary if any
	errorSummary := d.engine.GetErrorSummary()

	errorLines := d.renderer.RenderErrors(errorSummary)
	if errorLines != "" {
		lines = append(lines, errorLines)
		lines = append(lines, "")
	}

	fmt.Println(strings.Join(lines, "\n"))
}

// ShowError displays an error message.
func (d *Dashboard) ShowError(err error) {
	d.clearLines()

	errorMsg := fmt.Sprintf("❌ Error: %v", err)
	fmt.Println(d.formatMessage(errorMsg, color.FgRed))
}

// formatHeader creates a header with title and separator.
func (d *Dashboard) formatHeader() string {
	title := "🚀 Relay File Synchronization"
	if !d.colorEnabled {
		title = "Relay File Synchronization"
	}

	titleLine := d.formatMessage(title, color.FgCyan)
	separator := strings.Repeat("─", d.termWidth-1)
	separatorLine := d.formatMessage(separator, color.FgBlue)

	return titleLine + "\n" + separatorLine
}

// clearLines clears the previously printed lines.
func (d *Dashboard) clearLines() {
	if d.lastLines > 0 && d.colorEnabled {
		// Move cursor up and clear lines
		fmt.Printf("\033[%dA", d.lastLines)

		for i := 0; i < d.lastLines; i++ {
			fmt.Print("\033[2K\033[1B")
		}

		fmt.Printf("\033[%dA", d.lastLines)
	}
}

// formatMessage applies color formatting if enabled.
func (d *Dashboard) formatMessage(text string, colorAttr color.Attribute) string {
	return d.renderer.formatMessage(text, colorAttr)
}

// PrintSimpleProgress prints progress without dashboard (for non-interactive mode).
func PrintSimpleProgress(engine *core.SyncEngine, colorEnabled bool) {
	renderer := NewProgressRenderer(colorEnabled, 80)

	progress := engine.GetProgress()
	stats := engine.GetStats()

	progressLine := renderer.RenderProgress(progress, stats)
	fmt.Println(progressLine)
}

// PrintSimpleStats prints final statistics without dashboard.
func PrintSimpleStats(engine *core.SyncEngine, colorEnabled bool) {
	renderer := NewProgressRenderer(colorEnabled, 80)

	stats := engine.GetStats()
	errorSummary := engine.GetErrorSummary()

	statsLines := renderer.RenderStats(stats)
	if statsLines != "" {
		fmt.Println(statsLines)
	}

	errorLines := renderer.RenderErrors(errorSummary)
	if errorLines != "" {
		fmt.Println(errorLines)
	}
}
