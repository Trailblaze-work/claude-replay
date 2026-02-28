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

// ProjectSelected is sent when a project is chosen.
type ProjectSelected struct {
	Project session.Project
}

// projectItem wraps a Project for the list.
type projectItem struct {
	project session.Project
}

func (i projectItem) FilterValue() string {
	return i.project.Name + " " + i.project.Path
}

type projectDelegate struct{}

func (d projectDelegate) Height() int                             { return 2 }
func (d projectDelegate) Spacing() int                            { return 1 }
func (d projectDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d projectDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(projectItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	name := item.project.Name
	path := item.project.Path
	sessions := fmt.Sprintf("%d sessions", item.project.Sessions)
	lastUsed := item.project.LastUsed.Format("Jan 02 15:04")

	var nameStyle, detailStyle lipgloss.Style
	if isSelected {
		nameStyle = lipgloss.NewStyle().Foreground(theme.ColorPrimary).Bold(true).PaddingLeft(2)
		detailStyle = lipgloss.NewStyle().Foreground(theme.ColorSecondary).PaddingLeft(4)
		fmt.Fprintf(w, "%s\n%s",
			nameStyle.Render("> "+name),
			detailStyle.Render(fmt.Sprintf("%s  ·  %s  ·  %s", path, sessions, lastUsed)),
		)
	} else {
		nameStyle = lipgloss.NewStyle().Foreground(theme.ColorText).PaddingLeft(2)
		detailStyle = lipgloss.NewStyle().Foreground(theme.ColorDim).PaddingLeft(4)
		fmt.Fprintf(w, "%s\n%s",
			nameStyle.Render("  "+name),
			detailStyle.Render(fmt.Sprintf("%s  ·  %s  ·  %s", path, sessions, lastUsed)),
		)
	}
}

// ProjectListModel is the project browser screen.
type ProjectListModel struct {
	list     list.Model
	projects []session.Project
	width    int
	height   int
}

// NewProjectList creates a project browser from discovered projects.
func NewProjectList(projects []session.Project, width, height int) ProjectListModel {
	items := make([]list.Item, len(projects))
	for i, p := range projects {
		items[i] = projectItem{project: p}
	}

	delegate := projectDelegate{}
	l := list.New(items, delegate, width, height-4)
	l.Title = "Claude Code Projects"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = theme.StyleListTitle
	l.SetShowHelp(true)

	return ProjectListModel{
		list:     l,
		projects: projects,
		width:    width,
		height:   height,
	}
}

func (m ProjectListModel) Init() tea.Cmd {
	return nil
}

func (m ProjectListModel) Update(msg tea.Msg) (ProjectListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys when filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, theme.DefaultKeyMap.Select):
			if item, ok := m.list.SelectedItem().(projectItem); ok {
				return m, func() tea.Msg { return ProjectSelected{Project: item.project} }
			}
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

func (m ProjectListModel) View() string {
	header := lipgloss.NewStyle().
		Foreground(theme.ColorPrimary).
		Bold(true).
		PaddingLeft(1).
		Render("> claude-replay")

	subtitle := lipgloss.NewStyle().
		Foreground(theme.ColorDim).
		PaddingLeft(1).
		Render(fmt.Sprintf("  %d projects found", len(m.projects)))

	return strings.Join([]string{
		header + subtitle,
		"",
		m.list.View(),
	}, "\n")
}
