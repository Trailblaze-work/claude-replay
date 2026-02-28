package session

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/trailblaze/claude-replay/internal/parser"
)

// LocalSource implements SessionSource using the local filesystem (~/.claude).
type LocalSource struct {
	ClaudeDir string
}

func (s *LocalSource) ListProjects() ([]Project, error) {
	return DiscoverProjects(s.ClaudeDir)
}

func (s *LocalSource) ListSessions(projectDirPath string) ([]SessionInfo, error) {
	return DiscoverSessions(projectDirPath)
}

func (s *LocalSource) LoadSession(sessionID string) (*Session, error) {
	path, err := FindSessionByID(s.ClaudeDir, sessionID)
	if err != nil {
		return nil, err
	}
	return LoadSession(path)
}

func (s *LocalSource) FindSession(query string) (*SessionInfo, error) {
	path, err := FindSessionByID(s.ClaudeDir, query)
	if err != nil {
		return nil, err
	}

	id := strings.TrimSuffix(filepath.Base(path), ".jsonl")

	// Quick scan for metadata
	slug, model, firstTime, lastTime, turnCount, _ := parser.QuickScan(path)

	info := &SessionInfo{
		ID:        id,
		Path:      path,
		Slug:      slug,
		Model:     model,
		TurnCount: turnCount,
	}

	if firstTime != "" {
		if t, err := time.Parse(time.RFC3339Nano, firstTime); err == nil {
			info.FirstTime = t
		}
	}
	if lastTime != "" {
		if t, err := time.Parse(time.RFC3339Nano, lastTime); err == nil {
			info.LastTime = t
		}
	}

	return info, nil
}
