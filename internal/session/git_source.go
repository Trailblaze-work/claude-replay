package session

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Trailblaze-work/claude-replay/internal/parser"
)

const gitBranch = "claude-sessions"

// sessionMeta mirrors the .meta.json sidecar files on the claude-sessions branch.
type sessionMeta struct {
	SessionID      string         `json:"session_id"`
	Slug           string         `json:"slug"`
	Started        string         `json:"started"`
	LastUpdated    string         `json:"last_updated"`
	Models         []string       `json:"models"`
	ClientVersion  string         `json:"client_version"`
	GitBranch      string         `json:"git_branch"`
	UserTurns      int            `json:"user_turns"`
	AssistantTurns int            `json:"assistant_turns"`
	ToolsUsed      map[string]int `json:"tools_used"`
	CompressedSize int64          `json:"compressed_size"`
}

// GitSource implements SessionSource by reading from a claude-sessions git branch.
type GitSource struct {
	RepoPath string
}

func (s *GitSource) git(args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", s.RepoPath}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func (s *GitSource) ListProjects() ([]Project, error) {
	// Verify the branch exists
	if _, err := s.git("rev-parse", "--verify", gitBranch); err != nil {
		return nil, fmt.Errorf("branch %q not found: %w", gitBranch, err)
	}

	// Count sessions from ls-tree
	metas, err := s.listMetaFiles()
	if err != nil {
		return nil, err
	}

	var lastUsed time.Time
	for _, m := range metas {
		if t, err := time.Parse(time.RFC3339Nano, m.LastUpdated); err == nil {
			if t.After(lastUsed) {
				lastUsed = t
			}
		}
	}

	repoName := filepath.Base(s.RepoPath)

	return []Project{{
		Name:     repoName,
		Path:     s.RepoPath,
		DirName:  repoName,
		DirPath:  "", // not used for git source
		Sessions: len(metas),
		LastUsed: lastUsed,
	}}, nil
}

func (s *GitSource) ListSessions(_ string) ([]SessionInfo, error) {
	metas, err := s.listMetaFiles()
	if err != nil {
		return nil, err
	}

	var sessions []SessionInfo
	for _, m := range metas {
		model := ""
		if len(m.Models) > 0 {
			model = m.Models[0]
		}
		si := SessionInfo{
			ID:        m.SessionID,
			Slug:      m.Slug,
			Model:     model,
			TurnCount: m.UserTurns,
			FileSize:  m.CompressedSize,
		}
		if m.Started != "" {
			if t, err := time.Parse(time.RFC3339Nano, m.Started); err == nil {
				si.FirstTime = t
			}
		}
		if m.LastUpdated != "" {
			if t, err := time.Parse(time.RFC3339Nano, m.LastUpdated); err == nil {
				si.LastTime = t
			}
		}
		sessions = append(sessions, si)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastTime.After(sessions[j].LastTime)
	})

	return sessions, nil
}

func (s *GitSource) LoadSession(sessionID string) (*Session, error) {
	objPath := fmt.Sprintf("%s:sessions/%s.jsonl.gz", gitBranch, sessionID)
	data, err := s.git("show", objPath)
	if err != nil {
		return nil, fmt.Errorf("reading session %s from git: %w", sessionID, err)
	}

	// Decompress gzip
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decompressing session %s: %w", sessionID, err)
	}
	defer gz.Close()

	// Parse JSONL
	records, err := parser.Parse(gz)
	if err != nil {
		return nil, fmt.Errorf("parsing session %s: %w", sessionID, err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("empty session: %s", sessionID)
	}

	sess := &Session{ID: sessionID}
	turns := segmentTurns(records, sess)
	sess.Turns = turns

	if len(turns) > 0 {
		sess.StartTime = turns[0].Timestamp
		sess.EndTime = turns[len(turns)-1].Timestamp
	}

	return sess, nil
}

func (s *GitSource) FindSession(query string) (*SessionInfo, error) {
	sessions, err := s.ListSessions("")
	if err != nil {
		return nil, err
	}

	// Exact ID match
	for i := range sessions {
		if sessions[i].ID == query {
			return &sessions[i], nil
		}
	}

	// Prefix match on ID
	for i := range sessions {
		if strings.HasPrefix(sessions[i].ID, query) {
			return &sessions[i], nil
		}
	}

	// Slug match
	for i := range sessions {
		if sessions[i].Slug == query {
			return &sessions[i], nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", query)
}

// listMetaFiles reads all .meta.json files from the claude-sessions branch.
func (s *GitSource) listMetaFiles() ([]sessionMeta, error) {
	// List all files under sessions/
	out, err := s.git("ls-tree", "--name-only", gitBranch, "sessions/")
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var metas []sessionMeta

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasSuffix(line, ".meta.json") {
			continue
		}

		objPath := fmt.Sprintf("%s:%s", gitBranch, line)
		data, err := s.git("show", objPath)
		if err != nil {
			continue
		}

		var m sessionMeta
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}

		metas = append(metas, m)
	}

	return metas, nil
}
