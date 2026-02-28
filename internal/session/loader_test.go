package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeDirName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"-Users-gilles-Documents-trailblaze", "trailblaze"},
		{"-Users-gilles", "gilles"},
		{"-Users-gilles-Downloads", "Downloads"},
	}

	for _, tt := range tests {
		got := decodeDirName(tt.input)
		if got != tt.expected {
			t.Errorf("decodeDirName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDecodeDirPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"-Users-gilles-Documents-trailblaze", "/Users/gilles/Documents/trailblaze"},
		{"-Users-gilles", "/Users/gilles"},
	}

	for _, tt := range tests {
		got := decodeDirPath(tt.input)
		if got != tt.expected {
			t.Errorf("decodeDirPath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDiscoverProjects(t *testing.T) {
	// Create temp claude dir structure
	dir := t.TempDir()
	projectsDir := filepath.Join(dir, "projects")
	os.MkdirAll(filepath.Join(projectsDir, "-Users-test-project1"), 0755)
	os.MkdirAll(filepath.Join(projectsDir, "-Users-test-project2"), 0755)

	// Add a session file to project1
	sessionContent := `{"type":"user","parentUuid":null,"uuid":"u1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","message":{"role":"user","content":"hello"},"isSidechain":false}
`
	os.WriteFile(
		filepath.Join(projectsDir, "-Users-test-project1", "abc-123.jsonl"),
		[]byte(sessionContent),
		0644,
	)

	// project2 has no sessions - should be excluded
	projects, err := DiscoverProjects(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("expected 1 project (with sessions), got %d", len(projects))
	}

	if projects[0].Name != "project1" {
		t.Errorf("expected project1, got %s", projects[0].Name)
	}
	if projects[0].Sessions != 1 {
		t.Errorf("expected 1 session, got %d", projects[0].Sessions)
	}
}

func TestDiscoverSessions(t *testing.T) {
	dir := t.TempDir()

	content := `{"type":"user","parentUuid":null,"uuid":"u1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","message":{"role":"user","content":"hello"},"slug":"test-slug","isSidechain":false}
{"type":"assistant","parentUuid":"u1","uuid":"a1","sessionId":"s1","timestamp":"2026-02-13T12:00:01.000Z","message":{"model":"claude-opus-4-6","id":"msg_1","role":"assistant","content":[{"type":"text","text":"hi"}]},"isSidechain":false}
`
	os.WriteFile(filepath.Join(dir, "session1.jsonl"), []byte(content), 0644)

	sessions, err := DiscoverSessions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	if s.ID != "session1" {
		t.Errorf("expected session1, got %s", s.ID)
	}
	if s.Slug != "test-slug" {
		t.Errorf("expected test-slug, got %s", s.Slug)
	}
	if s.TurnCount != 1 {
		t.Errorf("expected 1 turn, got %d", s.TurnCount)
	}
}

func TestFindSessionByID(t *testing.T) {
	dir := t.TempDir()
	projectsDir := filepath.Join(dir, "projects", "-Users-test-proj")
	os.MkdirAll(projectsDir, 0755)

	content := `{"type":"user","parentUuid":null,"uuid":"u1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","message":{"role":"user","content":"hello"},"slug":"my-session","isSidechain":false}
`
	sessionFile := filepath.Join(projectsDir, "abcd1234-5678-9012-3456-789012345678.jsonl")
	os.WriteFile(sessionFile, []byte(content), 0644)

	// Test exact UUID
	path, err := FindSessionByID(dir, "abcd1234-5678-9012-3456-789012345678")
	if err != nil {
		t.Fatalf("exact UUID: %v", err)
	}
	if path != sessionFile {
		t.Errorf("exact UUID: got %s", path)
	}

	// Test prefix UUID
	path, err = FindSessionByID(dir, "abcd1234")
	if err != nil {
		t.Fatalf("prefix UUID: %v", err)
	}
	if path != sessionFile {
		t.Errorf("prefix UUID: got %s", path)
	}

	// Test slug match
	path, err = FindSessionByID(dir, "my-session")
	if err != nil {
		t.Fatalf("slug match: %v", err)
	}
	if path != sessionFile {
		t.Errorf("slug match: got %s", path)
	}

	// Test not found
	_, err = FindSessionByID(dir, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

// --- countSessions tests ---

func TestCountSessions_Empty(t *testing.T) {
	dir := t.TempDir()
	count, latest := countSessions(dir)
	if count != 0 {
		t.Errorf("count: got %d, want 0", count)
	}
	if !latest.IsZero() {
		t.Errorf("latest: got %v, want zero time", latest)
	}
}

func TestCountSessions_MixedFiles(t *testing.T) {
	dir := t.TempDir()
	// Create .jsonl files (should be counted)
	os.WriteFile(filepath.Join(dir, "session1.jsonl"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "session2.jsonl"), []byte("{}"), 0644)
	// Create non-jsonl files (should be ignored)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0644)
	os.WriteFile(filepath.Join(dir, "data.json"), []byte("{}"), 0644)
	// Create a subdirectory (should be ignored)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)

	count, _ := countSessions(dir)
	if count != 2 {
		t.Errorf("count: got %d, want 2", count)
	}
}

// --- FindSessionByID edge cases ---

func TestFindSessionByID_DirectPath(t *testing.T) {
	dir := t.TempDir()
	// FindSessionByID reads the projects dir even for direct paths,
	// so we need it to exist
	os.MkdirAll(filepath.Join(dir, "projects"), 0755)

	path := filepath.Join(dir, "direct.jsonl")
	os.WriteFile(path, []byte(`{"type":"user"}`), 0644)

	got, err := FindSessionByID(dir, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != path {
		t.Errorf("got %q, want %q", got, path)
	}
}

func TestFindSessionByID_NoProjects(t *testing.T) {
	dir := t.TempDir()
	// No "projects" subdirectory exists
	_, err := FindSessionByID(dir, "some-uuid")
	if err == nil {
		t.Error("expected error when projects dir missing")
	}
}

// --- LocalSource tests ---

func TestLocalSource_ListProjects(t *testing.T) {
	dir := t.TempDir()
	projectsDir := filepath.Join(dir, "projects", "-Users-test-myproject")
	os.MkdirAll(projectsDir, 0755)

	content := `{"type":"user","parentUuid":null,"uuid":"u1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","message":{"role":"user","content":"hello"},"isSidechain":false}
`
	os.WriteFile(filepath.Join(projectsDir, "sess-1.jsonl"), []byte(content), 0644)

	src := &LocalSource{ClaudeDir: dir}
	projects, err := src.ListProjects()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].Name != "myproject" {
		t.Errorf("name: got %q, want %q", projects[0].Name, "myproject")
	}
}

func TestLocalSource_FindSession(t *testing.T) {
	dir := t.TempDir()
	projectsDir := filepath.Join(dir, "projects", "-Users-test-proj")
	os.MkdirAll(projectsDir, 0755)

	content := `{"type":"user","parentUuid":null,"uuid":"u1","sessionId":"s1","timestamp":"2026-02-13T12:00:00.000Z","message":{"role":"user","content":"hello"},"slug":"find-me","isSidechain":false}
{"type":"assistant","parentUuid":"u1","uuid":"a1","sessionId":"s1","timestamp":"2026-02-13T12:00:01.000Z","message":{"model":"claude-opus-4-6","id":"msg_1","role":"assistant","content":[{"type":"text","text":"hi"}]},"isSidechain":false}
`
	os.WriteFile(filepath.Join(projectsDir, "abcd-1234.jsonl"), []byte(content), 0644)

	src := &LocalSource{ClaudeDir: dir}
	info, err := src.FindSession("abcd-1234")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ID != "abcd-1234" {
		t.Errorf("ID: got %q, want %q", info.ID, "abcd-1234")
	}
	if info.Slug != "find-me" {
		t.Errorf("Slug: got %q, want %q", info.Slug, "find-me")
	}
	if info.TurnCount != 1 {
		t.Errorf("TurnCount: got %d, want 1", info.TurnCount)
	}
}
