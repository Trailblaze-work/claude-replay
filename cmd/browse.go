package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/trailblaze/claude-replay/internal/ui"
)

var browseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Browse projects and sessions interactively",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := ui.NewApp(claudeDir)
		p := tea.NewProgram(app, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("running TUI: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(browseCmd)
}
