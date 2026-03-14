package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAtomicSave(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm := NewManager(tmpDir)

	err = sm.SetLastChannel("telegram:123456")
	if err != nil {
		t.Fatalf("SetLastChannel failed: %v", err)
	}

	if sm.GetLastChannel() != "telegram:123456" {
		t.Errorf("Expected 'telegram:123456', got '%s'", sm.GetLastChannel())
	}

	if sm.GetTimestamp().IsZero() {
		t.Error("Expected timestamp to be updated")
	}

	stateFile := filepath.Join(tmpDir, "state", "state.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("Expected state file to exist")
	}

	sm2 := NewManager(tmpDir)
	if sm2.GetLastChannel() != "telegram:123456" {
		t.Errorf("Expected persistent 'telegram:123456', got '%s'", sm2.GetLastChannel())
	}
}

func TestAtomicity_NoCorruptionOnInterrupt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm := NewManager(tmpDir)

	err = sm.SetLastChannel("telegram:111")
	if err != nil {
		t.Fatalf("SetLastChannel failed: %v", err)
	}

	// Simulate a crash by creating a corrupted temp file
	tempFile := filepath.Join(tmpDir, "state", "state.json.tmp")
	err = os.WriteFile(tempFile, []byte("corrupted data"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if sm.GetLastChannel() != "telegram:111" {
		t.Errorf("Expected 'telegram:111' after corrupted temp file, got '%s'", sm.GetLastChannel())
	}

	os.Remove(tempFile)

	err = sm.SetLastChannel("telegram:222")
	if err != nil {
		t.Fatalf("SetLastChannel failed: %v", err)
	}

	if sm.GetLastChannel() != "telegram:222" {
		t.Errorf("Expected 'telegram:222', got '%s'", sm.GetLastChannel())
	}
}

func TestConcurrentAccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm := NewManager(tmpDir)

	done := make(chan bool, 10)
	for i := range 10 {
		go func(idx int) {
			sm.SetLastChannel(fmt.Sprintf("telegram:%d", idx))
			done <- true
		}(i)
	}

	for range 10 {
		<-done
	}

	if sm.GetLastChannel() == "" {
		t.Error("Expected non-empty channel after concurrent writes")
	}

	stateFile := filepath.Join(tmpDir, "state", "state.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		t.Errorf("State file contains invalid JSON: %v", err)
	}
}

func TestNewManager_ExistingState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm1 := NewManager(tmpDir)
	sm1.SetLastChannel("slack:C1234567")

	sm2 := NewManager(tmpDir)
	if sm2.GetLastChannel() != "slack:C1234567" {
		t.Errorf("Expected 'slack:C1234567', got '%s'", sm2.GetLastChannel())
	}
}

func TestNewManager_EmptyWorkspace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm := NewManager(tmpDir)

	if sm.GetLastChannel() != "" {
		t.Errorf("Expected empty channel, got '%s'", sm.GetLastChannel())
	}

	if !sm.GetTimestamp().IsZero() {
		t.Error("Expected zero timestamp for new state")
	}
}

func TestNewManager_MkdirFailureCrashes(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		tmpDir := os.Getenv("CRASH_DIR")

		statePath := filepath.Join(tmpDir, "state")
		if err := os.WriteFile(statePath, []byte("I'm a file, not a folder"), 0o644); err != nil {
			fmt.Printf("setup failed: %v", err)
			os.Exit(0)
		}

		NewManager(tmpDir)
		os.Exit(0)
	}

	tmpDir, err := os.MkdirTemp("", "state-crash-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command(os.Args[0], "-test.run=TestNewManager_MkdirFailureCrashes")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1", "CRASH_DIR="+tmpDir)

	err = cmd.Run()

	var e *exec.ExitError
	if errors.As(err, &e) && !e.Success() {
		return
	}

	t.Fatalf("The process ended without error, a crash was expected via os.Exit(1). Err: %v", err)
}
