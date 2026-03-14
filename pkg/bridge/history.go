package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type historyEntry struct {
	Ts      string `json:"ts"`
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AppendHistory appends a user/assistant exchange to the daily history file.
// File: <workDir>/history/YYYY-MM-DD/messages.jsonl
// Permissions: 0o600 (sensitive content)
func AppendHistory(workDir, userMsg, assistantMsg string) error {
	dir := filepath.Join(workDir, "history", time.Now().UTC().Format("2006-01-02"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("history: create directory: %w", err)
	}

	path := filepath.Join(dir, "messages.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("history: open file: %w", err)
	}
	defer f.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, entry := range []historyEntry{
		{Ts: now, Role: "user", Content: userMsg},
		{Ts: now, Role: "assistant", Content: assistantMsg},
	} {
		line, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("history: marshal entry: %w", err)
		}
		line = append(line, '\n')
		if _, err := f.Write(line); err != nil {
			return fmt.Errorf("history: write entry: %w", err)
		}
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("history: sync: %w", err)
	}
	return nil
}
