// Package display provides beautiful terminal UI components for relay.
package display

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/howmanysmall/relay/src/internal/core"
)

// ProgressRenderer handles rendering progress information to the terminal.
type ProgressRenderer struct {
	colorEnabled bool
	width        int
}

// NewProgressRenderer creates a new progress renderer.
func NewProgressRenderer(colorEnabled bool, terminalWidth int) *ProgressRenderer {
	if terminalWidth <= 0 {
		terminalWidth = 80
	}

	return &ProgressRenderer{
		colorEnabled: colorEnabled,
		width:        terminalWidth,
	}
}

// RenderProgress renders a progress bar with statistics.
func (pr *ProgressRenderer) RenderProgress(progress *core.Progress, _ *core.SyncStats) string {
	if progress.Total == 0 {
		return pr.formatMessage("🔍 Scanning files...", color.FgCyan)
	}

	percentage := progress.Percentage
	if percentage > 100 {
		percentage = 100
	}

	// Calculate progress bar width (reserve space for text)
	barWidth := pr.width - 50
	if barWidth < 20 {
		barWidth = 20
	}

	filled := int(float64(barWidth) * percentage / 100)
	if filled > barWidth {
		filled = barWidth
	}

	// Create progress bar
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Format speed
	speed := pr.formatSpeed(progress.Speed)

	// Format ETA
	eta := pr.formatDuration(progress.ETA)

	// Current file (truncate if too long)
	currentFile := progress.CurrentFile

	maxFileLen := 30
	if len(currentFile) > maxFileLen {
		currentFile = "..." + currentFile[len(currentFile)-maxFileLen+3:]
	}

	progressLine := fmt.Sprintf("📁 %s %s %6.1f%% %s ETA: %s",
		bar,
		pr.formatMessage(fmt.Sprintf("%d/%d", progress.Current, progress.Total), FgWhite),
		percentage,
		speed,
		eta,
	)

	statusLine := fmt.Sprintf("📄 %s", pr.formatMessage(currentFile, color.FgYellow))

	return progressLine + "\n" + statusLine
}

// RenderStats renders synchronization statistics.
func (pr *ProgressRenderer) RenderStats(stats *core.SyncStats) string {
	if stats.StartTime.IsZero() {
		return ""
	}

	var lines []string

	// Summary line
	summary := fmt.Sprintf("✅ Completed: %s | 📝 Created: %s | 🔄 Modified: %s | ⚠️  Errors: %s",
		pr.formatMessage(fmt.Sprintf("%d files", stats.FilesChanged), color.FgGreen),
		pr.formatMessage(fmt.Sprintf("%d", stats.FilesCreated), color.FgBlue),
		pr.formatMessage(fmt.Sprintf("%d", stats.FilesModified), color.FgYellow),
		pr.formatMessage(fmt.Sprintf("%d", stats.ErrorsEncountered), color.FgRed),
	)
	lines = append(lines, summary)

	// Transfer stats
	if stats.BytesTransferred > 0 {
		transferred := pr.formatBytes(stats.BytesTransferred)

		var duration time.Duration
		if !stats.EndTime.IsZero() {
			duration = stats.EndTime.Sub(stats.StartTime)
		} else {
			duration = time.Since(stats.StartTime)
		}

		avgSpeed := float64(stats.BytesTransferred) / duration.Seconds()
		speed := pr.formatSpeed(int64(avgSpeed))

		transferLine := fmt.Sprintf("📊 Transferred: %s in %s (avg: %s)",
			pr.formatMessage(transferred, color.FgCyan),
			pr.formatMessage(pr.formatDuration(duration), FgWhite),
			pr.formatMessage(speed, color.FgGreen),
		)
		lines = append(lines, transferLine)
	}

	// Conflicts
	if stats.ConflictsFound > 0 {
		conflictLine := fmt.Sprintf("⚔️  Conflicts: %s found, %s resolved",
			pr.formatMessage(fmt.Sprintf("%d", stats.ConflictsFound), color.FgMagenta),
			pr.formatMessage(fmt.Sprintf("%d", stats.ConflictsResolved), color.FgGreen),
		)
		lines = append(lines, conflictLine)
	}

	return strings.Join(lines, "\n")
}

// RenderErrors renders error information.
func (pr *ProgressRenderer) RenderErrors(errorSummary map[core.ErrorCategory]int) string {
	if len(errorSummary) == 0 {
		return ""
	}

	var lines []string

	lines = append(lines, pr.formatMessage("❌ Errors encountered:", color.FgRed))

	for category, count := range errorSummary {
		if count > 0 {
			icon := pr.getErrorIcon(category)
			categoryName := pr.getErrorCategoryName(category)
			line := fmt.Sprintf("  %s %s: %s",
				icon,
				categoryName,
				pr.formatMessage(fmt.Sprintf("%d", count), FgWhite),
			)
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

// formatMessage applies color formatting if enabled.
func (pr *ProgressRenderer) formatMessage(text string, colorAttr color.Attribute) string {
	if !pr.colorEnabled {
		return text
	}

	return color.New(colorAttr).Sprint(text)
}

// formatSpeed formats transfer speed in human-readable format.
func (pr *ProgressRenderer) formatSpeed(bytesPerSecond int64) string {
	if bytesPerSecond == 0 {
		return "-- B/s"
	}

	units := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	size := float64(bytesPerSecond)
	unitIndex := 0

	for size >= 1024 && unitIndex < len(units)-1 {
		size /= 1024
		unitIndex++
	}

	if unitIndex == 0 {
		return fmt.Sprintf("%.0f %s", size, units[unitIndex])
	}

	return fmt.Sprintf("%.1f %s", size, units[unitIndex])
}

// formatBytes formats byte count in human-readable format.
func (pr *ProgressRenderer) formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(bytes)
	unitIndex := 0

	for size >= 1024 && unitIndex < len(units)-1 {
		size /= 1024
		unitIndex++
	}

	if unitIndex == 0 {
		return fmt.Sprintf("%.0f %s", size, units[unitIndex])
	}

	return fmt.Sprintf("%.1f %s", size, units[unitIndex])
}

// formatDuration formats duration in human-readable format.
func (pr *ProgressRenderer) formatDuration(d time.Duration) string {
	if d <= 0 {
		return "--"
	}

	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}

	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}

	return fmt.Sprintf("%.1fh", d.Hours())
}

// getErrorIcon returns an icon for the error category.
func (pr *ProgressRenderer) getErrorIcon(category core.ErrorCategory) string {
	switch category {
	case core.ErrorCategoryNetwork:
		return "🌐"
	case core.ErrorCategoryPermission:
		return "🔒"
	case core.ErrorCategoryDisk:
		return "💾"
	case core.ErrorCategoryCorruption:
		return "⚠️"
	case core.ErrorCategoryConfiguration:
		return "⚙️"
	case core.ErrorCategoryCancellation:
		return "🛑"
	default:
		return "❓"
	}
}

// getErrorCategoryName returns a human-readable name for the error category.
func (pr *ProgressRenderer) getErrorCategoryName(category core.ErrorCategory) string {
	switch category {
	case core.ErrorCategoryNetwork:
		return "Network"
	case core.ErrorCategoryPermission:
		return "Permission"
	case core.ErrorCategoryDisk:
		return "Disk Space"
	case core.ErrorCategoryCorruption:
		return "Data Corruption"
	case core.ErrorCategoryConfiguration:
		return "Configuration"
	case core.ErrorCategoryCancellation:
		return "Cancelled"
	default:
		return "Unknown"
	}
}
