package providers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSanitizeLogKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"telegram:300185392", "telegram_300185392"},
		{"channel/123", "channel_123"},
		{"has space", "has_space"},
		{"no:change/needed here", "no_change_needed_here"},
		{"plain", "plain"},
	}
	for _, tc := range tests {
		got := sanitizeLogKey(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeLogKey(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestInvokeLogger_NilSafe(t *testing.T) {
	var l *InvokeLogger
	// None of these should panic.
	l.LogInvoke("key", "claude", "sess", "model", "msg")
	l.LogLine("key", "line")
	l.LogStderr("key", "stderr")
	l.LogDone("key", "sess", 10, 5, 100, time.Second)
	if err := l.Close(); err != nil {
		t.Errorf("Close() on nil logger returned %v, want nil", err)
	}
}

func TestInvokeLogger_EmptyKeyIsNoOp(t *testing.T) {
	dir := t.TempDir()
	l, err := NewInvokeLogger(dir)
	if err != nil {
		t.Fatalf("NewInvokeLogger() error = %v", err)
	}
	defer l.Close()

	// All calls with an empty key should be no-ops — no file should be created.
	l.LogInvoke("", "claude", "sess", "model", "msg")
	l.LogLine("", "line")
	l.LogStderr("", "stderr")
	l.LogDone("", "sess", 10, 5, 100, time.Second)

	entries, _ := os.ReadDir(filepath.Join(dir, "logs"))
	if len(entries) != 0 {
		t.Errorf("expected no log files for empty key, got %d file(s)", len(entries))
	}
}

func TestInvokeLogger_WritesOutput(t *testing.T) {
	dir := t.TempDir()
	l, err := NewInvokeLogger(dir)
	if err != nil {
		t.Fatalf("NewInvokeLogger() error = %v", err)
	}
	defer l.Close()

	l.LogInvoke("test-key", "claude", "sess-123", "claude-opus", "hello there")
	l.LogLine("test-key", `{"type":"result"}`)
	l.LogStderr("test-key", "some warning")
	l.LogDone("test-key", "sess-123", 10, 5, 11, 500*time.Millisecond)

	data, err := os.ReadFile(filepath.Join(dir, "logs", "test-key.log"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(data)

	for _, want := range []string{
		"claude", "sess-123", "hello there",
		`{"type":"result"}`,
		"[stderr]", "some warning",
		"in=10", "out=5",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("log file missing %q\nfull content:\n%s", want, content)
		}
	}
}

func TestInvokeLogger_SanitizesKeyInFilename(t *testing.T) {
	dir := t.TempDir()
	l, err := NewInvokeLogger(dir)
	if err != nil {
		t.Fatalf("NewInvokeLogger() error = %v", err)
	}
	defer l.Close()

	l.LogLine("telegram:12345", `{"type":"result"}`)

	logPath := filepath.Join(dir, "logs", "telegram_12345.log")
	if _, err := os.Stat(logPath); err != nil {
		t.Errorf("expected log file at %q: %v", logPath, err)
	}
}

func TestNewInvokeLogger_CreatesLogsDir(t *testing.T) {
	workDir := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	l, err := NewInvokeLogger(workDir)
	if err != nil {
		t.Fatalf("NewInvokeLogger() error = %v", err)
	}
	defer l.Close()

	if _, err := os.Stat(filepath.Join(workDir, "logs")); err != nil {
		t.Errorf("logs dir not created: %v", err)
	}
}
