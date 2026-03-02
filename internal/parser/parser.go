package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// ParseFile reads a JSONL session file and returns all records.
// Progress records and file-history-snapshot records are filtered out.
func ParseFile(path string) ([]Record, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse reads JSONL records from a reader.
func Parse(r io.Reader) ([]Record, error) {
	var records []Record
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 4*1024*1024), 16*1024*1024) // up to 16MB per line

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var rec Record
		if err := json.Unmarshal(line, &rec); err != nil {
			continue // skip malformed lines
		}

		// Filter out noise records
		switch rec.Type {
		case RecordTypeProgress, RecordTypeSnapshot:
			continue
		}

		// Skip sidechain records
		if rec.IsSidechain {
			continue
		}

		records = append(records, rec)
	}

	if err := scanner.Err(); err != nil {
		return records, err
	}

	return records, nil
}

// QuickScan reads just enough of a session file to extract metadata
// without parsing the entire file. Returns slug, model, first timestamp,
// last timestamp, and approximate turn count.
func QuickScan(path string) (slug, model string, firstTime, lastTime string, turnCount int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", "", "", 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)

	type quickRecord struct {
		Type      string `json:"type"`
		Slug      string `json:"slug"`
		Timestamp string `json:"timestamp"`
		Subtype   string `json:"subtype"`
		IsMeta    bool   `json:"isMeta"`
		Message   *struct {
			Role    string          `json:"role"`
			Model   string          `json:"model"`
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var rec quickRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}

		if rec.Timestamp != "" {
			if firstTime == "" {
				firstTime = rec.Timestamp
			}
			lastTime = rec.Timestamp
		}

		if rec.Slug != "" && slug == "" {
			slug = rec.Slug
		}

		if rec.Type == "user" && rec.Message != nil && rec.Message.Role == "user" {
			// Skip meta messages (expanded skill prompts)
			if rec.IsMeta {
				continue
			}
			if len(rec.Message.Content) > 0 {
				switch rec.Message.Content[0] {
				case '"':
					// Plain string content — skip bash output
					if bytes.Contains(rec.Message.Content, []byte("bash-stdout")) ||
						bytes.Contains(rec.Message.Content, []byte("bash-stderr")) {
						continue
					}
					turnCount++
				case '[':
					// Array content — check if it's tool results vs text+image
					var items []struct {
						Type string `json:"type"`
					}
					if err := json.Unmarshal(rec.Message.Content, &items); err == nil && len(items) > 0 {
						if items[0].Type != "tool_result" {
							turnCount++
						}
					}
				}
			}
		}

		if rec.Type == "assistant" && rec.Message != nil && rec.Message.Model != "" && model == "" {
			model = rec.Message.Model
		}
	}

	return slug, model, firstTime, lastTime, turnCount, scanner.Err()
}
