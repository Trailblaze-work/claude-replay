package replay

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/lipgloss"
)

// syntaxColors maps chroma token types to foreground colors extracted
// from Claude Code's diff rendering, brightened ~20% to match CC's
// vivid appearance on colored diff backgrounds.
var syntaxColors = map[chroma.TokenType]lipgloss.Color{
	// Keywords: purple/magenta
	chroma.Keyword:            lipgloss.Color("#E09EFF"),
	chroma.KeywordDeclaration: lipgloss.Color("#E09EFF"),
	chroma.KeywordNamespace:   lipgloss.Color("#E09EFF"),
	chroma.KeywordType:        lipgloss.Color("#E09EFF"),
	chroma.KeywordReserved:    lipgloss.Color("#E09EFF"),
	chroma.KeywordPseudo:      lipgloss.Color("#E09EFF"),
	chroma.OperatorWord:       lipgloss.Color("#E09EFF"),

	// Strings: warm yellow
	chroma.LiteralString: lipgloss.Color("#FFE4A4"),

	// String escapes: cyan
	chroma.LiteralStringEscape: lipgloss.Color("#80DCE8"),

	// Numbers / constants: amber
	chroma.LiteralNumber:   lipgloss.Color("#F2BE88"),
	chroma.KeywordConstant: lipgloss.Color("#F2BE88"),

	// Function names: blue
	chroma.NameFunction: lipgloss.Color("#8ED0FF"),
	chroma.NameBuiltin:  lipgloss.Color("#8ED0FF"),

	// Comments: grey
	chroma.Comment: lipgloss.Color("#8A949E"),
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
