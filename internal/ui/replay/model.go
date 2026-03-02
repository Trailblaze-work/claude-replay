package replay

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/Trailblaze-work/claude-replay/internal/session"
	"github.com/Trailblaze-work/claude-replay/internal/ui/components"
	"github.com/Trailblaze-work/claude-replay/internal/ui/theme"
)

// BackToList signals to return to the session list.
type BackToList struct{}

// autoPlayTick is sent during autoplay mode.
type autoPlayTick struct{}

// Model is the replay screen model.
type Model struct {
	session       *session.Session
	currentTurn   int // 0-indexed
	viewport      viewport.Model
	width         int
	height        int
	allExpanded   bool
	showHelp      bool
	autoPlay      bool
	autoPlaySpeed time.Duration
	ready         bool
}

// New creates a new replay model for the given session.
func New(sess *session.Session, width, height int) Model {
	m := Model{
		session:       sess,
		currentTurn:   0,
		width:         width,
		height:        height,
		autoPlaySpeed: 2 * time.Second,
	}
	m.initViewport()
	return m
}

func (m *Model) initViewport() {
	headerHeight := 3
	statusHeight := 3
	contentHeight := m.height - headerHeight - statusHeight
	if contentHeight < 5 {
		contentHeight = 5
	}

	m.viewport = viewport.New(m.width, contentHeight)
	// Add j/k to viewport scroll keys
	m.viewport.KeyMap.Up.SetKeys("up", "k")
	m.viewport.KeyMap.Down.SetKeys("down", "j")
	m.updateContent()
	m.ready = true
}

func (m *Model) updateContent() {
	if len(m.session.Turns) == 0 {
		m.viewport.SetContent("No turns to display")
		return
	}

	turn := m.session.Turns[m.currentTurn]
	content := RenderTurn(turn, m.allExpanded, m.width, m.session.CWD)
	m.viewport.SetContent(content)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		switch {
		case key.Matches(msg, theme.DefaultKeyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, theme.DefaultKeyMap.Back):
			return m, func() tea.Msg { return BackToList{} }

		case key.Matches(msg, theme.DefaultKeyMap.NextTurn):
			if m.currentTurn < len(m.session.Turns)-1 {
				m.currentTurn++
				m.updateContent()
				m.viewport.GotoTop()
			}
		case key.Matches(msg, theme.DefaultKeyMap.PrevTurn):
			if m.currentTurn > 0 {
				m.currentTurn--
				m.updateContent()
				m.viewport.GotoTop()
			}
		case key.Matches(msg, theme.DefaultKeyMap.FirstTurn):
			m.currentTurn = 0
			m.updateContent()
			m.viewport.GotoTop()
		case key.Matches(msg, theme.DefaultKeyMap.LastTurn):
			m.currentTurn = len(m.session.Turns) - 1
			m.updateContent()
			m.viewport.GotoTop()

		case key.Matches(msg, theme.DefaultKeyMap.ExpandTool):
			m.allExpanded = !m.allExpanded
			m.updateContent()

		case key.Matches(msg, theme.DefaultKeyMap.AutoPlay):
			m.autoPlay = !m.autoPlay
			if m.autoPlay {
				return m, m.autoPlayCmd()
			}

		case key.Matches(msg, theme.DefaultKeyMap.SpeedUp):
			if m.autoPlaySpeed > 500*time.Millisecond {
				m.autoPlaySpeed -= 500 * time.Millisecond
			}
		case key.Matches(msg, theme.DefaultKeyMap.SpeedDown):
			m.autoPlaySpeed += 500 * time.Millisecond

		case key.Matches(msg, theme.DefaultKeyMap.Help):
			m.showHelp = !m.showHelp
		}

	case autoPlayTick:
		if !m.autoPlay {
			return m, nil
		}
		if m.currentTurn < len(m.session.Turns)-1 {
			m.currentTurn++
			m.updateContent()
			m.viewport.GotoTop()
			return m, m.autoPlayCmd()
		}
		m.autoPlay = false

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.initViewport()
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) autoPlayCmd() tea.Cmd {
	return tea.Tick(m.autoPlaySpeed, func(time.Time) tea.Msg {
		return autoPlayTick{}
	})
}

func (m Model) View() string {
	if !m.ready || len(m.session.Turns) == 0 {
		return "Loading..."
	}

	if m.showHelp {
		return m.helpView()
	}

	turn := m.session.Turns[m.currentTurn]

	slug := m.session.Slug
	if slug == "" && len(m.session.ID) > 8 {
		slug = m.session.ID[:8]
	}

	header := components.RenderHeader(slug, m.session.CWD, m.session.GitBranch, m.width)
	content := m.viewport.View()
	timeline := components.RenderTimeline(m.currentTurn+1, len(m.session.Turns), m.width)
	status := components.RenderStatusBar(
		m.currentTurn+1,
		len(m.session.Turns),
		turn.Model,
		turn.Duration,
		turn.Timestamp,
		m.width,
	)

	return header + "\n" + content + "\n" + timeline + "\n" + status
}

func (m Model) helpView() string {
	help := `
  Navigation
  ──────────
  ←/h        Previous turn
  →/l        Next turn
  Home/g     First turn
  End/G      Last turn
  ↑/k/↓/j   Scroll
  PgUp/PgDn  Page up/down

  Display
  ───────
  Ctrl+O     Expand/collapse all
  Space      Toggle autoplay
  +/-        Adjust autoplay speed

  General
  ───────
  ?          Toggle help
  Esc        Back to session list
  q          Quit

  Press any key to close help
`
	return theme.StyleBorder.Width(m.width - 4).Render(help)
}
