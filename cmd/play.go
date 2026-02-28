package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/trailblaze/claude-replay/internal/session"
	"github.com/trailblaze/claude-replay/internal/ui/replay"
)

// replayWrapper wraps replay.Model to implement tea.Model.
type replayWrapper struct {
	model replay.Model
}

func (w replayWrapper) Init() tea.Cmd {
	return w.model.Init()
}

func (w replayWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	w.model, cmd = w.model.Update(msg)
	return w, cmd
}

func (w replayWrapper) View() string {
	return w.model.View()
}

var playCmd = &cobra.Command{
	Use:   "play <session>",
	Short: "Replay a specific session",
	Long:  "Replay a session by UUID, slug, or file path",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		path, err := session.FindSessionByID(claudeDir, query)
		if err != nil {
			return fmt.Errorf("finding session: %w", err)
		}

		sess, err := session.LoadSession(path)
		if err != nil {
			return fmt.Errorf("loading session: %w", err)
		}

		if len(sess.Turns) == 0 {
			return fmt.Errorf("session has no turns")
		}

		model := replay.New(sess, 120, 40)
		p := tea.NewProgram(replayWrapper{model: model}, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("running replay: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(playCmd)
}
