package providers

import (
	"fmt"
	"time"

	"github.com/ectoclaw/ectoclaw/pkg/config"
	"github.com/ectoclaw/ectoclaw/pkg/logger"
)

// NewProvider constructs the appropriate Provider from the configuration.
func NewProvider(cfg *config.Config) (Provider, error) {
	var il *InvokeLogger
	if cfg.Logs.Agent {
		var err error
		il, err = NewInvokeLogger(cfg.WorkspacePath())
		if err != nil {
			logger.WarnCF("bridge", "Failed to open invoke log", map[string]any{"error": err.Error()})
		}
	}

	idleTimeout := time.Duration(cfg.Bridge.IdleTimeoutMinutes) * time.Minute
	maxRetries := cfg.Bridge.MaxRetries

	switch cfg.Bridge.Provider {
	case "codex":
		return NewCodexProvider(il, idleTimeout, maxRetries), nil
	case "", "claude":
		return NewClaudeProvider(il, idleTimeout, maxRetries), nil
	default:
		return nil, fmt.Errorf("unknown provider %q (supported: claude, codex)", cfg.Bridge.Provider)
	}
}
