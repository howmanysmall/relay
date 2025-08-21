package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	preferLocal  bool
	preferRemote bool
	ask          bool
	backup       bool
)

var syncCmd = &cobra.Command{
	Use:   "sync <path1> <path2>",
	Short: "Two-way synchronization between two directories",
	Long: `Synchronize files between two directories in both directions.
Changes in either directory will be reflected in the other, with
intelligent conflict resolution.

Examples:
  relay sync ./local ./remote             # Basic two-way sync
  relay sync ./a ./b --prefer-local       # Local changes win conflicts
  relay sync ./a ./b --ask                # Interactive conflict resolution
  relay sync ./docs ./backup --backup     # Create backups before overwriting`,
	Args: cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		path1, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("invalid path1: %w", err)
		}

		path2, err := filepath.Abs(args[1])
		if err != nil {
			return fmt.Errorf("invalid path2: %w", err)
		}

		fmt.Printf("🔄 Relay Sync\n")
		fmt.Printf("Path 1:      %s\n", path1)
		fmt.Printf("Path 2:      %s\n", path2)
		fmt.Printf("Mode:        Two-way synchronization\n")

		if preferLocal {
			fmt.Printf("Conflicts:   Prefer local changes\n")
		} else if preferRemote {
			fmt.Printf("Conflicts:   Prefer remote changes\n")
		} else if ask {
			fmt.Printf("Conflicts:   Interactive resolution\n")
		} else {
			fmt.Printf("Conflicts:   Smart resolution (newest)\n")
		}

		if dryRun {
			fmt.Printf("Status:      Dry run (preview mode)\n")
		}

		// TODO: Implement sync logic
		return fmt.Errorf("sync functionality not yet implemented")
	},
}

func init() {
	syncCmd.Flags().BoolVar(&preferLocal, "prefer-local", false, "prefer local files in conflicts")
	syncCmd.Flags().BoolVar(&preferRemote, "prefer-remote", false, "prefer remote files in conflicts")
	syncCmd.Flags().BoolVar(&ask, "ask", false, "interactive conflict resolution")
	syncCmd.Flags().BoolVar(&backup, "backup", false, "create backups before overwriting")

	rootCmd.AddCommand(syncCmd)
}
