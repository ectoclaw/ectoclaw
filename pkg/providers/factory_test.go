package providers

import (
	"testing"

	"github.com/ectoclaw/ectoclaw/pkg/config"
)

func TestNewProvider_DefaultIsClaude(t *testing.T) {
	cfg := &config.Config{}
	p, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	if p.Name() != "claude" {
		t.Errorf("Name() = %q, want %q", p.Name(), "claude")
	}
}

func TestNewProvider_ExplicitClaude(t *testing.T) {
	cfg := &config.Config{Bridge: config.BridgeConfig{Provider: "claude"}}
	p, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	if p.Name() != "claude" {
		t.Errorf("Name() = %q, want %q", p.Name(), "claude")
	}
}

func TestNewProvider_Codex(t *testing.T) {
	cfg := &config.Config{Bridge: config.BridgeConfig{Provider: "codex"}}
	p, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	if p.Name() != "codex" {
		t.Errorf("Name() = %q, want %q", p.Name(), "codex")
	}
}

func TestNewProvider_UnknownReturnsError(t *testing.T) {
	cfg := &config.Config{Bridge: config.BridgeConfig{Provider: "gpt5"}}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("NewProvider() expected error for unknown provider, got nil")
	}
}
