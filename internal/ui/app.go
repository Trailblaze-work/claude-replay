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
	claudeDir    string
	width        int
	height       int

	projectList  browse.ProjectListModel
	sessionList  browse.SessionListModel
	replayModel  replay.Model

	currentProject session.Project

	err error
}

// NewApp creates the top-level application model.
func NewApp(claudeDir string) AppModel {
	return AppModel{
		claudeDir: claudeDir,
		screen:    ScreenProjects,
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
		return m, m.loadSession(msg.Session.Path)

	case browse.GoBack:
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
		projects, err := session.DiscoverProjects(m.claudeDir)
		return projectsLoadedMsg{projects: projects, err: err}
	}
}

func (m AppModel) loadSessions(dirPath string) tea.Cmd {
	return func() tea.Msg {
		sessions, err := session.DiscoverSessions(dirPath)
		return sessionsLoadedMsg{sessions: sessions, err: err}
	}
}

func (m AppModel) loadSession(path string) tea.Cmd {
	return func() tea.Msg {
		sess, err := session.LoadSession(path)
		return sessionLoadedMsg{session: sess, err: err}
	}
}
