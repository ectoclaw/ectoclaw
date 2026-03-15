package providers

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ectoclaw/ectoclaw/pkg/logger"
)

// ClaudeProvider invokes the `claude` CLI subprocess.
type ClaudeProvider struct {
	log         *InvokeLogger
	idleTimeout time.Duration
	maxRetries  int
}

// NewClaudeProvider returns a Provider backed by the claude CLI.
func NewClaudeProvider(log *InvokeLogger, idleTimeout time.Duration, maxRetries int) Provider {
	return &ClaudeProvider{log: log, idleTimeout: idleTimeout, maxRetries: maxRetries}
}

// Name returns the provider identifier.
func (p *ClaudeProvider) Name() string { return "claude" }

// claudeTypeEvent is used to peek at the event type before full decoding.
type claudeTypeEvent struct {
	Type string `json:"type"`
}

// claudeResultEvent holds the final "result" event decoded for session/token data.
type claudeResultEvent struct {
	Subtype   string            `json:"subtype"`
	SessionID string            `json:"session_id"`
	Result    string            `json:"result"`
	IsError   bool              `json:"is_error"`
	Errors    []string          `json:"errors"`
	Usage     claudeResultUsage `json:"usage"`
}

type claudeResultUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Invoke runs the claude CLI as a subprocess and returns the result.
// Retry logic (session expiry and idle timeout) is handled by invokeWithRetry.
func (p *ClaudeProvider) Invoke(ctx context.Context, req InvokeRequest) (InvokeResult, error) {
	return invokeWithRetry(ctx, p, req, p.maxRetries, "claude")
}

// runOnce performs a single subprocess invocation. The retryReason return value signals
// whether the caller should retry and why; retryNone means the result is final.
func (p *ClaudeProvider) runOnce(ctx context.Context, req InvokeRequest) (InvokeResult, retryReason, error) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "claude", p.buildArgs(req)...)
	cmd.Dir = req.WorkDir

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return InvokeResult{}, retryNone, err
	}

	p.log.LogInvoke(req.LogKey, "claude", req.SessionID, req.Model, req.UserMessage)

	if err := cmd.Start(); err != nil {
		return InvokeResult{}, retryNone, err
	}

	start := time.Now()
	var result claudeResultEvent
	for line := range scanWithIdleTimeout(cancel, stdout, p.idleTimeout) {
		p.log.LogLine(req.LogKey, line)
		var ev claudeTypeEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if ev.Type == "result" {
			_ = json.Unmarshal([]byte(line), &result)
		}
	}

	// If our cancel fired but the parent context is still alive, the idle timer triggered.
	// cmd.Wait() is called to reap the process; the error is expected (signal: killed).
	if runCtx.Err() != nil && ctx.Err() == nil {
		_ = cmd.Wait() // process was killed; discard the "signal: killed" error
		return InvokeResult{SessionID: result.SessionID}, retryIdleTimeout, errIdleTimeout
	}

	// Check for session-not-found before inspecting the exit code — the parsed
	// event gives a more precise signal than a generic non-zero exit.
	if result.Subtype == "error_during_execution" {
		for _, errMsg := range result.Errors {
			if strings.Contains(errMsg, "No conversation found") {
				_ = cmd.Wait()
				return InvokeResult{}, retrySessionExpired, nil
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		stderr := stderrBuf.String()
		p.log.LogStderr(req.LogKey, stderr)
		logger.WarnCF("bridge", "claude subprocess failed", map[string]any{
			"error":  err.Error(),
			"stderr": stderr,
		})
		// Unknown exit failure with a session — treat as expired so the caller retries.
		if req.SessionID != "" {
			return InvokeResult{}, retrySessionExpired, nil
		}
		return InvokeResult{}, retryNone, err
	}

	if s := stderrBuf.String(); s != "" {
		p.log.LogStderr(req.LogKey, s)
		if logger.GetLevel() == logger.DEBUG {
			os.Stderr.WriteString(s)
		}
	}

	// is_error:true with a non-empty result means claude returned a human-readable
	// error message (e.g. rate limit, out of credits). Surface it to the user directly.
	if result.IsError && result.Result != "" {
		return InvokeResult{}, retryNone, &ErrProviderMessage{Message: result.Result}
	}

	duration := time.Since(start)
	p.log.LogDone(req.LogKey, result.SessionID, result.Usage.InputTokens, result.Usage.OutputTokens, len(result.Result), duration)

	return InvokeResult{
		SessionID: result.SessionID,
		Text:      result.Result,
		TokensIn:  result.Usage.InputTokens,
		TokensOut: result.Usage.OutputTokens,
	}, retryNone, nil
}

func (p *ClaudeProvider) buildArgs(req InvokeRequest) []string {
	args := []string{
		"-p", req.UserMessage,
		"--output-format", "stream-json",
		"--verbose",
		"--dangerously-skip-permissions",
	}
	if req.SessionID != "" {
		args = append([]string{"--resume", req.SessionID}, args...)
	}
	if req.SystemPrompt != "" {
		args = append(args, "--system-prompt", req.SystemPrompt)
	}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	return args
}
