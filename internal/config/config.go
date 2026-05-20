// Package config provides functionality for loading and managing application configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	NotionToken       string `mapstructure:"notion_token" yaml:"notion_token"`
	DefaultDatabaseID string `mapstructure:"default_database_id" yaml:"default_database_id"`
}

// Load loads the configuration from environment variables and the config file.
func Load() (*Config, error) {
	viper.SetEnvPrefix("NOCTL")
	if err := viper.BindEnv("notion_token"); err != nil {
		return nil, fmt.Errorf("failed to bind env var notion_token: %w", err)
	}
	if err := viper.BindEnv("default_database_id"); err != nil {
		return nil, fmt.Errorf("failed to bind env var default_database_id: %w", err)
	}

	// Search locations
	home, err := os.UserHomeDir()
	if err == nil {
		viper.AddConfigPath(home)                                    // ~/.noctl.yaml
		viper.AddConfigPath(filepath.Join(home, ".config", "noctl")) // ~/.config/noctl/config.yaml
	}
	viper.SetConfigName(".noctl")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// If .noctl.yaml not found, try config.yaml in ~/.config/noctl/
			viper.SetConfigName("config")
			if err := viper.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
					return nil, fmt.Errorf("reading config file: %w", err)
				}
			}
		} else {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}

// GetConfigDir returns the primary directory where the config file is expected.
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home, nil
}
