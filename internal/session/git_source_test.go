package session

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// setupTestGitRepo creates a temporary git repo with a claude-sessions branch
// containing test session data.
func setupTestGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	// Initialize repo with an initial commit on main
	run("init", "-b", "main")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644)
	run("add", "README.md")
	run("commit", "-m", "initial")

	// Create orphan claude-sessions branch
	run("checkout", "--orphan", "claude-sessions")
	run("rm", "-rf", ".")

	// Create sessions directory
	sessionsDir := filepath.Join(dir, "sessions")
	os.MkdirAll(sessionsDir, 0755)

	sessionID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	startTime := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	// Create .meta.json
	meta := sessionMeta{
		SessionID: sessionID,
		Slug:      "test-session",
		Model:     "claude-sonnet-4-20250514",
		StartTime: startTime.Format(time.RFC3339Nano),
		EndTime:   endTime.Format(time.RFC3339Nano),
		TurnCount: 2,
		ToolsUsed: []string{"Bash", "Read"},
		FileSize:  1234,
	}
	metaJSON, _ := json.Marshal(meta)
	os.WriteFile(filepath.Join(sessionsDir, sessionID+".meta.json"), metaJSON, 0644)

	// Create .jsonl.gz with test session data
	var jsonlBuf bytes.Buffer
	records := []map[string]interface{}{
		{
			"type":      "user",
			"sessionId": sessionID,
			"slug":      "test-session",
			"timestamp": startTime.Format(time.RFC3339Nano),
			"message":   map[string]interface{}{"role": "user", "content": "Hello, what is 2+2?"},
		},
		{
			"type":      "assistant",
			"sessionId": sessionID,
			"timestamp": startTime.Add(time.Second).Format(time.RFC3339Nano),
			"message": map[string]interface{}{
				"role":  "assistant",
				"model": "claude-sonnet-4-20250514",
				"content": []map[string]interface{}{
					{"type": "text", "text": "2+2 equals 4."},
				},
			},
		},
		{
			"type":      "user",
			"sessionId": sessionID,
			"timestamp": startTime.Add(2 * time.Minute).Format(time.RFC3339Nano),
			"message":   map[string]interface{}{"role": "user", "content": "Thanks!"},
		},
		{
			"type":      "assistant",
			"sessionId": sessionID,
			"timestamp": startTime.Add(2*time.Minute + time.Second).Format(time.RFC3339Nano),
			"message": map[string]interface{}{
				"role":  "assistant",
				"model": "claude-sonnet-4-20250514",
				"content": []map[string]interface{}{
					{"type": "text", "text": "You're welcome!"},
				},
			},
		},
	}
	for _, rec := range records {
		line, _ := json.Marshal(rec)
		jsonlBuf.Write(line)
		jsonlBuf.WriteByte('\n')
	}

	var gzBuf bytes.Buffer
	gz := gzip.NewWriter(&gzBuf)
	gz.Write(jsonlBuf.Bytes())
	gz.Close()
	os.WriteFile(filepath.Join(sessionsDir, sessionID+".jsonl.gz"), gzBuf.Bytes(), 0644)

	// Add a second session
	sessionID2 := "11111111-2222-3333-4444-555555555555"
	meta2 := sessionMeta{
		SessionID: sessionID2,
		Slug:      "second-session",
		Model:     "claude-opus-4-20250514",
		StartTime: endTime.Add(time.Hour).Format(time.RFC3339Nano),
		EndTime:   endTime.Add(2 * time.Hour).Format(time.RFC3339Nano),
		TurnCount: 1,
		ToolsUsed: []string{"Write"},
		FileSize:  567,
	}
	metaJSON2, _ := json.Marshal(meta2)
	os.WriteFile(filepath.Join(sessionsDir, sessionID2+".meta.json"), metaJSON2, 0644)

	records2 := []map[string]interface{}{
		{
			"type":      "user",
			"sessionId": sessionID2,
			"slug":      "second-session",
			"timestamp": endTime.Add(time.Hour).Format(time.RFC3339Nano),
			"message":   map[string]interface{}{"role": "user", "content": "Write a file"},
		},
		{
			"type":      "assistant",
			"sessionId": sessionID2,
			"timestamp": endTime.Add(time.Hour + time.Second).Format(time.RFC3339Nano),
			"message": map[string]interface{}{
				"role":  "assistant",
				"model": "claude-opus-4-20250514",
				"content": []map[string]interface{}{
					{"type": "text", "text": "Done."},
				},
			},
		},
	}
	var jsonl2 bytes.Buffer
	for _, rec := range records2 {
		line, _ := json.Marshal(rec)
		jsonl2.Write(line)
		jsonl2.WriteByte('\n')
	}
	var gz2Buf bytes.Buffer
	gz2 := gzip.NewWriter(&gz2Buf)
	gz2.Write(jsonl2.Bytes())
	gz2.Close()
	os.WriteFile(filepath.Join(sessionsDir, sessionID2+".jsonl.gz"), gz2Buf.Bytes(), 0644)

	// Commit everything
	run("add", "sessions/")
	run("commit", "-m", "add sessions")

	// Switch back to main
	run("checkout", "main")

	return dir
}

func TestGitSource_ListProjects(t *testing.T) {
	repo := setupTestGitRepo(t)
	src := &GitSource{RepoPath: repo}

	projects, err := src.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}

	p := projects[0]
	if p.Sessions != 2 {
		t.Errorf("expected 2 sessions, got %d", p.Sessions)
	}
	if p.LastUsed.IsZero() {
		t.Error("expected non-zero LastUsed")
	}
}

func TestGitSource_ListSessions(t *testing.T) {
	repo := setupTestGitRepo(t)
	src := &GitSource{RepoPath: repo}

	sessions, err := src.ListSessions("")
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// Should be sorted by last time (most recent first)
	if sessions[0].Slug != "second-session" {
		t.Errorf("expected second-session first (most recent), got %s", sessions[0].Slug)
	}
	if sessions[1].Slug != "test-session" {
		t.Errorf("expected test-session second, got %s", sessions[1].Slug)
	}

	// Check metadata populated correctly
	s := sessions[1] // test-session
	if s.ID != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Errorf("unexpected ID: %s", s.ID)
	}
	if s.Model != "claude-sonnet-4-20250514" {
		t.Errorf("unexpected model: %s", s.Model)
	}
	if s.TurnCount != 2 {
		t.Errorf("expected 2 turns, got %d", s.TurnCount)
	}
	if s.FileSize != 1234 {
		t.Errorf("expected file size 1234, got %d", s.FileSize)
	}
}

func TestGitSource_FindSession_ExactID(t *testing.T) {
	repo := setupTestGitRepo(t)
	src := &GitSource{RepoPath: repo}

	info, err := src.FindSession("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err != nil {
		t.Fatalf("FindSession: %v", err)
	}
	if info.Slug != "test-session" {
		t.Errorf("expected test-session, got %s", info.Slug)
	}
}

func TestGitSource_FindSession_Prefix(t *testing.T) {
	repo := setupTestGitRepo(t)
	src := &GitSource{RepoPath: repo}

	info, err := src.FindSession("aaaaaaaa")
	if err != nil {
		t.Fatalf("FindSession prefix: %v", err)
	}
	if info.Slug != "test-session" {
		t.Errorf("expected test-session, got %s", info.Slug)
	}
}

func TestGitSource_FindSession_Slug(t *testing.T) {
	repo := setupTestGitRepo(t)
	src := &GitSource{RepoPath: repo}

	info, err := src.FindSession("second-session")
	if err != nil {
		t.Fatalf("FindSession slug: %v", err)
	}
	if info.ID != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("unexpected ID: %s", info.ID)
	}
}

func TestGitSource_FindSession_NotFound(t *testing.T) {
	repo := setupTestGitRepo(t)
	src := &GitSource{RepoPath: repo}

	_, err := src.FindSession("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestGitSource_LoadSession(t *testing.T) {
	repo := setupTestGitRepo(t)
	src := &GitSource{RepoPath: repo}

	sess, err := src.LoadSession("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}

	if len(sess.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(sess.Turns))
	}

	if sess.Turns[0].UserText != "Hello, what is 2+2?" {
		t.Errorf("unexpected turn 1 user text: %s", sess.Turns[0].UserText)
	}
	if sess.Turns[1].UserText != "Thanks!" {
		t.Errorf("unexpected turn 2 user text: %s", sess.Turns[1].UserText)
	}

	// Check assistant response blocks
	if len(sess.Turns[0].Blocks) == 0 {
		t.Fatal("expected blocks in turn 1")
	}
	if sess.Turns[0].Blocks[0].Text != "2+2 equals 4." {
		t.Errorf("unexpected block text: %s", sess.Turns[0].Blocks[0].Text)
	}

	if sess.Model != "claude-sonnet-4-20250514" {
		t.Errorf("unexpected model: %s", sess.Model)
	}
}

func TestGitSource_LoadSession_NotFound(t *testing.T) {
	repo := setupTestGitRepo(t)
	src := &GitSource{RepoPath: repo}

	_, err := src.LoadSession("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestGitSource_NoBranch(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "-C", dir, "init", "-b", "main")
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	src := &GitSource{RepoPath: dir}
	_, err := src.ListProjects()
	if err == nil {
		t.Fatal("expected error when claude-sessions branch does not exist")
	}
}
