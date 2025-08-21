// Package cli provides command-line commands and helpers for the relay application,
// including subcommands such as mirror which perform file synchronization and
// display-related utilities for interactive and non-interactive modes.
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/howmanysmall/relay/src/internal/core"
	"github.com/howmanysmall/relay/src/internal/display"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	ifNewer  bool
	smart    bool
	turbo    bool
	gentle   bool
	since    string
	filters  []string
	excludes []string
)

var mirrorCmd = &cobra.Command{
	Use:   "mirror <source> <destination>",
	Short: "One-way file mirroring from source to destination",
	Long: `Mirror files from source to destination directory.
This is a one-way operation - files are copied from source to destination,
but changes in destination won't affect source.

Examples:
  relay mirror ./source ./backup          # Basic mirror
  relay mirror ./src ./dst --if-newer     # Only copy newer files
  relay mirror ./project ./backup --smart # Auto-exclude build artifacts
  relay mirror ./src ./dst --turbo        # Maximum performance mode
  relay mirror ./docs ./web --since 1h    # Changes in last hour`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		source, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("invalid source path: %w", err)
		}

		destination, err := filepath.Abs(args[1])
		if err != nil {
			return fmt.Errorf("invalid destination path: %w", err)
		}

		// Determine if we can use interactive UI
		isInteractive := term.IsTerminal(int(os.Stdout.Fd())) && !verbose && !dryRun
		colorEnabled := term.IsTerminal(int(os.Stdout.Fd()))

		statusRenderer := display.NewStatusRenderer(colorEnabled, false)

		// Show banner
		fmt.Println(display.CreateBanner("Relay File Mirroring", colorEnabled))
		fmt.Println()

		statusRenderer.PrintInfo("Starting mirror operation")
		statusRenderer.PrintInfo(fmt.Sprintf("Source: %s", source))
		statusRenderer.PrintInfo(fmt.Sprintf("Destination: %s", destination))
		statusRenderer.PrintInfo("Mode: One-way mirror")

		if dryRun {
			statusRenderer.PrintWarning("Running in dry-run mode (preview only)")
		}
		fmt.Println()

		engine, err := createSyncEngine()
		if err != nil {
			return fmt.Errorf("failed to create sync engine: %w", err)
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		// Start mirror operation with UI
		if isInteractive {
			// Use dashboard for interactive mode
			dashboard := display.NewDashboard(engine, 100*time.Millisecond)

			// Start dashboard in background
			dashCtx, dashCancel := context.WithCancel(ctx)
			go dashboard.Run(dashCtx)

			// Run mirror operation
			err := engine.Mirror(ctx, source, destination)

			// Stop dashboard
			dashCancel()
			time.Sleep(200 * time.Millisecond) // Allow final update

			if err != nil {
				dashboard.ShowError(err)
				return fmt.Errorf("mirror operation failed: %w", err)
			}

			// Show completion summary
			stats := engine.GetStats()
			dashboard.ShowCompletion(stats)
		} else {
			// Use simple progress for non-interactive mode
			statusRenderer.PrintProgress("Starting file scan...")

			err := engine.Mirror(ctx, source, destination)
			if err != nil {
				statusRenderer.PrintError("Mirror operation failed", err.Error())
				return fmt.Errorf("mirror operation failed: %w", err)
			}

			// Show final statistics
			fmt.Println()
			statusRenderer.PrintSuccess("Mirror completed successfully!")
			display.PrintSimpleStats(engine, colorEnabled)
		}

		return nil
	},
}

func init() {
	mirrorCmd.Flags().BoolVar(&ifNewer, "if-newer", false, "only copy files that are newer")
	mirrorCmd.Flags().BoolVar(&smart, "smart", false, "automatically exclude common build artifacts")
	mirrorCmd.Flags().BoolVar(&turbo, "turbo", false, "maximum performance mode")
	mirrorCmd.Flags().BoolVar(&gentle, "gentle", false, "low resource usage mode")
	mirrorCmd.Flags().StringVar(&since, "since", "", "only sync changes since specified time (e.g., '1h', '2d')")
	mirrorCmd.Flags().StringSliceVar(&filters, "include", nil, "include patterns (glob)")
	mirrorCmd.Flags().StringSliceVar(&excludes, "exclude", nil, "exclude patterns (glob)")

	rootCmd.AddCommand(mirrorCmd)
}

func createSyncEngine() (*core.SyncEngine, error) {
	return core.NewSyncEngine()
}
