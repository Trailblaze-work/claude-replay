package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var claudeDir string

var rootCmd = &cobra.Command{
	Use:   "claude-replay",
	Short: "Browse and replay Claude Code sessions",
	Long:  "A TUI tool to browse all Claude Code projects/sessions and replay them in a terminal interface that mimics Claude Code's look and feel.",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	home, _ := os.UserHomeDir()
	defaultDir := filepath.Join(home, ".claude")

	rootCmd.PersistentFlags().StringVar(&claudeDir, "claude-dir", defaultDir, "path to Claude Code data directory")

	// Default command is browse
	rootCmd.RunE = browseCmd.RunE
}
