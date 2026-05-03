package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadAppliesDefaults(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	content := []byte(`sources:
  - name: stdin
    type: stdin
`)

	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.App.BufferSize != 4096 {
		t.Fatalf("unexpected buffer size: %d", cfg.App.BufferSize)
	}
	if cfg.Detection.Window != 5*time.Minute {
		t.Fatalf("unexpected default detection window: %s", cfg.Detection.Window)
	}
	if cfg.Alerts.RateLimit != 15*time.Minute {
		t.Fatalf("unexpected default alert rate limit: %s", cfg.Alerts.RateLimit)
	}
}
