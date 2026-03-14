package providers

import (
	"context"
	"errors"
	"testing"
)

func TestClaudeProvider_Name(t *testing.T) {
	p := NewClaudeProvider(nil)
	if p.Name() != "claude" {
		t.Errorf("Name() = %q, want %q", p.Name(), "claude")
	}
}

func TestClaudeProvider_BuildArgs(t *testing.T) {
	p := &ClaudeProvider{}
	tests := []struct {
		name     string
		req      InvokeRequest
		wantArgs []string
	}{
		{
			name: "minimal",
			req:  InvokeRequest{UserMessage: "hello"},
			wantArgs: []string{
				"-p", "hello",
				"--output-format", "stream-json",
				"--verbose",
				"--dangerously-skip-permissions",
			},
		},
		{
			name: "with session",
			req:  InvokeRequest{UserMessage: "hello", SessionID: "sess-123"},
			wantArgs: []string{
				"--resume", "sess-123",
				"-p", "hello",
				"--output-format", "stream-json",
				"--verbose",
				"--dangerously-skip-permissions",
			},
		},
		{
			name: "with system prompt",
			req:  InvokeRequest{UserMessage: "hello", SystemPrompt: "be helpful"},
			wantArgs: []string{
				"-p", "hello",
				"--output-format", "stream-json",
				"--verbose",
				"--dangerously-skip-permissions",
				"--system-prompt", "be helpful",
			},
		},
		{
			name: "with model",
			req:  InvokeRequest{UserMessage: "hello", Model: "claude-opus-4-5"},
			wantArgs: []string{
				"-p", "hello",
				"--output-format", "stream-json",
				"--verbose",
				"--dangerously-skip-permissions",
				"--model", "claude-opus-4-5",
			},
		},
		{
			name: "all fields set",
			req: InvokeRequest{
				UserMessage:  "hello",
				SessionID:    "sess-123",
				SystemPrompt: "be helpful",
				Model:        "claude-opus-4-5",
			},
			wantArgs: []string{
				"--resume", "sess-123",
				"-p", "hello",
				"--output-format", "stream-json",
				"--verbose",
				"--dangerously-skip-permissions",
				"--system-prompt", "be helpful",
				"--model", "claude-opus-4-5",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := p.buildArgs(tc.req)
			if len(got) != len(tc.wantArgs) {
				t.Fatalf("buildArgs() = %v (len %d), want %v (len %d)", got, len(got), tc.wantArgs, len(tc.wantArgs))
			}
			for i := range got {
				if got[i] != tc.wantArgs[i] {
					t.Errorf("args[%d] = %q, want %q", i, got[i], tc.wantArgs[i])
				}
			}
		})
	}
}

func TestClaudeProvider_Invoke_Success(t *testing.T) {
	fakeInPath(t, "claude", "claude_success")
	p := NewClaudeProvider(nil)

	result, err := p.Invoke(context.Background(), InvokeRequest{
		UserMessage: "hello",
		WorkDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if result.Text != "hello world" {
		t.Errorf("Text = %q, want %q", result.Text, "hello world")
	}
	if result.SessionID != "sess-abc" {
		t.Errorf("SessionID = %q, want %q", result.SessionID, "sess-abc")
	}
	if result.TokensIn != 10 {
		t.Errorf("TokensIn = %d, want 10", result.TokensIn)
	}
	if result.TokensOut != 5 {
		t.Errorf("TokensOut = %d, want 5", result.TokensOut)
	}
}

func TestClaudeProvider_Invoke_IsError(t *testing.T) {
	fakeInPath(t, "claude", "claude_is_error")
	p := NewClaudeProvider(nil)

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
	if provErr.Message != "Rate limit exceeded." {
		t.Errorf("Message = %q, want %q", provErr.Message, "Rate limit exceeded.")
	}
}

// TestClaudeProvider_RunOnce_SessionExpiredViaEvent verifies that a "No conversation found"
// event causes runOnce to return expired=true so Invoke can retry with a new session.
func TestClaudeProvider_RunOnce_SessionExpiredViaEvent(t *testing.T) {
	fakeInPath(t, "claude", "claude_session_expired")
	p := &ClaudeProvider{}

	result, expired, err := p.runOnce(context.Background(), InvokeRequest{
		UserMessage: "hello",
		SessionID:   "old-session-123",
		WorkDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("runOnce() unexpected error = %v", err)
	}
	if !expired {
		t.Error("runOnce() expired = false, want true")
	}
	if result.Text != "" || result.SessionID != "" {
		t.Errorf("runOnce() result = %+v, want empty on session expiry", result)
	}
}

// TestClaudeProvider_RunOnce_SessionExpiredViaExitFailure verifies that a non-zero exit
// while a session ID was provided is treated as session expiry so Invoke can retry.
func TestClaudeProvider_RunOnce_SessionExpiredViaExitFailure(t *testing.T) {
	fakeInPath(t, "claude", "claude_exit_failure")
	p := &ClaudeProvider{}

	_, expired, err := p.runOnce(context.Background(), InvokeRequest{
		UserMessage: "hello",
		SessionID:   "sess-with-id",
		WorkDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("runOnce() unexpected error = %v", err)
	}
	if !expired {
		t.Error("runOnce() expired = false, want true (exit failure with session is treated as expired)")
	}
}

// TestClaudeProvider_RunOnce_ExitFailureNoSession verifies that a non-zero exit
// without a session ID is a real error, not silently treated as expiry.
func TestClaudeProvider_RunOnce_ExitFailureNoSession(t *testing.T) {
	fakeInPath(t, "claude", "claude_exit_failure")
	p := &ClaudeProvider{}

	_, expired, err := p.runOnce(context.Background(), InvokeRequest{
		UserMessage: "hello",
		WorkDir:     t.TempDir(),
	})
	if err == nil {
		t.Fatal("runOnce() expected error for exit failure without session, got nil")
	}
	if expired {
		t.Error("runOnce() expired = true, want false (exit failure without session is a real error)")
	}
}
