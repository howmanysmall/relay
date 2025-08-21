package display

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
)

// StatusType represents different types of status messages.
type StatusType int

// StatusType values enumerate the kinds of status messages that can be rendered.
const (
	StatusInfo StatusType = iota
	StatusSuccess
	StatusWarning
	StatusError
	StatusProgress
)

// StatusMessage represents a status message with formatting.
type StatusMessage struct {
	Type      StatusType
	Message   string
	Timestamp time.Time
	Details   string
}

// StatusRenderer handles rendering status messages.
type StatusRenderer struct {
	colorEnabled bool
	showTime     bool
}

// NewStatusRenderer creates a new status renderer.
func NewStatusRenderer(colorEnabled, showTime bool) *StatusRenderer {
	return &StatusRenderer{
		colorEnabled: colorEnabled,
		showTime:     showTime,
	}
}

// RenderStatus renders a status message with appropriate formatting.
func (sr *StatusRenderer) RenderStatus(status *StatusMessage) string {
	icon := sr.getStatusIcon(status.Type)
	color := sr.getStatusColor(status.Type)

	var parts []string

	// Add timestamp if enabled
	if sr.showTime {
		timestamp := status.Timestamp.Format("15:04:05")
		parts = append(parts, sr.formatMessage(fmt.Sprintf("[%s]", timestamp), FgWhite))
	}

	// Add icon and message
	message := fmt.Sprintf("%s %s", icon, status.Message)
	parts = append(parts, sr.formatMessage(message, color))

	result := strings.Join(parts, " ")

	// Add details if present
	if status.Details != "" {
		result += "\n" + sr.formatDetails(status.Details)
	}

	return result
}

// PrintInfo prints an info message.
func (sr *StatusRenderer) PrintInfo(message string, details ...string) {
	status := &StatusMessage{
		Type:      StatusInfo,
		Message:   message,
		Timestamp: time.Now(),
	}

	if len(details) > 0 {
		status.Details = strings.Join(details, "\n")
	}

	fmt.Println(sr.RenderStatus(status))
}

// PrintSuccess prints a success message.
func (sr *StatusRenderer) PrintSuccess(message string, details ...string) {
	status := &StatusMessage{
		Type:      StatusSuccess,
		Message:   message,
		Timestamp: time.Now(),
	}

	if len(details) > 0 {
		status.Details = strings.Join(details, "\n")
	}

	fmt.Println(sr.RenderStatus(status))
}

// PrintWarning prints a warning message.
func (sr *StatusRenderer) PrintWarning(message string, details ...string) {
	status := &StatusMessage{
		Type:      StatusWarning,
		Message:   message,
		Timestamp: time.Now(),
	}
	if len(details) > 0 {
		status.Details = strings.Join(details, "\n")
	}

	fmt.Println(sr.RenderStatus(status))
}

// PrintError prints an error message.
func (sr *StatusRenderer) PrintError(message string, details ...string) {
	status := &StatusMessage{
		Type:      StatusError,
		Message:   message,
		Timestamp: time.Now(),
	}
	if len(details) > 0 {
		status.Details = strings.Join(details, "\n")
	}

	fmt.Println(sr.RenderStatus(status))
}

// PrintProgress prints a progress message.
func (sr *StatusRenderer) PrintProgress(message string, details ...string) {
	status := &StatusMessage{
		Type:      StatusProgress,
		Message:   message,
		Timestamp: time.Now(),
	}
	if len(details) > 0 {
		status.Details = strings.Join(details, "\n")
	}

	fmt.Println(sr.RenderStatus(status))
}

// getStatusIcon returns the appropriate icon for a status type.
func (sr *StatusRenderer) getStatusIcon(statusType StatusType) string {
	switch statusType {
	case StatusInfo:
		return "ℹ️"
	case StatusSuccess:
		return "✅"
	case StatusWarning:
		return "⚠️"
	case StatusError:
		return "❌"
	case StatusProgress:
		return "🔄"
	default:
		return "•"
	}
}

// getStatusColor returns the appropriate color for a status type.
func (sr *StatusRenderer) getStatusColor(statusType StatusType) color.Attribute {
	switch statusType {
	case StatusInfo:
		return color.FgCyan
	case StatusSuccess:
		return color.FgGreen
	case StatusWarning:
		return color.FgYellow
	case StatusError:
		return color.FgRed
	case StatusProgress:
		return color.FgBlue
	default:
		return FgWhite
	}
}

// formatMessage applies color formatting if enabled.
func (sr *StatusRenderer) formatMessage(text string, colorAttr color.Attribute) string {
	if !sr.colorEnabled {
		return text
	}

	return color.New(colorAttr).Sprint(text)
}

// formatDetails formats detail text with indentation.
func (sr *StatusRenderer) formatDetails(details string) string {
	lines := strings.Split(details, "\n")

	var formattedLines []string

	for _, line := range lines {
		if line != "" {
			formattedLine := "  " + sr.formatMessage(line, FgWhite)
			formattedLines = append(formattedLines, formattedLine)
		}
	}

	return strings.Join(formattedLines, "\n")
}

// CreateBanner creates a decorative banner for the application.
func CreateBanner(title string, colorEnabled bool) string {
	width := 60

	var lines []string

	// Top border
	topBorder := "╭" + strings.Repeat("─", width-2) + "╮"
	if colorEnabled {
		topBorder = color.New(color.FgCyan).Sprint(topBorder)
	}

	lines = append(lines, topBorder)

	// Title line
	padding := (width - len(title) - 2) / 2
	leftPad := strings.Repeat(" ", padding)
	rightPad := strings.Repeat(" ", width-len(title)-padding-2)
	titleLine := "│" + leftPad + title + rightPad + "│"

	if colorEnabled {
		titleLine = color.New(color.FgCyan).Sprint("│") +
			color.New(FgWhite, color.Bold).Sprint(leftPad+title+rightPad) +
			color.New(color.FgCyan).Sprint("│")
	}

	lines = append(lines, titleLine)

	// Bottom border
	bottomBorder := "╰" + strings.Repeat("─", width-2) + "╯"
	if colorEnabled {
		bottomBorder = color.New(color.FgCyan).Sprint(bottomBorder)
	}

	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
}

// CreateSeparator creates a visual separator line.
func CreateSeparator(width int, colorEnabled bool) string {
	if width <= 0 {
		width = 60
	}

	separator := strings.Repeat("─", width)
	if colorEnabled {
		separator = color.New(color.FgBlue).Sprint(separator)
	}

	return separator
}
