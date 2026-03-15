package providers

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCodexProvider_Name(t *testing.T) {
	p := NewCodexProvider(nil, 0, 0)
	if p.Name() != "codex" {
		t.Errorf("Name() = %q, want %q", p.Name(), "codex")
	}
}

func TestCodexProvider_BuildArgs(t *testing.T) {
	p := &CodexProvider{}

	t.Run("new session uses exec form", func(t *testing.T) {
		args := p.buildArgs(InvokeRequest{UserMessage: "hello"}, "")
		if args[0] != "exec" {
			t.Errorf("args[0] = %q, want %q", args[0], "exec")
		}
		if sliceContains(args, "resume") {
			t.Errorf("new session args should not contain 'resume': %v", args)
		}
		// Message should be the last arg.
		if args[len(args)-1] != "hello" {
			t.Errorf("last arg = %q, want %q", args[len(args)-1], "hello")
		}
	})

	t.Run("resume session uses exec resume form", func(t *testing.T) {
		args := p.buildArgs(InvokeRequest{UserMessage: "hello", SessionID: "thread-123"}, "")
		if len(args) < 4 {
			t.Fatalf("too few args: %v", args)
		}
		if args[0] != "exec" || args[1] != "resume" || args[2] != "thread-123" || args[3] != "hello" {
			t.Errorf("args = %v, want [exec resume thread-123 hello ...]", args)
		}
	})

	t.Run("stateless adds --ephemeral", func(t *testing.T) {
		args := p.buildArgs(InvokeRequest{UserMessage: "hello", Stateless: true}, "")
		if !sliceContains(args, "--ephemeral") {
			t.Errorf("stateless args %v should contain --ephemeral", args)
		}
	})

	t.Run("non-stateless omits --ephemeral", func(t *testing.T) {
		args := p.buildArgs(InvokeRequest{UserMessage: "hello"}, "")
		if sliceContains(args, "--ephemeral") {
			t.Errorf("non-stateless args %v should not contain --ephemeral", args)
		}
	})

	t.Run("model flag", func(t *testing.T) {
		args := p.buildArgs(InvokeRequest{UserMessage: "hello", Model: "gpt-4o"}, "")
		idx := sliceIndexOf(args, "--model")
		if idx == -1 || idx+1 >= len(args) || args[idx+1] != "gpt-4o" {
			t.Errorf("args %v should contain --model gpt-4o", args)
		}
	})

	t.Run("no sysPromptFile omits -c flag", func(t *testing.T) {
		args := p.buildArgs(InvokeRequest{UserMessage: "user question", SystemPrompt: "be helpful"}, "")
		if sliceContains(args, "-c") {
			t.Errorf("args %v should not contain -c when no sysPromptFile", args)
		}
		// Message should be unchanged (no <system> wrapping).
		msg := args[len(args)-1]
		if strings.Contains(msg, "<system>") {
			t.Errorf("message should not contain <system>: %q", msg)
		}
		if msg != "user question" {
			t.Errorf("message = %q, want %q", msg, "user question")
		}
	})

	t.Run("sysPromptFile adds -c flag", func(t *testing.T) {
		args := p.buildArgs(InvokeRequest{UserMessage: "hello"}, "/tmp/x.md")
		idx := sliceIndexOf(args, "-c")
		if idx == -1 || idx+1 >= len(args) {
			t.Fatalf("args %v should contain -c <value>", args)
		}
		if args[idx+1] != "model_instructions_file=/tmp/x.md" {
			t.Errorf("args[%d] = %q, want %q", idx+1, args[idx+1], "model_instructions_file=/tmp/x.md")
		}
	})

	t.Run("no system prompt leaves message unchanged", func(t *testing.T) {
		args := p.buildArgs(InvokeRequest{UserMessage: "hello"}, "")
		msg := args[len(args)-1]
		if strings.Contains(msg, "<system>") {
			t.Errorf("message without system prompt should not contain <system>: %q", msg)
		}
		if msg != "hello" {
			t.Errorf("message = %q, want %q", msg, "hello")
		}
	})
}

func TestCodexProvider_Invoke_Success(t *testing.T) {
	fakeInPath(t, "codex", "codex_success")
	p := NewCodexProvider(nil, 0, 0)

	result, err := p.Invoke(context.Background(), InvokeRequest{
		UserMessage: "hello",
		WorkDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	// The last agent_message item is the final reply.
	if result.Text != "hello from codex" {
		t.Errorf("Text = %q, want %q", result.Text, "hello from codex")
	}
	if result.SessionID != "thread-xyz" {
		t.Errorf("SessionID = %q, want %q", result.SessionID, "thread-xyz")
	}
	if result.TokensIn != 20 {
		t.Errorf("TokensIn = %d, want 20", result.TokensIn)
	}
	if result.TokensOut != 8 {
		t.Errorf("TokensOut = %d, want 8", result.TokensOut)
	}
}

func TestCodexProvider_Invoke_StderrOnFailure(t *testing.T) {
	fakeInPath(t, "codex", "codex_stderr_failure")
	p := NewCodexProvider(nil, 0, 0)

	_, err := p.Invoke(context.Background(), InvokeRequest{
		UserMessage: "hello",
		WorkDir:     t.TempDir(),
	})
	if err == nil {
		t.Fatal("Invoke() expected error, got nil")
	}
	var provErr *ErrProviderMessage
	if !errors.As(err, &provErr) {
		t.Fatalf("err type = %T, want *ErrProviderMessage", err)
	}
	if !strings.Contains(provErr.Message, "authentication required") {
		t.Errorf("Message = %q, want to contain %q", provErr.Message, "authentication required")
	}
}

// TestCodexProvider_NoTimeout_Disabled verifies that idleTimeout=0 disables the timer and
// normal successful execution still works.
func TestCodexProvider_NoTimeout_Disabled(t *testing.T) {
	fakeInPath(t, "codex", "codex_success")
	p := NewCodexProvider(nil, 0, 0)

	result, err := p.Invoke(context.Background(), InvokeRequest{
		UserMessage: "hello",
		WorkDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v (timeout disabled should not affect success)", err)
	}
	if result.Text != "hello from codex" {
		t.Errorf("Text = %q, want %q", result.Text, "hello from codex")
	}
}

// TestCodexProvider_IdleTimeout_TriggersRetry verifies that when the subprocess emits no output
// within the idle deadline, Invoke returns an error wrapping errIdleTimeout.
func TestCodexProvider_IdleTimeout_TriggersRetry(t *testing.T) {
	fakeInPath(t, "codex", "codex_idle_hang")
	p := NewCodexProvider(nil, 200*time.Millisecond, 1)

	_, err := p.Invoke(context.Background(), InvokeRequest{
		UserMessage: "hello",
		WorkDir:     t.TempDir(),
	})
	if err == nil {
		t.Fatal("Invoke() expected error on idle timeout, got nil")
	}
	if !errors.Is(err, errIdleTimeout) {
		t.Errorf("err = %v, want to wrap errIdleTimeout", err)
	}
}

// TestCodexProvider_SystemPromptFile verifies that runOnce writes the system prompt to a temp
// file, passes it via -c model_instructions_file, and removes the file after returning.
func TestCodexProvider_SystemPromptFile(t *testing.T) {
	fakeInPath(t, "codex", "codex_check_instructions_file")
	p := &CodexProvider{}

	result, _, err := p.runOnce(context.Background(), InvokeRequest{
		UserMessage:  "hello",
		SystemPrompt: "be helpful",
		WorkDir:      t.TempDir(),
	})
	if err != nil {
		t.Fatalf("runOnce() error = %v", err)
	}
	// The fake scenario echoes the file path as the agent_message text.
	tmpPath := result.Text
	if tmpPath == "" {
		t.Fatal("expected result.Text to contain the temp file path")
	}
	if _, statErr := os.Stat(tmpPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("temp file %q should have been removed after runOnce returned", tmpPath)
	}
}

// TestCodexProvider_IdleTimeout_RetrySucceeds verifies that after an idle timeout the provider
// retries with the captured thread ID and returns the successful result.
func TestCodexProvider_IdleTimeout_RetrySucceeds(t *testing.T) {
	fakeInPath(t, "codex", "codex_idle_then_succeed")
	p := NewCodexProvider(nil, 500*time.Millisecond, 3)

	result, err := p.Invoke(context.Background(), InvokeRequest{
		UserMessage: "hello",
		WorkDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if result.Text != "recovered after idle" {
		t.Errorf("Text = %q, want %q", result.Text, "recovered after idle")
	}
	if result.SessionID != "thread-recovered" {
		t.Errorf("SessionID = %q, want %q", result.SessionID, "thread-recovered")
	}
}
