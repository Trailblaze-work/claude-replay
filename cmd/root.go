package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/Trailblaze-work/claude-replay/internal/session"
)

var (
	claudeDir string
	gitMode   bool
	gitRepo   string
)

// source is the session source used by all subcommands.
// Initialized in the root PersistentPreRunE.
var source session.SessionSource

var rootCmd = &cobra.Command{
	Use:   "claude-replay",
	Short: "Browse and replay Claude Code sessions",
	Long:  "A TUI tool to browse all Claude Code projects/sessions and replay them in a terminal interface that mimics Claude Code's look and feel.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if gitMode {
			repo := gitRepo
			if repo == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("getting current directory: %w", err)
				}
				repo = cwd
			}
			source = &session.GitSource{RepoPath: repo}
		} else {
			source = &session.LocalSource{ClaudeDir: claudeDir}
		}
		return nil
	},
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
	rootCmd.PersistentFlags().BoolVar(&gitMode, "git", false, "browse sessions from a claude-sessions git branch")
	rootCmd.PersistentFlags().StringVar(&gitRepo, "git-repo", "", "path to git repository (default: current directory)")

	// Default command is browse
	rootCmd.RunE = browseCmd.RunE
}
