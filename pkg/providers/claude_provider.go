package providers

import (
	"bufio"
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
	log *InvokeLogger
}

// NewClaudeProvider returns a Provider backed by the claude CLI.
func NewClaudeProvider(log *InvokeLogger) Provider {
	return &ClaudeProvider{log: log}
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
// If the session has expired it silently retries with a new session.
func (p *ClaudeProvider) Invoke(ctx context.Context, req InvokeRequest) (InvokeResult, error) {
	result, sessionExpired, err := p.runOnce(ctx, req)
	if sessionExpired {
		logger.WarnCF("bridge", "Claude session expired, retrying with new session", map[string]any{
			"old_session_id": req.SessionID,
		})
		req.SessionID = ""
		result, _, err = p.runOnce(ctx, req)
	}
	return result, err
}

// runOnce performs a single subprocess invocation. The second return value is true
// when the failure is specifically a missing/expired session so Invoke can retry.
func (p *ClaudeProvider) runOnce(ctx context.Context, req InvokeRequest) (InvokeResult, bool, error) {
	cmd := exec.CommandContext(ctx, "claude", p.buildArgs(req)...)
	cmd.Dir = req.WorkDir

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return InvokeResult{}, false, err
	}

	p.log.LogInvoke(req.LogKey, "claude", req.SessionID, req.Model, req.UserMessage)

	if err := cmd.Start(); err != nil {
		return InvokeResult{}, false, err
	}

	start := time.Now()
	var result claudeResultEvent
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		p.log.LogLine(req.LogKey, line)
		// Capture the result event for session/token data.
		var ev claudeTypeEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if ev.Type == "result" {
			_ = json.Unmarshal([]byte(line), &result)
		}
	}

	// Check for session-not-found before inspecting the exit code — the parsed
	// event gives a more precise signal than a generic non-zero exit.
	if result.Subtype == "error_during_execution" {
		for _, errMsg := range result.Errors {
			if strings.Contains(errMsg, "No conversation found") {
				_ = cmd.Wait()
				return InvokeResult{}, true, nil
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
			return InvokeResult{}, true, nil
		}
		return InvokeResult{}, false, err
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
		return InvokeResult{}, false, &ErrProviderMessage{Message: result.Result}
	}

	duration := time.Since(start)
	p.log.LogDone(req.LogKey, result.SessionID, result.Usage.InputTokens, result.Usage.OutputTokens, len(result.Result), duration)

	return InvokeResult{
		SessionID: result.SessionID,
		Text:      result.Result,
		TokensIn:  result.Usage.InputTokens,
		TokensOut: result.Usage.OutputTokens,
	}, false, nil
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
