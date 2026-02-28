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
		var app ui.AppModel
		if gitMode {
			// Git mode: skip project browser, go straight to sessions
			projects, err := source.ListProjects()
			if err != nil {
				return fmt.Errorf("listing projects: %w", err)
			}
			if len(projects) == 0 {
				return fmt.Errorf("no sessions found on claude-sessions branch")
			}
			app = ui.NewAppSkipProjects(source, projects[0])
		} else {
			app = ui.NewApp(source)
		}
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
