package session

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/trailblaze/claude-replay/internal/parser"
)

// Project represents a Claude Code project directory.
type Project struct {
	Name      string    // Display name (decoded from directory name)
	Path      string    // Original path the project was for
	DirName   string    // Raw directory name
	DirPath   string    // Full path to the project directory
	Sessions  int       // Number of session files
	LastUsed  time.Time // Most recent session modification
}

// SessionInfo holds metadata about a session file without fully parsing it.
type SessionInfo struct {
	ID        string
	Path      string    // Full path to the JSONL file
	Slug      string
	Model     string
	TurnCount int
	FirstTime time.Time
	LastTime  time.Time
	FileSize  int64
}

// DiscoverProjects finds all Claude Code projects in the given claude directory.
func DiscoverProjects(claudeDir string) ([]Project, error) {
	projectsDir := filepath.Join(claudeDir, "projects")

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, fmt.Errorf("reading projects directory: %w", err)
	}

	var projects []Project
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(projectsDir, entry.Name())
		sessions, lastUsed := countSessions(dirPath)
		if sessions == 0 {
			continue
		}

		projects = append(projects, Project{
			Name:     decodeDirName(entry.Name()),
			Path:     decodeDirPath(entry.Name()),
			DirName:  entry.Name(),
			DirPath:  dirPath,
			Sessions: sessions,
			LastUsed: lastUsed,
		})
	}

	// Sort by last used (most recent first)
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].LastUsed.After(projects[j].LastUsed)
	})

	return projects, nil
}

// DiscoverSessions finds all session files in a project directory.
func DiscoverSessions(projectDir string) ([]SessionInfo, error) {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, fmt.Errorf("reading project directory: %w", err)
	}

	var sessions []SessionInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		path := filepath.Join(projectDir, entry.Name())
		id := strings.TrimSuffix(entry.Name(), ".jsonl")

		info, err := entry.Info()
		if err != nil {
			continue
		}

		slug, model, firstTime, lastTime, turnCount, err := parser.QuickScan(path)
		if err != nil || turnCount == 0 {
			continue
		}

		si := SessionInfo{
			ID:        id,
			Path:      path,
			Slug:      slug,
			Model:     model,
			TurnCount: turnCount,
			FileSize:  info.Size(),
		}

		if firstTime != "" {
			if t, err := time.Parse(time.RFC3339Nano, firstTime); err == nil {
				si.FirstTime = t
			}
		}
		if lastTime != "" {
			if t, err := time.Parse(time.RFC3339Nano, lastTime); err == nil {
				si.LastTime = t
			}
		}

		sessions = append(sessions, si)
	}

	// Sort by last time (most recent first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastTime.After(sessions[j].LastTime)
	})

	return sessions, nil
}

// FindSessionByID searches all projects for a session with the given ID or slug.
func FindSessionByID(claudeDir, query string) (string, error) {
	projectsDir := filepath.Join(claudeDir, "projects")

	// First try direct UUID match
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return "", err
	}

	// Try as a full path
	if _, err := os.Stat(query); err == nil && strings.HasSuffix(query, ".jsonl") {
		return query, nil
	}

	for _, projEntry := range entries {
		if !projEntry.IsDir() {
			continue
		}
		projDir := filepath.Join(projectsDir, projEntry.Name())

		// Try exact UUID match
		candidate := filepath.Join(projDir, query+".jsonl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		// Try prefix UUID match and slug match
		sessEntries, err := os.ReadDir(projDir)
		if err != nil {
			continue
		}
		for _, sessEntry := range sessEntries {
			if sessEntry.IsDir() || !strings.HasSuffix(sessEntry.Name(), ".jsonl") {
				continue
			}
			id := strings.TrimSuffix(sessEntry.Name(), ".jsonl")
			path := filepath.Join(projDir, sessEntry.Name())

			// Prefix match on UUID
			if strings.HasPrefix(id, query) {
				return path, nil
			}
		}
	}

	// Try slug match (slower - needs to scan file content)
	for _, projEntry := range entries {
		if !projEntry.IsDir() {
			continue
		}
		projDir := filepath.Join(projectsDir, projEntry.Name())
		sessEntries, err := os.ReadDir(projDir)
		if err != nil {
			continue
		}
		for _, sessEntry := range sessEntries {
			if sessEntry.IsDir() || !strings.HasSuffix(sessEntry.Name(), ".jsonl") {
				continue
			}
			path := filepath.Join(projDir, sessEntry.Name())
			slug, _, _, _, _, err := parser.QuickScan(path)
			if err == nil && slug == query {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("session not found: %s", query)
}

func countSessions(dirPath string) (int, time.Time) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, time.Time{}
	}

	count := 0
	var latest time.Time
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			count++
			if info, err := entry.Info(); err == nil {
				if info.ModTime().After(latest) {
					latest = info.ModTime()
				}
			}
		}
	}
	return count, latest
}

// decodeDirName converts the hyphen-encoded directory name to a readable name.
// e.g., "-Users-gilles-Documents-trailblaze" -> "trailblaze"
func decodeDirName(dirName string) string {
	path := decodeDirPath(dirName)
	return filepath.Base(path)
}

// decodeDirPath converts the hyphen-encoded directory name back to a path.
// e.g., "-Users-gilles-Documents-trailblaze" -> "/Users/gilles/Documents/trailblaze"
func decodeDirPath(dirName string) string {
	// Replace hyphens with path separators
	// The encoding uses hyphens for path separators
	parts := strings.Split(dirName, "-")
	// Filter empty parts (from leading hyphen)
	var filtered []string
	for _, p := range parts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	return "/" + strings.Join(filtered, "/")
}
