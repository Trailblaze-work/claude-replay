package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Trailblaze-work/claude-replay/internal/parser"
)

// Turn represents one conversational turn: a user message followed by
// all assistant responses and tool exchanges until the next user text message.
type Turn struct {
	Number    int
	UserText  string           // The user's text message that started this turn
	Blocks    []Block          // All content blocks in this turn (assistant text, thinking, tool_use, tool_result)
	Timestamp time.Time        // Timestamp of the user message
	Duration  time.Duration    // Turn duration from system records
	Model     string           // Model used for this turn
	CWD       string           // Working directory
	GitBranch string           // Git branch
	Slug      string           // Session slug
}

// BlockType identifies what kind of content a block represents.
type BlockType int

const (
	BlockText       BlockType = iota
	BlockThinking
	BlockToolUse
	BlockToolResult
)

// Block is a single renderable piece of content within a turn.
type Block struct {
	Type       BlockType
	Text       string // For text and thinking blocks
	ToolName   string // For tool_use blocks
	ToolInput  map[string]interface{} // Parsed tool input
	ToolID     string // Tool use ID (links tool_use to tool_result)
	IsError    bool   // For tool_result blocks
	RawInput   string // Raw JSON of tool input for display
}

// Session holds all turns parsed from a JSONL file.
type Session struct {
	ID        string
	Slug      string
	Path      string
	Turns     []Turn
	Model     string
	StartTime time.Time
	EndTime   time.Time
	CWD       string
	GitBranch string
	Version   string
}

// LoadSession parses a JSONL file and segments it into turns.
func LoadSession(path string) (*Session, error) {
	records, err := parser.ParseFile(path)
	if err != nil {
		return nil, fmt.Errorf("parsing session file: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("empty session file")
	}

	sess := &Session{Path: path}
	turns := segmentTurns(records, sess)
	sess.Turns = turns

	if len(turns) > 0 {
		sess.StartTime = turns[0].Timestamp
		sess.EndTime = turns[len(turns)-1].Timestamp
	}

	return sess, nil
}

// segmentTurns groups records into conversational turns.
func segmentTurns(records []parser.Record, sess *Session) []Turn {
	var turns []Turn
	var currentTurn *Turn
	turnNum := 0

	// Track durations from system records
	pendingDuration := time.Duration(0)

	for _, rec := range records {
		// Extract session metadata from first records we see
		if sess.ID == "" && rec.SessionID != "" {
			sess.ID = rec.SessionID
		}
		if sess.Slug == "" && rec.Slug != "" {
			sess.Slug = rec.Slug
		}
		if sess.Version == "" && rec.Version != "" {
			sess.Version = rec.Version
		}

		switch rec.Type {
		case parser.RecordTypeUser:
			// Skip meta messages (expanded skill prompts injected after commands)
			if rec.IsMeta {
				continue
			}

			userMsg, err := rec.ParseUserMessage()
			if err != nil {
				continue
			}

			if userMsg.IsBashOutput() {
				// Shell escape output (!cmd) belongs to the current turn
				if currentTurn != nil {
					stdout, stderr := userMsg.ParseBashOutput()
					output := stdout
					if stderr != "" {
						if output != "" {
							output += "\n"
						}
						output += stderr
					}
					if output != "" {
						currentTurn.Blocks = append(currentTurn.Blocks, Block{
							Type: BlockText,
							Text: output,
						})
					}
				}
			} else if userMsg.IsBashInput() {
				// Shell escape command (!cmd) starts a new turn
				cmd := userMsg.ParseBashInput()

				if currentTurn != nil && pendingDuration > 0 {
					currentTurn.Duration = pendingDuration
					pendingDuration = 0
				}
				if currentTurn != nil {
					turns = append(turns, *currentTurn)
				}

				turnNum++
				currentTurn = &Turn{
					Number:    turnNum,
					UserText:  "!" + cmd,
					Timestamp: rec.Timestamp,
					CWD:       rec.CWD,
					GitBranch: rec.GitBranch,
					Slug:      rec.Slug,
				}

				if sess.CWD == "" {
					sess.CWD = rec.CWD
				}
				if sess.GitBranch == "" {
					sess.GitBranch = rec.GitBranch
				}
			} else if userMsg.IsToolResults() {
				// Tool results belong to the current turn
				if currentTurn != nil {
					results, err := userMsg.ParseToolResults()
					if err == nil {
						for _, tr := range results {
							if tr.Type != "tool_result" {
								continue
							}
							block := Block{
								Type:   BlockToolResult,
								ToolID: tr.ToolUseID,
							}
							// Parse content: can be string or array
							block.Text = extractToolResultContent(tr.Content)
							if tr.IsError != nil && *tr.IsError {
								block.IsError = true
							}
							currentTurn.Blocks = append(currentTurn.Blocks, block)
						}
					}
				}
			} else {
				// Check for slash command messages
				text := userMsg.UserText()
				if cmdName, ok := userMsg.CommandName(); ok {
					text = cmdName
				}
				if text == "" {
					continue
				}

				// Save pending duration to previous turn
				if currentTurn != nil && pendingDuration > 0 {
					currentTurn.Duration = pendingDuration
					pendingDuration = 0
				}

				// Finalize previous turn
				if currentTurn != nil {
					turns = append(turns, *currentTurn)
				}

				turnNum++
				currentTurn = &Turn{
					Number:    turnNum,
					UserText:  text,
					Timestamp: rec.Timestamp,
					CWD:       rec.CWD,
					GitBranch: rec.GitBranch,
					Slug:      rec.Slug,
				}

				if sess.CWD == "" {
					sess.CWD = rec.CWD
				}
				if sess.GitBranch == "" {
					sess.GitBranch = rec.GitBranch
				}
			}

		case parser.RecordTypeAssistant:
			if currentTurn == nil {
				continue
			}

			aMsg, err := rec.ParseAssistantMessage()
			if err != nil {
				continue
			}

			if currentTurn.Model == "" && aMsg.Model != "" {
				currentTurn.Model = aMsg.Model
				if sess.Model == "" {
					sess.Model = aMsg.Model
				}
			}

			for _, cb := range aMsg.Content {
				switch cb.Type {
				case "text":
					text := strings.TrimSpace(cb.Text)
					if text == "" {
						continue
					}
					currentTurn.Blocks = append(currentTurn.Blocks, Block{
						Type: BlockText,
						Text: text,
					})
				case "thinking":
					if cb.Thinking == "" {
						continue
					}
					currentTurn.Blocks = append(currentTurn.Blocks, Block{
						Type: BlockThinking,
						Text: cb.Thinking,
					})
				case "tool_use":
					block := Block{
						Type:     BlockToolUse,
						ToolName: cb.Name,
						ToolID:   cb.ID,
					}
					if cb.Input != nil {
						var input map[string]interface{}
						if err := json.Unmarshal(cb.Input, &input); err == nil {
							block.ToolInput = input
						}
						block.RawInput = string(cb.Input)
					}
					currentTurn.Blocks = append(currentTurn.Blocks, block)
				}
			}

		case parser.RecordTypeSystem:
			if rec.Subtype == "turn_duration" && rec.DurationMs > 0 {
				pendingDuration = time.Duration(rec.DurationMs) * time.Millisecond
			}
		}
	}

	// Finalize last turn
	if currentTurn != nil {
		if pendingDuration > 0 {
			currentTurn.Duration = pendingDuration
		}
		turns = append(turns, *currentTurn)
	}

	return turns
}

// extractToolResultContent parses tool result content which can be a string
// or an array of objects with text fields.
func extractToolResultContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	// Try array of content blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}

	return string(raw)
}
