package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ectoclaw/ectoclaw/pkg/fileutil"
)

// State represents the persistent state for a workspace.
type State struct {
	// LastChannel is the last channel used, in "platform:chatID" format (e.g. "telegram:123456")
	LastChannel string `json:"last_channel,omitempty"`

	// Timestamp is the last time this state was updated
	Timestamp time.Time `json:"timestamp"`
}

// Manager manages persistent state with atomic saves.
type Manager struct {
	workspace string
	state     *State
	mu        sync.RWMutex
	stateFile string
}

// NewManager creates a new state manager for the given workspace.
func NewManager(workspace string) *Manager {
	stateDir := filepath.Join(workspace, "state")
	stateFile := filepath.Join(stateDir, "state.json")
	oldStateFile := filepath.Join(workspace, "state.json")

	// Create state directory if it doesn't exist
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		log.Fatalf("[FATAL] state: failed to create state directory: %v", err)
	}

	sm := &Manager{
		workspace: workspace,
		stateFile: stateFile,
		state:     &State{},
	}

	// Try to load from new location first
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		// New file doesn't exist, try migrating from old location
		if data, err := os.ReadFile(oldStateFile); err == nil {
			if err := json.Unmarshal(data, sm.state); err == nil {
				// Migrate to new location
				if err := sm.saveAtomic(); err != nil {
					log.Printf("[WARN] state: failed to save state: %v", err)
				}
				log.Printf("[INFO] state: migrated state from %s to %s", oldStateFile, stateFile)
			}
		}
	} else {
		// Load from new location
		if err := sm.load(); err != nil {
			log.Printf("[WARN] state: failed to load state: %v", err)
		}
	}

	return sm
}

// SetLastChannel atomically updates the last channel and saves the state.
// channel must be in "platform:chatID" format (e.g. "telegram:123456").
func (sm *Manager) SetLastChannel(channel string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state.LastChannel = channel
	sm.state.Timestamp = time.Now()

	if err := sm.saveAtomic(); err != nil {
		return fmt.Errorf("failed to save state atomically: %w", err)
	}

	return nil
}

// GetLastChannel returns the last channel in "platform:chatID" format.
func (sm *Manager) GetLastChannel() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.LastChannel
}

// GetTimestamp returns the timestamp of the last state update.
func (sm *Manager) GetTimestamp() time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.Timestamp
}

// saveAtomic performs an atomic save using temp file + rename.
// Must be called with the lock held.
func (sm *Manager) saveAtomic() error {
	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	return fileutil.WriteFileAtomic(sm.stateFile, data, 0o600)
}

// load loads the state from disk.
func (sm *Manager) load() error {
	data, err := os.ReadFile(sm.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, sm.state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return nil
}
