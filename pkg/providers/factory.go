package providers

import (
	"fmt"

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

	switch cfg.Bridge.Provider {
	case "codex":
		return NewCodexProvider(il), nil
	case "", "claude":
		return NewClaudeProvider(il), nil
	default:
		return nil, fmt.Errorf("unknown provider %q (supported: claude, codex)", cfg.Bridge.Provider)
	}
}
