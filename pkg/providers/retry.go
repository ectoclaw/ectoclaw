package providers

import (
	"context"
	"fmt"

	"github.com/ectoclaw/ectoclaw/pkg/logger"
)

// retryReason is returned by runOnce to signal whether and why the caller should retry.
type retryReason int

const (
	retryNone           retryReason = iota
	retrySessionExpired             // missing/expired session — retry clearing the session ID
	retryIdleTimeout                // subprocess produced no output within the idle deadline
)

// atomicRunner is the internal interface for a single subprocess attempt.
// Both ClaudeProvider and CodexProvider implement it; their Invoke methods
// delegate to invokeWithRetry, which owns the retry loop.
type atomicRunner interface {
	runOnce(ctx context.Context, req InvokeRequest) (InvokeResult, retryReason, error)
}

// invokeWithRetry executes runner.runOnce in a loop, retrying transparently on:
//   - idle timeout: up to maxRetries attempts; the partial session ID (if any) is carried
//     forward so the next attempt can resume where the previous one was interrupted.
//   - session expiry: at most once, without consuming an idle-retry slot.
//
// providerName is used only for log and error messages.
func invokeWithRetry(ctx context.Context, runner atomicRunner, req InvokeRequest, maxRetries int, providerName string) (InvokeResult, error) {
	maxAttempts := maxRetries
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	sessionRetried := false
	idleAttempts := 0
	for {
		result, reason, err := runner.runOnce(ctx, req)
		switch reason {
		case retrySessionExpired:
			if sessionRetried {
				return result, err
			}
			sessionRetried = true
			logger.WarnCF("bridge", "session expired, retrying without session ID", map[string]any{
				"provider":       providerName,
				"old_session_id": req.SessionID,
			})
			req.SessionID = ""

		case retryIdleTimeout:
			idleAttempts++
			if idleAttempts >= maxAttempts {
				return InvokeResult{}, fmt.Errorf("%s hung after %d idle timeout(s): %w",
					providerName, idleAttempts, errIdleTimeout)
			}
			logger.WarnCF("bridge", "idle timeout — retrying", map[string]any{
				"provider": providerName,
				"attempt":  idleAttempts,
				"max":      maxAttempts,
			})
			if result.SessionID != "" {
				req.SessionID = result.SessionID
			}

		default:
			return result, err
		}
	}
}
