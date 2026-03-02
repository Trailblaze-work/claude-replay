package replay

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/lipgloss"
)

// syntaxColors maps chroma token types to foreground colors matching
// Claude Code's diff highlighting: keywords magenta, strings green,
// comments grey, everything else uses the default diff foreground.
var syntaxColors = map[chroma.TokenType]lipgloss.Color{
	chroma.Keyword:            lipgloss.Color("#C678DD"), // magenta/pink
	chroma.KeywordDeclaration: lipgloss.Color("#C678DD"),
	chroma.KeywordNamespace:   lipgloss.Color("#C678DD"),
	chroma.KeywordType:        lipgloss.Color("#C678DD"),
	chroma.KeywordConstant:    lipgloss.Color("#D19A66"), // amber
	chroma.LiteralString:      lipgloss.Color("#98C379"), // green
	chroma.LiteralNumber:      lipgloss.Color("#D19A66"), // amber
	chroma.Comment:            lipgloss.Color("#5C6370"), // grey
}

// tokenColor returns the foreground color for a chroma token type,
// walking up the type hierarchy to find a match.
func tokenColor(tt chroma.TokenType) lipgloss.Color {
	for t := tt; t > 0; t = t.Parent() {
		if c, ok := syntaxColors[t]; ok {
			return c
		}
	}
	return ""
}

// getLexer returns a chroma lexer for the given file path, or nil.
func getLexer(filePath string) chroma.Lexer {
	lexer := lexers.Match(filePath)
	if lexer == nil {
		return nil
	}
	return chroma.Coalesce(lexer)
}

// highlightDiffLine renders a diff line with syntax highlighting.
// lexer may be nil, in which case no syntax highlighting is applied.
// prefix is "- ", "+ ", or "  ". bg is the line background color.
// defaultFg is used for tokens without syntax highlighting.
func highlightDiffLine(prefix, text string, lexer chroma.Lexer, bg, defaultFg lipgloss.Color, totalWidth int) string {
	fallback := func() string {
		return lipgloss.NewStyle().
			Foreground(defaultFg).
			Background(bg).
			Width(totalWidth).
			Render(prefix + text)
	}

	if lexer == nil {
		return fallback()
	}

	iterator, err := lexer.Tokenise(nil, text)
	if err != nil {
		return fallback()
	}

	var result strings.Builder

	// Render prefix with default color
	pStyle := lipgloss.NewStyle().Foreground(defaultFg).Background(bg)
	result.WriteString(pStyle.Render(prefix))

	for _, token := range iterator.Tokens() {
		val := strings.TrimRight(token.Value, "\n\r")
		if val == "" {
			continue
		}
		fg := tokenColor(token.Type)
		style := lipgloss.NewStyle().Background(bg)
		if fg != "" {
			style = style.Foreground(fg)
		} else {
			style = style.Foreground(defaultFg)
		}
		result.WriteString(style.Render(val))
	}

	// Pad to totalWidth with background color
	str := result.String()
	visWidth := lipgloss.Width(str)
	if visWidth < totalWidth {
		pad := strings.Repeat(" ", totalWidth-visWidth)
		str += lipgloss.NewStyle().Background(bg).Render(pad)
	}

	return str
}
