package bridge

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/ectoclaw/ectoclaw/pkg/fileutil"
)

// Sessions is a thread-safe SessionKey → claude session_id map with atomic file persistence.
type Sessions struct {
	mu   sync.RWMutex
	data map[string]string
	path string
}

// NewSessions creates a new Sessions store backed by the given file path.
func NewSessions(path string) *Sessions {
	return &Sessions{
		data: make(map[string]string),
		path: path,
	}
}

// Get returns the claude session_id for the given key, if present.
func (s *Sessions) Get(key string) (sessionID string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sessionID, ok = s.data[key]
	return
}

// Set stores or updates the claude session_id for the given key.
func (s *Sessions) Set(key, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = sessionID
}

// Delete removes the session entry for the given key.
func (s *Sessions) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// Clear removes all session entries from memory and persists the empty state to disk.
func (s *Sessions) Clear() error {
	s.mu.Lock()
	s.data = make(map[string]string)
	s.mu.Unlock()
	return s.Save()
}

// Save atomically persists the current session map to disk.
func (s *Sessions) Save() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.data, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(s.path, data, 0o600)
}

// Load reads the session map from disk. Returns nil if the file does not exist.
func (s *Sessions) Load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.Unmarshal(data, &s.data)
}
