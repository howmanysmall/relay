package display

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/howmanysmall/relay/src/internal/core"
)

// ConflictUI provides an interactive interface for resolving file conflicts.
type ConflictUI struct {
	colorEnabled bool
	reader       *bufio.Reader
}

// NewConflictUI creates a new conflict resolution interface.
func NewConflictUI(colorEnabled bool) *ConflictUI {
	return &ConflictUI{
		colorEnabled: colorEnabled,
		reader:       bufio.NewReader(os.Stdin),
	}
}

// ShowConflict displays conflict information and prompts for resolution.
func (cui *ConflictUI) ShowConflict(conflict *core.ConflictInfo) (core.ConflictResolution, error) {
	cui.clearScreen()
	cui.printHeader("File Conflict Detected")

	// Show conflict details
	cui.printConflictDetails(conflict)

	// Show file comparison
	cui.printFileComparison(conflict)

	// Show resolution options
	cui.printResolutionOptions()

	// Get user choice
	return cui.promptForResolution(conflict)
}

// ShowBatchConflictOptions shows options for handling multiple conflicts.
func (cui *ConflictUI) ShowBatchConflictOptions() (core.ConflictResolution, bool, error) {
	cui.printHeader("Multiple Conflicts Detected")

	fmt.Println(cui.formatMessage("Choose how to handle remaining conflicts:", color.FgYellow))
	fmt.Println()

	options := []string{
		"[s] Always use source (overwrite destination)",
		"[d] Always use destination (keep current)",
		"[n] Always use newest file",
		"[b] Always backup destination and use source",
		"[k] Always skip conflicts",
		"[i] Continue with interactive resolution",
	}

	for _, option := range options {
		fmt.Println("  " + cui.formatMessage(option, FgWhite))
	}

	fmt.Println()

	for {
		fmt.Print(cui.formatMessage("Your choice [s/d/n/b/k/i]: ", color.FgCyan))

		input, err := cui.reader.ReadString('\n')
		if err != nil {
			return core.ResolutionSkip, false, fmt.Errorf("failed to read input: %w", err)
		}

		choice := strings.ToLower(strings.TrimSpace(input))
		switch choice {
		case "s", "source":
			return core.ResolutionUseSource, true, nil
		case "d", "dest", "destination":
			return core.ResolutionUseDestination, true, nil
		case "n", "newest":
			// Return a special value to indicate "newest" strategy
			return core.ResolutionUseSource, true, nil // Will be handled by caller
		case "b", "backup":
			return core.ResolutionBackupAndUseSource, true, nil
		case "k", "skip":
			return core.ResolutionSkip, true, nil
		case "i", "interactive":
			return core.ResolutionSkip, false, nil // Continue interactive mode
		default:
			fmt.Println(cui.formatMessage("Invalid choice. Please enter s, d, n, b, k, or i.", color.FgRed))
			continue
		}
	}
}

// printHeader prints a styled header.
func (cui *ConflictUI) printHeader(title string) {
	width := 80

	// Title line
	titleLine := fmt.Sprintf("🔄 %s", title)
	fmt.Println(cui.formatMessage(titleLine, color.FgCyan))

	// Separator
	separator := strings.Repeat("═", width-1)
	fmt.Println(cui.formatMessage(separator, color.FgBlue))
	fmt.Println()
}

// printConflictDetails shows details about the conflict.
func (cui *ConflictUI) printConflictDetails(conflict *core.ConflictInfo) {
	fmt.Println(cui.formatMessage("📁 File:", color.FgYellow), conflict.Path)
	fmt.Println(cui.formatMessage("⚠️  Conflict:", color.FgRed), conflict.Conflict.String())
	fmt.Println()
}

// printFileComparison shows a comparison between source and destination files.
func (cui *ConflictUI) printFileComparison(conflict *core.ConflictInfo) {
	fmt.Println(cui.formatMessage("📊 File Comparison", color.FgMagenta))
	fmt.Println(strings.Repeat("─", 50))

	// Source file info
	fmt.Printf("%-15s %s\n",
		cui.formatMessage("Source:", color.FgGreen),
		conflict.SourceInfo.Path)
	fmt.Printf("%-15s %s\n",
		"Size:",
		cui.formatBytes(conflict.SourceInfo.Size))
	fmt.Printf("%-15s %s\n",
		"Modified:",
		conflict.SourceInfo.ModTime.Format("2006-01-02 15:04:05"))

	if conflict.SourceInfo.Checksum != "" {
		fmt.Printf("%-15s %s\n",
			"Checksum:",
			conflict.SourceInfo.Checksum[:16]+"...")
	}

	fmt.Println()

	// Destination file info
	fmt.Printf("%-15s %s\n",
		cui.formatMessage("Destination:", color.FgBlue),
		conflict.DestInfo.Path)
	fmt.Printf("%-15s %s\n",
		"Size:",
		cui.formatBytes(conflict.DestInfo.Size))
	fmt.Printf("%-15s %s\n",
		"Modified:",
		conflict.DestInfo.ModTime.Format("2006-01-02 15:04:05"))

	if conflict.DestInfo.Checksum != "" {
		fmt.Printf("%-15s %s\n",
			"Checksum:",
			conflict.DestInfo.Checksum[:16]+"...")
	}

	fmt.Println()
}

// printResolutionOptions shows available resolution options.
func (cui *ConflictUI) printResolutionOptions() {
	fmt.Println(cui.formatMessage("🛠️  Resolution Options", color.FgMagenta))
	fmt.Println(strings.Repeat("─", 50))

	options := []string{
		"[s] Use source (overwrite destination)",
		"[d] Use destination (keep current)",
		"[b] Backup destination and use source",
		"[k] Skip this file",
		"[v] View file content preview",
		"[a] Apply to all remaining conflicts",
	}

	for _, option := range options {
		fmt.Println("  " + cui.formatMessage(option, FgWhite))
	}

	fmt.Println()
}

// promptForResolution prompts the user for a resolution choice.
func (cui *ConflictUI) promptForResolution(conflict *core.ConflictInfo) (core.ConflictResolution, error) {
	for {
		fmt.Print(cui.formatMessage("Your choice [s/d/b/k/v/a]: ", color.FgCyan))

		input, err := cui.reader.ReadString('\n')
		if err != nil {
			return core.ResolutionSkip, fmt.Errorf("failed to read input: %w", err)
		}

		choice := strings.ToLower(strings.TrimSpace(input))
		switch choice {
		case "s", "source":
			return core.ResolutionUseSource, nil

		case "d", "dest", "destination":
			return core.ResolutionUseDestination, nil

		case "b", "backup":
			return core.ResolutionBackupAndUseSource, nil

		case "k", "skip":
			return core.ResolutionSkip, nil

		case "v", "view":
			cui.showFilePreview(conflict)
			continue

		case "a", "all":
			resolution, _, err := cui.ShowBatchConflictOptions()
			return resolution, err

		default:
			fmt.Println(cui.formatMessage("Invalid choice. Please enter s, d, b, k, v, or a.", color.FgRed))
			continue
		}
	}
}

// showFilePreview displays a preview of the conflicting files.
func (cui *ConflictUI) showFilePreview(conflict *core.ConflictInfo) {
	fmt.Println()
	fmt.Println(cui.formatMessage("📄 File Content Preview", color.FgMagenta))
	fmt.Println(strings.Repeat("═", 50))

	// Check file sizes
	maxPreviewSize := int64(1024 * 10) // 10KB
	if conflict.SourceInfo.Size > maxPreviewSize || conflict.DestInfo.Size > maxPreviewSize {
		fmt.Println(cui.formatMessage("Files too large for preview (>10KB)", color.FgYellow))
		return
	}

	// Show source preview
	fmt.Println(cui.formatMessage("Source file:", color.FgGreen))
	fmt.Println(strings.Repeat("─", 25))

	if err := cui.showFileLines(conflict.SourceInfo.Path, 10); err != nil {
		fmt.Printf("Error reading source: %v\n", err)
	}

	fmt.Println()

	// Show destination preview
	fmt.Println(cui.formatMessage("Destination file:", color.FgBlue))
	fmt.Println(strings.Repeat("─", 25))

	if err := cui.showFileLines(conflict.DestInfo.Path, 10); err != nil {
		fmt.Printf("Error reading destination: %v\n", err)
	}

	fmt.Println()
	fmt.Print(cui.formatMessage("Press Enter to continue...", FgWhite))

	if _, err := cui.reader.ReadString('\n'); err != nil {
		_ = err // ignore read error for preview pause
	}

	// Redisplay conflict information
	cui.clearScreen()
	cui.printHeader("File Conflict Detected")
	cui.printConflictDetails(conflict)
	cui.printFileComparison(conflict)
	cui.printResolutionOptions()
}

// showFileLines displays the first N lines of a file.
func (cui *ConflictUI) showFileLines(path string, maxLines int) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := file.Close(); cerr != nil {
			_ = cerr // intentionally ignore close error
		}
	}()

	scanner := bufio.NewScanner(file)
	lineNum := 1

	for scanner.Scan() && lineNum <= maxLines {
		line := scanner.Text()
		// Truncate very long lines
		if len(line) > 100 {
			line = line[:97] + "..."
		}

		fmt.Printf("%3d: %s\n", lineNum, line)
		lineNum++
	}

	if lineNum > maxLines {
		fmt.Println(cui.formatMessage("... (truncated)", color.FgYellow))
	}

	return scanner.Err()
}

// clearScreen clears the terminal screen.
func (cui *ConflictUI) clearScreen() {
	if cui.colorEnabled {
		fmt.Print("\033[2J\033[H")
	} else {
		// Fallback: print some newlines
		fmt.Print(strings.Repeat("\n", 3))
	}
}

// formatMessage applies color formatting if enabled.
func (cui *ConflictUI) formatMessage(text string, colorAttr color.Attribute) string {
	if !cui.colorEnabled {
		return text
	}

	return color.New(colorAttr).Sprint(text)
}

// formatBytes formats byte count in human-readable format.
func (cui *ConflictUI) formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	units := []string{"B", "KB", "MB", "GB"}
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
