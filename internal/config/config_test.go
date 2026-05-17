package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestConfigLoad(t *testing.T) {
	// Create a temporary config file
	tempHome := t.TempDir()
	configContent := `
notion_token: secret_test_token
default_database_id: test_db_id
`
	configPath := filepath.Join(tempHome, ".noctl.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// Mock home directory for viper
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", origHome)

	// Reset viper for testing
	viper.Reset()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.NotionToken != "secret_test_token" {
		t.Errorf("expected NotionToken secret_test_token, got %s", cfg.NotionToken)
	}
	if cfg.DefaultDatabaseID != "test_db_id" {
		t.Errorf("expected DefaultDatabaseID test_db_id, got %s", cfg.DefaultDatabaseID)
	}
}

func TestConfigLoadEnv(t *testing.T) {
	viper.Reset()
	os.Setenv("NOCTL_NOTION_TOKEN", "env_token")
	defer os.Unsetenv("NOCTL_NOTION_TOKEN")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.NotionToken != "env_token" {
		t.Errorf("expected NotionToken env_token, got %s", cfg.NotionToken)
	}
}
