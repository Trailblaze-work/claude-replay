package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/Trailblaze-work/claude-replay/internal/ui/theme"
)

// RenderStatusBar renders the bottom status bar.
func RenderStatusBar(turnNum, totalTurns int, model string, duration time.Duration, timestamp time.Time, width int) string {
	turnInfo := lipgloss.NewStyle().
		Foreground(theme.ColorPrimary).
		Bold(true).
		Render(fmt.Sprintf("Turn %d/%d", turnNum, totalTurns))

	modelInfo := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Render(formatModelShort(model))

	durationInfo := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Render(formatDuration(duration))

	timeInfo := lipgloss.NewStyle().
		Foreground(theme.ColorDim).
		Render(timestamp.Format("Jan 02 15:04"))

	sep := lipgloss.NewStyle().
		Foreground(theme.ColorDim).
		Render("  │  ")

	content := turnInfo + sep + modelInfo + sep + durationInfo + sep + timeInfo

	bar := lipgloss.NewStyle().
		Background(theme.ColorBgAlt).
		Width(width).
		PaddingLeft(1).
		PaddingRight(1)

	return bar.Render(content)
}

// RenderTimeline renders the visual timeline scrubber.
func RenderTimeline(current, total, width int) string {
	if total <= 0 {
		return ""
	}

	prefix := " ◀◀  ◀ "
	suffix := " ▶  ▶▶ "
	barWidth := width - len(prefix) - len(suffix) - 4
	if barWidth < 10 {
		barWidth = 10
	}

	filled := 0
	if total > 1 {
		filled = (current - 1) * barWidth / (total - 1)
	}
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	left := lipgloss.NewStyle().Foreground(theme.ColorDim).Render(prefix)
	activeBar := lipgloss.NewStyle().Foreground(theme.ColorPrimary).Render(bar)
	right := lipgloss.NewStyle().Foreground(theme.ColorDim).Render(suffix)

	return left + activeBar + right
}

func formatModelShort(model string) string {
	switch {
	case strings.Contains(model, "opus-4-6"):
		return "opus-4.6"
	case strings.Contains(model, "opus"):
		return "opus"
	case strings.Contains(model, "sonnet-4-6"):
		return "sonnet-4.6"
	case strings.Contains(model, "sonnet"):
		return "sonnet"
	case strings.Contains(model, "haiku"):
		return "haiku"
	default:
		if len(model) > 20 {
			return model[:20] + "…"
		}
		return model
	}
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "—"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}
