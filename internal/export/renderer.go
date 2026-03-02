package export

import (
	"fmt"
	"strings"

	"github.com/Trailblaze-work/claude-replay/internal/session"
	"github.com/Trailblaze-work/claude-replay/internal/ui/components"
	"github.com/Trailblaze-work/claude-replay/internal/ui/replay"
)

// RenderFrame renders a complete TUI frame for a given turn as a string.
func RenderFrame(sess *session.Session, turnIndex int, width, height int) string {
	if turnIndex < 0 || turnIndex >= len(sess.Turns) {
		return ""
	}

	turn := sess.Turns[turnIndex]

	slug := sess.Slug
	if slug == "" && len(sess.ID) > 8 {
		slug = sess.ID[:8]
	}

	// Header
	header := components.RenderHeader(slug, sess.CWD, sess.GitBranch, width)

	// Content
	content := replay.RenderTurn(turn, false, width, sess.CWD)

	// Ensure content fills available space
	contentLines := strings.Split(content, "\n")
	headerLines := strings.Count(header, "\n") + 1
	statusLines := 2
	availableLines := height - headerLines - statusLines
	if len(contentLines) < availableLines {
		for i := len(contentLines); i < availableLines; i++ {
			contentLines = append(contentLines, "")
		}
	} else if len(contentLines) > availableLines {
		contentLines = contentLines[:availableLines]
	}
	content = strings.Join(contentLines, "\n")

	// Timeline + Status
	timeline := components.RenderTimeline(turnIndex+1, len(sess.Turns), width)
	status := components.RenderStatusBar(
		turnIndex+1,
		len(sess.Turns),
		turn.Model,
		turn.Duration,
		turn.Timestamp,
		width,
	)

	return fmt.Sprintf("%s\n%s\n%s\n%s", header, content, timeline, status)
}
