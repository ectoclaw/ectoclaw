package providers

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCodexProvider_Name(t *testing.T) {
	p := NewCodexProvider(nil)
	if p.Name() != "codex" {
		t.Errorf("Name() = %q, want %q", p.Name(), "codex")
	}
}

func TestCodexProvider_BuildArgs(t *testing.T) {
	p := &CodexProvider{}

	t.Run("new session uses exec form", func(t *testing.T) {
		args := p.buildArgs(InvokeRequest{UserMessage: "hello"})
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
		args := p.buildArgs(InvokeRequest{UserMessage: "hello", SessionID: "thread-123"})
		if len(args) < 4 {
			t.Fatalf("too few args: %v", args)
		}
		if args[0] != "exec" || args[1] != "resume" || args[2] != "thread-123" || args[3] != "hello" {
			t.Errorf("args = %v, want [exec resume thread-123 hello ...]", args)
		}
	})

	t.Run("stateless adds --ephemeral", func(t *testing.T) {
		args := p.buildArgs(InvokeRequest{UserMessage: "hello", Stateless: true})
		if !sliceContains(args, "--ephemeral") {
			t.Errorf("stateless args %v should contain --ephemeral", args)
		}
	})

	t.Run("non-stateless omits --ephemeral", func(t *testing.T) {
		args := p.buildArgs(InvokeRequest{UserMessage: "hello"})
		if sliceContains(args, "--ephemeral") {
			t.Errorf("non-stateless args %v should not contain --ephemeral", args)
		}
	})

	t.Run("model flag", func(t *testing.T) {
		args := p.buildArgs(InvokeRequest{UserMessage: "hello", Model: "gpt-4o"})
		idx := sliceIndexOf(args, "--model")
		if idx == -1 || idx+1 >= len(args) || args[idx+1] != "gpt-4o" {
			t.Errorf("args %v should contain --model gpt-4o", args)
		}
	})

	t.Run("system prompt embedded in message", func(t *testing.T) {
		args := p.buildArgs(InvokeRequest{UserMessage: "user question", SystemPrompt: "be helpful"})
		// For a new session the message is always the last argument.
		msg := args[len(args)-1]
		if !strings.Contains(msg, "<system>") {
			t.Errorf("message should contain <system> tag: %q", msg)
		}
		if !strings.Contains(msg, "be helpful") {
			t.Errorf("message should contain system prompt content: %q", msg)
		}
		if !strings.Contains(msg, "user question") {
			t.Errorf("message should contain original user message: %q", msg)
		}
	})

	t.Run("no system prompt leaves message unchanged", func(t *testing.T) {
		args := p.buildArgs(InvokeRequest{UserMessage: "hello"})
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
	p := NewCodexProvider(nil)

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
	p := NewCodexProvider(nil)

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
