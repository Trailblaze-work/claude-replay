package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/trailblaze/claude-replay/internal/session"
	"github.com/trailblaze/claude-replay/internal/ui/browse"
	"github.com/trailblaze/claude-replay/internal/ui/replay"
)

// Screen identifies the current UI screen.
type Screen int

const (
	ScreenProjects Screen = iota
	ScreenSessions
	ScreenReplay
)

// AppModel is the top-level Bubble Tea model.
type AppModel struct {
	screen       Screen
	source       session.SessionSource
	skipProjects bool // skip project screen (e.g., git mode with single project)
	width        int
	height       int

	projectList  browse.ProjectListModel
	sessionList  browse.SessionListModel
	replayModel  replay.Model

	currentProject session.Project

	err error
}

// NewApp creates the top-level application model.
func NewApp(source session.SessionSource) AppModel {
	return AppModel{
		source: source,
		screen: ScreenProjects,
	}
}

// NewAppSkipProjects creates an app that skips the project screen
// and goes directly to the session list for the given project.
func NewAppSkipProjects(source session.SessionSource, project session.Project) AppModel {
	return AppModel{
		source:         source,
		screen:         ScreenSessions,
		skipProjects:   true,
		currentProject: project,
	}
}

func (m AppModel) Init() tea.Cmd {
	return nil
}

type projectsLoadedMsg struct {
	projects []session.Project
	err      error
}

type sessionsLoadedMsg struct {
	sessions []session.SessionInfo
	err      error
}

type sessionLoadedMsg struct {
	session *session.Session
	err     error
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		switch m.screen {
		case ScreenProjects:
			return m, m.loadProjects()
		case ScreenSessions:
			if m.skipProjects {
				return m, m.loadSessions(m.currentProject.DirPath)
			}
			m.sessionList, _ = m.sessionList.Update(msg)
		case ScreenReplay:
			var cmd tea.Cmd
			m.replayModel, cmd = m.replayModel.Update(msg)
			return m, cmd
		}
		return m, nil

	case projectsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.projectList = browse.NewProjectList(msg.projects, m.width, m.height)
		return m, nil

	case sessionsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.sessionList = browse.NewSessionList(msg.sessions, m.currentProject.Name, m.width, m.height)
		return m, nil

	case sessionLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.screen = ScreenReplay
		m.replayModel = replay.New(msg.session, m.width, m.height)
		return m, nil

	case browse.ProjectSelected:
		m.currentProject = msg.Project
		m.screen = ScreenSessions
		return m, m.loadSessions(msg.Project.DirPath)

	case browse.SessionSelected:
		return m, m.loadSession(msg.Session.ID)

	case browse.GoBack:
		if m.skipProjects {
			return m, tea.Quit
		}
		m.screen = ScreenProjects
		return m, m.loadProjects()

	case replay.BackToList:
		m.screen = ScreenSessions
		return m, m.loadSessions(m.currentProject.DirPath)
	}

	// Route updates to current screen
	switch m.screen {
	case ScreenProjects:
		var cmd tea.Cmd
		m.projectList, cmd = m.projectList.Update(msg)
		return m, cmd
	case ScreenSessions:
		var cmd tea.Cmd
		m.sessionList, cmd = m.sessionList.Update(msg)
		return m, cmd
	case ScreenReplay:
		var cmd tea.Cmd
		m.replayModel, cmd = m.replayModel.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m AppModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	switch m.screen {
	case ScreenProjects:
		return m.projectList.View()
	case ScreenSessions:
		return m.sessionList.View()
	case ScreenReplay:
		return m.replayModel.View()
	}

	return "Loading..."
}

func (m AppModel) loadProjects() tea.Cmd {
	return func() tea.Msg {
		projects, err := m.source.ListProjects()
		return projectsLoadedMsg{projects: projects, err: err}
	}
}

func (m AppModel) loadSessions(projectID string) tea.Cmd {
	return func() tea.Msg {
		sessions, err := m.source.ListSessions(projectID)
		return sessionsLoadedMsg{sessions: sessions, err: err}
	}
}

func (m AppModel) loadSession(sessionID string) tea.Cmd {
	return func() tea.Msg {
		sess, err := m.source.LoadSession(sessionID)
		return sessionLoadedMsg{session: sess, err: err}
	}
}
