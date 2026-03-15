package providers

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/ectoclaw/ectoclaw/pkg/logger"
)

// CodexProvider invokes the OpenAI `codex` CLI subprocess via `codex exec`.
type CodexProvider struct {
	log         *InvokeLogger
	idleTimeout time.Duration
	maxRetries  int
}

// NewCodexProvider returns a Provider backed by the codex CLI.
func NewCodexProvider(log *InvokeLogger, idleTimeout time.Duration, maxRetries int) Provider {
	return &CodexProvider{log: log, idleTimeout: idleTimeout, maxRetries: maxRetries}
}

// Name returns the provider identifier.
func (p *CodexProvider) Name() string { return "codex" }

// codex exec --json emits these JSONL event types.
type codexEvent struct {
	Type     string      `json:"type"`
	ThreadID string      `json:"thread_id"` // thread.started
	Item     *codexItem  `json:"item"`      // item.completed
	Usage    *codexUsage `json:"usage"`     // turn.completed
}

type codexItem struct {
	Type string `json:"type"` // "agent_message", "tool_call", etc.
	Text string `json:"text"`
}

type codexUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Invoke runs `codex exec` (or `codex exec resume`) as a subprocess and returns the result.
// Retry logic (idle timeout) is handled by invokeWithRetry.
func (p *CodexProvider) Invoke(ctx context.Context, req InvokeRequest) (InvokeResult, error) {
	return invokeWithRetry(ctx, p, req, p.maxRetries, "codex")
}

// runOnce performs a single subprocess invocation. The retryReason return value signals
// whether the caller should retry and why; retryNone means the result is final.
func (p *CodexProvider) runOnce(ctx context.Context, req InvokeRequest) (InvokeResult, retryReason, error) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "codex", p.buildArgs(req)...)
	cmd.Dir = req.WorkDir

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return InvokeResult{}, retryNone, err
	}

	p.log.LogInvoke(req.LogKey, "codex", req.SessionID, req.Model, req.UserMessage)

	if err := cmd.Start(); err != nil {
		return InvokeResult{}, retryNone, err
	}

	start := time.Now()
	var sessionID string
	var lastAgentMessage string
	var usage codexUsage
	for line := range scanWithIdleTimeout(cancel, stdout, p.idleTimeout) {
		var ev codexEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		p.log.LogLine(req.LogKey, line)
		switch ev.Type {
		case "thread.started":
			sessionID = ev.ThreadID
		case "item.completed":
			// Codex emits agent_message items for mid-turn narration ("I'll edit X now…")
			// and for the final user-facing reply. Only the last one is the actual answer.
			if ev.Item != nil && ev.Item.Type == "agent_message" && ev.Item.Text != "" {
				lastAgentMessage = ev.Item.Text
			}
		case "turn.completed":
			if ev.Usage != nil {
				usage = *ev.Usage
			}
		}
	}

	// If our cancel fired but the parent context is still alive, the idle timer triggered.
	// cmd.Wait() is called to reap the process; the error is expected (signal: killed).
	if runCtx.Err() != nil && ctx.Err() == nil {
		_ = cmd.Wait() // process was killed; discard the "signal: killed" error
		return InvokeResult{SessionID: sessionID}, retryIdleTimeout, errIdleTimeout
	}

	if err := cmd.Wait(); err != nil {
		stderr := strings.TrimSpace(stderrBuf.String())
		p.log.LogStderr(req.LogKey, stderr)
		logger.WarnCF("bridge", "codex subprocess failed", map[string]any{
			"error":  err.Error(),
			"stderr": stderr,
		})
		if stderr != "" {
			return InvokeResult{}, retryNone, &ErrProviderMessage{Message: stderr}
		}
		return InvokeResult{}, retryNone, err
	}

	duration := time.Since(start)
	p.log.LogDone(req.LogKey, sessionID, usage.InputTokens, usage.OutputTokens, len(lastAgentMessage), duration)

	return InvokeResult{
		SessionID: sessionID,
		Text:      lastAgentMessage,
		TokensIn:  usage.InputTokens,
		TokensOut: usage.OutputTokens,
	}, retryNone, nil
}

func (p *CodexProvider) buildArgs(req InvokeRequest) []string {
	// Codex has no --system-prompt flag, so we embed it in the message using explicit delimiters.
	// The framing instructs the model to treat it as background context, not user input,
	// and never to repeat or summarise it unless directly asked.
	message := req.UserMessage
	if req.SystemPrompt != "" {
		message = "<system>\n" +
			"The following is your background context and instructions. " +
			"Treat it as your own internal knowledge — never quote, summarise, or mention it unless the user explicitly asks.\n\n" +
			req.SystemPrompt +
			"\n</system>\n\n" +
			req.UserMessage
	}

	baseFlags := []string{
		"--json",
		"--dangerously-bypass-approvals-and-sandbox",
		"--skip-git-repo-check",
	}
	if req.Stateless {
		baseFlags = append(baseFlags, "--ephemeral")
	}
	if req.Model != "" {
		baseFlags = append(baseFlags, "--model", req.Model)
	}
	// WorkDir is handled via cmd.Dir — codex exec resume does not support -C.

	if req.SessionID != "" {
		// codex exec resume <session_id> <prompt> [flags]
		args := []string{"exec", "resume", req.SessionID, message}
		return append(args, baseFlags...)
	}

	// codex exec [flags] <prompt>
	args := append([]string{"exec"}, baseFlags...)
	return append(args, message)
}
