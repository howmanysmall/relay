package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	configFile string
	verbose    bool
	dryRun     bool
	workers    int
	bufferSize string
	profile    string
)

var rootCmd = &cobra.Command{
	Use:   "relay",
	Short: "High-performance file mirroring and synchronization tool",
	Long: `Relay is a blazing-fast, cross-platform file mirroring and synchronization utility.
Built for performance and usability, it supports real-time monitoring, conflict resolution,
and beautiful terminal output.

Examples:
  relay mirror ./source ./backup          # One-way mirror
  relay sync ./local ./remote             # Two-way sync
  relay watch --config relay.jsonc        # Watch mode
  relay ./src ./dst --preview             # Preview changes`,
}

// SetVersionInfo sets the version information for the CLI.
func SetVersionInfo(version, buildTime, commit string) {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf(`relay version %s
Build time: %s
Commit: %s
`, version, buildTime, commit))
}

// Execute runs the root command for the relay CLI.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is relay.jsonc)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "preview changes without executing")
	rootCmd.PersistentFlags().IntVar(&workers, "workers", 0, "number of worker goroutines (0 = auto)")
	rootCmd.PersistentFlags().StringVar(&bufferSize, "buffer", "auto", "buffer size for operations")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "default", "configuration profile to use")

	// Version will be set dynamically
}
