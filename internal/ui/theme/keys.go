package theme

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings for the application.
type KeyMap struct {
	Quit         key.Binding
	Back         key.Binding
	Select       key.Binding
	NextTurn     key.Binding
	PrevTurn     key.Binding
	FirstTurn    key.Binding
	LastTurn     key.Binding
	ScrollUp     key.Binding
	ScrollDown   key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	ExpandTool   key.Binding
	AutoPlay     key.Binding
	SpeedUp      key.Binding
	SpeedDown    key.Binding
	Help         key.Binding
	Filter       key.Binding
}

// DefaultKeyMap returns the default key bindings.
var DefaultKeyMap = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	NextTurn: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "next turn"),
	),
	PrevTurn: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "prev turn"),
	),
	FirstTurn: key.NewBinding(
		key.WithKeys("home", "g"),
		key.WithHelp("Home/g", "first turn"),
	),
	LastTurn: key.NewBinding(
		key.WithKeys("end", "G"),
		key.WithHelp("End/G", "last turn"),
	),
	ScrollUp: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "prev section"),
	),
	ScrollDown: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "next section"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup", "ctrl+u"),
		key.WithHelp("PgUp", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown", "ctrl+d"),
		key.WithHelp("PgDn", "page down"),
	),
	ExpandTool: key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("ctrl+o", "expand/collapse"),
	),
	AutoPlay: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "autoplay"),
	),
	SpeedUp: key.NewBinding(
		key.WithKeys("+", "="),
		key.WithHelp("+", "speed up"),
	),
	SpeedDown: key.NewBinding(
		key.WithKeys("-"),
		key.WithHelp("-", "slow down"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
}
