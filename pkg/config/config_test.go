package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDefaultConfig_HeartbeatEnabled(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Heartbeat.Enabled {
		t.Error("Heartbeat should be enabled by default")
	}
}

func TestDefaultConfig_WorkspacePath(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Bridge.Workspace == "" {
		t.Error("Workspace should not be empty")
	}
}

func TestDefaultConfig_Gateway(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Gateway.Host != "127.0.0.1" {
		t.Error("Gateway host should have default value")
	}
	if cfg.Gateway.Port == 0 {
		t.Error("Gateway port should have default value")
	}
}


func TestDefaultConfig_Channels(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Channels.Telegram.Enabled {
		t.Error("Telegram should be disabled by default")
	}
	if cfg.Channels.Discord.Enabled {
		t.Error("Discord should be disabled by default")
	}
	if cfg.Channels.Slack.Enabled {
		t.Error("Slack should be disabled by default")
	}
	if cfg.Channels.Matrix.Enabled {
		t.Error("Matrix should be disabled by default")
	}
}

func TestSaveConfig_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission bits are not enforced on Windows")
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("config file has permission %04o, want 0600", perm)
	}
}

func TestConfig_Complete(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Bridge.Workspace == "" {
		t.Error("Workspace should not be empty")
	}
	if cfg.Bridge.Model != "" {
		t.Error("Model should be empty by default")
	}
	if cfg.Gateway.Host != "127.0.0.1" {
		t.Error("Gateway host should have default value")
	}
	if cfg.Gateway.Port == 0 {
		t.Error("Gateway port should have default value")
	}
	if !cfg.Heartbeat.Enabled {
		t.Error("Heartbeat should be enabled by default")
	}
}

func TestDefaultConfig_WorkspacePath_Default(t *testing.T) {
	t.Setenv("ECTOCLAW_HOME", "")
	t.Setenv("HOME", "/tmp/home")

	cfg := DefaultConfig()
	want := filepath.Join("/tmp/home", ".ectoclaw", "workspace")

	if cfg.Bridge.Workspace != want {
		t.Errorf("Default workspace path = %q, want %q", cfg.Bridge.Workspace, want)
	}
}

func TestDefaultConfig_WorkspacePath_WithEctoclawHome(t *testing.T) {
	t.Setenv("ECTOCLAW_HOME", "/custom/ectoclaw/home")

	cfg := DefaultConfig()
	want := "/custom/ectoclaw/home/workspace"

	if cfg.Bridge.Workspace != want {
		t.Errorf("Workspace path with ECTOCLAW_HOME = %q, want %q", cfg.Bridge.Workspace, want)
	}
}
