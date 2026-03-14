package heartbeat

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExecuteHeartbeat_HandlerCalled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hs := NewHeartbeatService(tmpDir, 30, true)
	hs.stopChan = make(chan struct{})

	called := false
	hs.SetHandler(func(prompt, channel, chatID string) (string, error) {
		called = true
		if prompt == "" {
			t.Error("Expected non-empty prompt")
		}
		return "", nil
	})

	os.WriteFile(filepath.Join(tmpDir, "HEARTBEAT.md"), []byte("Test task"), 0o644)
	hs.executeHeartbeat()

	if !called {
		t.Error("Expected handler to be called")
	}
}

func TestExecuteHeartbeat_HandlerOutcomes(t *testing.T) {
	tests := []struct {
		name string
		text string
		err  error
	}{
		{name: "error result", text: "", err: fmt.Errorf("connection error")},
		{name: "silent result", text: "", err: nil},
		{name: "user message", text: "Hello from heartbeat", err: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			hs := NewHeartbeatService(tmpDir, 30, true)
			hs.stopChan = make(chan struct{})
			hs.SetHandler(func(prompt, channel, chatID string) (string, error) {
				return tt.text, tt.err
			})
			os.WriteFile(filepath.Join(tmpDir, "HEARTBEAT.md"), []byte("Test task"), 0o644)
			// Should not panic regardless of outcome.
			hs.executeHeartbeat()
		})
	}
}

func TestHeartbeatService_StartStop(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hs := NewHeartbeatService(tmpDir, 1, true)

	err = hs.Start()
	if err != nil {
		t.Fatalf("Failed to start heartbeat service: %v", err)
	}

	hs.Stop()

	time.Sleep(100 * time.Millisecond)
}

func TestHeartbeatService_Disabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hs := NewHeartbeatService(tmpDir, 1, false)

	if hs.enabled != false {
		t.Error("Expected service to be disabled")
	}

	err = hs.Start()
	_ = err
}


func TestHeartbeatBuildPromptMissingFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hs := NewHeartbeatService(tmpDir, 30, true)
	prompt := hs.buildPrompt()

	if prompt != "" {
		t.Errorf("Expected empty prompt when HEARTBEAT.md is missing, got: %s", prompt)
	}
}
