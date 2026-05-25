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
calendar_database_ids:
  - id1
  - id2
theme:
  accent: "#7D56F4"
  select_bg: "#5F00FF"
`
	configPath := filepath.Join(tempHome, ".noctl.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// Mock home directory for viper
	origHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", tempHome); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}
	defer func() {
		if err := os.Setenv("HOME", origHome); err != nil {
			t.Errorf("failed to restore HOME: %v", err)
		}
	}()

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
	if len(cfg.CalendarDatabaseIDs) != 2 || cfg.CalendarDatabaseIDs[0] != "id1" || cfg.CalendarDatabaseIDs[1] != "id2" {
		t.Errorf("expected CalendarDatabaseIDs [id1 id2], got %v", cfg.CalendarDatabaseIDs)
	}
	if cfg.Theme.Accent != "#7D56F4" {
		t.Errorf("expected Theme.Accent #7D56F4, got %s", cfg.Theme.Accent)
	}
	if cfg.Theme.SelectBg != "#5F00FF" {
		t.Errorf("expected Theme.SelectBg #5F00FF, got %s", cfg.Theme.SelectBg)
	}
}

func TestConfigLoadEnv(t *testing.T) {
	viper.Reset()
	if err := os.Setenv("NOCTL_NOTION_TOKEN", "env_token"); err != nil {
		t.Fatalf("failed to set env var: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("NOCTL_NOTION_TOKEN"); err != nil {
			t.Errorf("failed to unset env var: %v", err)
		}
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.NotionToken != "env_token" {
		t.Errorf("expected NotionToken env_token, got %s", cfg.NotionToken)
	}
}
