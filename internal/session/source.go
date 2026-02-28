package session

// SessionSource provides access to Claude Code session data.
// Implementations include LocalSource (filesystem) and GitSource (git branch).
type SessionSource interface {
	// ListProjects returns all available projects.
	ListProjects() ([]Project, error)

	// ListSessions returns sessions for a given project.
	// The projectID parameter is source-specific (directory path for local, ignored for git).
	ListSessions(projectID string) ([]SessionInfo, error)

	// LoadSession loads a full session by its ID (UUID).
	LoadSession(sessionID string) (*Session, error)

	// FindSession searches for a session by query (UUID, UUID prefix, slug, or path).
	FindSession(query string) (*SessionInfo, error)
}
