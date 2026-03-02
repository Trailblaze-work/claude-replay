package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/Trailblaze-work/claude-replay/internal/ui/theme"
)

// RenderHeader renders the top header bar.
func RenderHeader(slug, projectPath, gitBranch string, width int) string {
	title := lipgloss.NewStyle().
		Foreground(theme.ColorPrimary).
		Bold(true).
		Render("> claude-replay")

	slugText := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Render("  " + slug)

	line1 := title + slugText

	pathText := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		PaddingLeft(1).
		Render(projectPath)

	branchText := ""
	if gitBranch != "" && gitBranch != "HEAD" {
		branchText = lipgloss.NewStyle().
			Foreground(theme.ColorSuccess).
			Render("  " + gitBranch)
	}

	line2 := pathText + branchText

	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(theme.ColorDim).
		Width(width)

	return border.Render(fmt.Sprintf("%s\n%s", line1, line2))
}
