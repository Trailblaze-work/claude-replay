package browse

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/trailblaze/claude-replay/internal/session"
	"github.com/trailblaze/claude-replay/internal/ui/theme"
)

// SessionSelected is sent when a session is chosen.
type SessionSelected struct {
	Session session.SessionInfo
}

// GoBack signals navigation back to projects.
type GoBack struct{}

// sessionItem wraps a SessionInfo for the list.
type sessionItem struct {
	session session.SessionInfo
}

func (i sessionItem) FilterValue() string {
	return i.session.Slug + " " + i.session.ID + " " + i.session.Model
}

type sessionDelegate struct{}

func (d sessionDelegate) Height() int                             { return 2 }
func (d sessionDelegate) Spacing() int                            { return 1 }
func (d sessionDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d sessionDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(sessionItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	s := item.session

	slug := s.Slug
	if slug == "" {
		slug = s.ID[:8] + "..."
	}
	turns := fmt.Sprintf("%d turns", s.TurnCount)
	model := formatModel(s.Model)
	date := s.LastTime.Format("Jan 02 15:04")
	size := formatSize(s.FileSize)

	var nameStyle, detailStyle lipgloss.Style
	if isSelected {
		nameStyle = lipgloss.NewStyle().Foreground(theme.ColorPrimary).Bold(true).PaddingLeft(2)
		detailStyle = lipgloss.NewStyle().Foreground(theme.ColorSecondary).PaddingLeft(4)
		fmt.Fprintf(w, "%s\n%s",
			nameStyle.Render("> "+slug),
			detailStyle.Render(fmt.Sprintf("%s  ·  %s  ·  %s  ·  %s", model, turns, date, size)),
		)
	} else {
		nameStyle = lipgloss.NewStyle().Foreground(theme.ColorText).PaddingLeft(2)
		detailStyle = lipgloss.NewStyle().Foreground(theme.ColorDim).PaddingLeft(4)
		fmt.Fprintf(w, "%s\n%s",
			nameStyle.Render("  "+slug),
			detailStyle.Render(fmt.Sprintf("%s  ·  %s  ·  %s  ·  %s", model, turns, date, size)),
		)
	}
}

// SessionListModel is the session browser screen.
type SessionListModel struct {
	list        list.Model
	sessions    []session.SessionInfo
	projectName string
	width       int
	height      int
}

// NewSessionList creates a session browser.
func NewSessionList(sessions []session.SessionInfo, projectName string, width, height int) SessionListModel {
	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = sessionItem{session: s}
	}

	delegate := sessionDelegate{}
	l := list.New(items, delegate, width, height-4)
	l.Title = fmt.Sprintf("Sessions — %s", projectName)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = theme.StyleListTitle
	l.SetShowHelp(true)

	return SessionListModel{
		list:        l,
		sessions:    sessions,
		projectName: projectName,
		width:       width,
		height:      height,
	}
}

func (m SessionListModel) Init() tea.Cmd {
	return nil
}

func (m SessionListModel) Update(msg tea.Msg) (SessionListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, theme.DefaultKeyMap.Select):
			if item, ok := m.list.SelectedItem().(sessionItem); ok {
				return m, func() tea.Msg { return SessionSelected{Session: item.session} }
			}
		case key.Matches(msg, theme.DefaultKeyMap.Back):
			return m, func() tea.Msg { return GoBack{} }
		case key.Matches(msg, theme.DefaultKeyMap.Quit):
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-4)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m SessionListModel) View() string {
	return m.list.View()
}

func formatModel(model string) string {
	switch {
	case strings.Contains(model, "opus"):
		return "opus"
	case strings.Contains(model, "sonnet"):
		return "sonnet"
	case strings.Contains(model, "haiku"):
		return "haiku"
	default:
		return model
	}
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.0fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
