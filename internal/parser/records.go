package parser

import (
	"encoding/json"
	"regexp"
	"time"
)

// RecordType identifies the kind of JSONL record.
type RecordType string

const (
	RecordTypeUser     RecordType = "user"
	RecordTypeAssistant RecordType = "assistant"
	RecordTypeSystem   RecordType = "system"
	RecordTypeProgress RecordType = "progress"
	RecordTypeSnapshot RecordType = "file-history-snapshot"
)

// Record is a single line from a Claude Code JSONL session file.
type Record struct {
	Type       RecordType      `json:"type"`
	ParentUUID *string         `json:"parentUuid"`
	UUID       string          `json:"uuid"`
	SessionID  string          `json:"sessionId"`
	Timestamp  time.Time       `json:"timestamp"`
	CWD        string          `json:"cwd"`
	GitBranch  string          `json:"gitBranch"`
	Slug       string          `json:"slug"`
	Version    string          `json:"version"`
	IsSidechain bool           `json:"isSidechain"`
	IsMeta     bool            `json:"isMeta"`

	// User fields
	Message json.RawMessage `json:"message"`

	// System fields
	Subtype    string  `json:"subtype"`
	DurationMs float64 `json:"durationMs"`

	// Thinking metadata (on user records)
	ThinkingMetadata *ThinkingMetadata `json:"thinkingMetadata"`
}

type ThinkingMetadata struct {
	MaxThinkingTokens int `json:"maxThinkingTokens"`
}

// UserMessage is the parsed form of a user record's message field.
type UserMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // string (user text) or array (tool results)
}

// AssistantMessage is the parsed form of an assistant record's message field.
type AssistantMessage struct {
	Model      string         `json:"model"`
	ID         string         `json:"id"`       // message ID, shared across content blocks
	Role       string         `json:"role"`
	Content    []ContentBlock `json:"content"`
	StopReason *string        `json:"stop_reason"`
	Usage      *Usage         `json:"usage"`
}

// ContentBlock is one block in an assistant message content array.
type ContentBlock struct {
	Type string `json:"type"` // "text", "thinking", "tool_use"

	// text block
	Text string `json:"text,omitempty"`

	// thinking block
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`

	// tool_use block
	ID    string          `json:"id,omitempty"`    // tool use ID
	Name  string          `json:"name,omitempty"`  // tool name
	Input json.RawMessage `json:"input,omitempty"` // tool input params
}

// ToolResult appears in user messages when content is an array.
type ToolResult struct {
	Type      string          `json:"type"`        // "tool_result"
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"` // string or array of {type, text}
	IsError   *bool           `json:"is_error,omitempty"`
}

// Usage tracks API token usage.
type Usage struct {
	InputTokens              int    `json:"input_tokens"`
	OutputTokens             int    `json:"output_tokens"`
	CacheCreationInputTokens int    `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int    `json:"cache_read_input_tokens"`
	ServiceTier              string `json:"service_tier"`
}

// ParseUserMessage extracts the UserMessage from a user record.
func (r *Record) ParseUserMessage() (*UserMessage, error) {
	var msg UserMessage
	if err := json.Unmarshal(r.Message, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ParseAssistantMessage extracts the AssistantMessage from an assistant record.
func (r *Record) ParseAssistantMessage() (*AssistantMessage, error) {
	var msg AssistantMessage
	if err := json.Unmarshal(r.Message, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// UserText returns the user's text content if the message content is a string.
// Returns empty string if content is an array (tool results).
func (msg *UserMessage) UserText() string {
	var text string
	if err := json.Unmarshal(msg.Content, &text); err != nil {
		return ""
	}
	return text
}

// IsToolResults returns true if the user message content is a tool result array.
func (msg *UserMessage) IsToolResults() bool {
	return len(msg.Content) > 0 && msg.Content[0] == '['
}

// ParseToolResults extracts tool results from a user message with array content.
func (msg *UserMessage) ParseToolResults() ([]ToolResult, error) {
	var results []ToolResult
	if err := json.Unmarshal(msg.Content, &results); err != nil {
		return nil, err
	}
	return results, nil
}

var commandNameRe = regexp.MustCompile(`<command-name>(/[^<]+)</command-name>`)

// CommandName extracts a slash command name from a command message.
// Returns the command (e.g. "/session-trail:backfill") and true if found,
// or empty string and false otherwise.
func (msg *UserMessage) CommandName() (string, bool) {
	text := msg.UserText()
	if m := commandNameRe.FindStringSubmatch(text); len(m) == 2 {
		return m[1], true
	}
	return "", false
}

var bashInputRe = regexp.MustCompile(`^<bash-input>([\s\S]*)</bash-input>$`)
var bashStdoutRe = regexp.MustCompile(`<bash-stdout>([\s\S]*?)</bash-stdout>`)
var bashStderrRe = regexp.MustCompile(`<bash-stderr>([\s\S]*?)</bash-stderr>`)

// IsBashInput returns true if the message is a shell escape command (!cmd).
func (msg *UserMessage) IsBashInput() bool {
	return bashInputRe.MatchString(msg.UserText())
}

// ParseBashInput extracts the command from a <bash-input> message.
func (msg *UserMessage) ParseBashInput() string {
	if m := bashInputRe.FindStringSubmatch(msg.UserText()); len(m) == 2 {
		return m[1]
	}
	return ""
}

// IsBashOutput returns true if the message contains <bash-stdout> or <bash-stderr> tags.
func (msg *UserMessage) IsBashOutput() bool {
	text := msg.UserText()
	return bashStdoutRe.MatchString(text) || bashStderrRe.MatchString(text)
}

// ParseBashOutput extracts stdout and stderr from a bash output message.
func (msg *UserMessage) ParseBashOutput() (stdout, stderr string) {
	text := msg.UserText()
	if m := bashStdoutRe.FindStringSubmatch(text); len(m) == 2 {
		stdout = m[1]
	}
	if m := bashStderrRe.FindStringSubmatch(text); len(m) == 2 {
		stderr = m[1]
	}
	return
}
