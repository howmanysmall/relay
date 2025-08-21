// Package main provides the entry point for the relay CLI tool.
package main

import (
	"os"

	"github.com/howmanysmall/relay/src/internal/cli"
)

// Build information. These are set by the build process.
var (
	version   = "dev"
	buildTime = "unknown"
	commit    = "unknown"
)

func main() {
	// Set version information in CLI
	cli.SetVersionInfo(version, buildTime, commit)

	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
