package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// InvokeLogger writes per-key invoke trace files to workspace/logs/<key>.log.
// Files are opened lazily on first use and kept open. The logger is nil-safe.
type InvokeLogger struct {
	mu      sync.Mutex
	workDir string
	handles map[string]*os.File
}

// NewInvokeLogger creates a logger that writes to workspace/logs/<key>.log.
func NewInvokeLogger(workDir string) (*InvokeLogger, error) {
	logsDir := filepath.Join(workDir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return nil, fmt.Errorf("invoke log: create logs dir: %w", err)
	}
	return &InvokeLogger{
		workDir: workDir,
		handles: make(map[string]*os.File),
	}, nil
}

// fileFor returns (or lazily opens) the file handle for the given key.
// Caller must hold l.mu.
func (l *InvokeLogger) fileFor(key string) *os.File {
	if f, ok := l.handles[key]; ok {
		return f
	}
	path := filepath.Join(l.workDir, "logs", sanitizeLogKey(key)+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil
	}
	l.handles[key] = f
	return f
}

// sanitizeLogKey converts a bridge session key like "telegram:300185392" to "telegram_300185392".
func sanitizeLogKey(key string) string {
	return strings.NewReplacer(":", "_", "/", "_", " ", "_").Replace(key)
}

// LogInvoke writes the invocation header line.
func (l *InvokeLogger) LogInvoke(logKey, provider, agentSessionID, model, userMessage string) {
	if l == nil || logKey == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	f := l.fileFor(logKey)
	if f == nil {
		return
	}
	sess := agentSessionID
	if sess == "" {
		sess = "(new)"
	}
	mod := model
	if mod == "" {
		mod = "(default)"
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	fmt.Fprintf(f, "\n=== BEGIN %s  provider=%s  session_id=%s  model=%s ===\n", ts, provider, sess, mod)
	fmt.Fprintf(f, "{\"type\":\"user\",\"text\":%s}\n", jsonString(userMessage))
}

// LogLine writes a raw JSON line from the subprocess stdout.
func (l *InvokeLogger) LogLine(logKey, line string) {
	if l == nil || logKey == "" || line == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	f := l.fileFor(logKey)
	if f == nil {
		return
	}
	fmt.Fprintln(f, line)
}

// LogStderr writes subprocess stderr output.
func (l *InvokeLogger) LogStderr(logKey, s string) {
	if l == nil || logKey == "" || s == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	f := l.fileFor(logKey)
	if f == nil {
		return
	}
	fmt.Fprintf(f, "[stderr] %s\n", s)
}

// LogDone writes the closing summary line.
func (l *InvokeLogger) LogDone(logKey, agentSessionID string, tokensIn, tokensOut, textLen int, duration time.Duration) {
	if l == nil || logKey == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	f := l.fileFor(logKey)
	if f == nil {
		return
	}
	fmt.Fprintf(f, "=== END  session_id=%s  in=%d  out=%d  len=%d  duration=%.2fs ===\n",
		agentSessionID, tokensIn, tokensOut, textLen, duration.Seconds())
}

// jsonString returns a JSON-encoded string value, falling back to a quoted literal on error.
func jsonString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return fmt.Sprintf("%q", s)
	}
	return string(b)
}

// Close closes all open file handles.
func (l *InvokeLogger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	var lastErr error
	for key, f := range l.handles {
		if err := f.Close(); err != nil {
			lastErr = err
		}
		delete(l.handles, key)
	}
	return lastErr
}
