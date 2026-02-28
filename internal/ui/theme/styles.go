package theme

import "github.com/charmbracelet/lipgloss"

// Claude Code color palette
var (
	ColorPrimary    = lipgloss.Color("#D4A574") // Claude's warm amber
	ColorSecondary  = lipgloss.Color("#A0A0A0") // Muted gray
	ColorAccent     = lipgloss.Color("#7AA2F7") // Blue accent
	ColorSuccess    = lipgloss.Color("#9ECE6A") // Green
	ColorError      = lipgloss.Color("#F7768E") // Red/pink
	ColorWarning    = lipgloss.Color("#E0AF68") // Yellow/amber
	ColorDim        = lipgloss.Color("#565656") // Dim gray
	ColorBg         = lipgloss.Color("#1A1B26") // Dark background
	ColorBgAlt      = lipgloss.Color("#24283B") // Slightly lighter bg
	ColorText       = lipgloss.Color("#C0CAF5") // Main text
	ColorThinking   = lipgloss.Color("#BB9AF7") // Purple for thinking
	ColorToolUse    = lipgloss.Color("#7DCFFF") // Cyan for tool use
	ColorUser       = lipgloss.Color("#9ECE6A") // Green for user
)

// Styles used throughout the app
var (
	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			PaddingLeft(1)

	StyleHeaderPath = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			PaddingLeft(1)

	StyleStatusBar = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorBgAlt).
			PaddingLeft(1).
			PaddingRight(1)

	StyleStatusKey = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	StyleStatusVal = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	StyleUserMessage = lipgloss.NewStyle().
			Foreground(ColorUser).
			Bold(true).
			PaddingLeft(2)

	StyleUserPrefix = lipgloss.NewStyle().
			Foreground(ColorUser).
			Bold(true)

	StyleAssistantText = lipgloss.NewStyle().
			Foreground(ColorText).
			PaddingLeft(2)

	StyleThinkingHeader = lipgloss.NewStyle().
			Foreground(ColorThinking).
			Italic(true).
			PaddingLeft(2)

	StyleThinkingBody = lipgloss.NewStyle().
			Foreground(ColorDim).
			PaddingLeft(4)

	StyleToolUseHeader = lipgloss.NewStyle().
			Foreground(ColorToolUse).
			Bold(true).
			PaddingLeft(2)

	StyleToolInput = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			PaddingLeft(4)

	StyleToolResult = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			PaddingLeft(4)

	StyleToolError = lipgloss.NewStyle().
			Foreground(ColorError).
			PaddingLeft(4)

	StyleTimeline = lipgloss.NewStyle().
			Foreground(ColorDim)

	StyleTimelineActive = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorDim)

	StyleDivider = lipgloss.NewStyle().
			Foreground(ColorDim)

	StyleListTitle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			PaddingLeft(1)

	StyleListItem = lipgloss.NewStyle().
			Foreground(ColorText)

	StyleListDesc = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	StyleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDim)
)
