package onboard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyEmbeddedToTarget(t *testing.T) {
	targetDir := t.TempDir()

	if err := copyEmbeddedToTarget(targetDir); err != nil {
		t.Fatalf("copyEmbeddedToTarget() error = %v", err)
	}

	systemPath := filepath.Join(targetDir, "SYSTEM.md")
	if _, err := os.Stat(systemPath); err != nil {
		t.Fatalf("expected %s to exist: %v", systemPath, err)
	}
}
