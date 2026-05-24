// Package config provides functionality for loading and managing application configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// ThemeConfig holds the color theme configuration.
type ThemeConfig struct {
	// Accent is the main accent color.
	Accent string `mapstructure:"accent" yaml:"accent"`
	// AccentText is the foreground text color on accent background.
	AccentText string `mapstructure:"accent_text" yaml:"accent_text"`
	// SelectBg is the background color for selected items.
	SelectBg string `mapstructure:"select_bg" yaml:"select_bg"`
	// SelectFg is the foreground color for selected items.
	SelectFg string `mapstructure:"select_fg" yaml:"select_fg"`
	// Border is the border color for windows and popups.
	Border string `mapstructure:"border" yaml:"border"`
	// Dimmed is the color for minor/dimmed text.
	Dimmed string `mapstructure:"dimmed" yaml:"dimmed"`
	// TitleBg is the title bar background color.
	TitleBg string `mapstructure:"title_bg" yaml:"title_bg"`
	// TitleFg is the title bar text color.
	TitleFg string `mapstructure:"title_fg" yaml:"title_fg"`
	// FooterBg is the footer background color.
	FooterBg string `mapstructure:"footer_bg" yaml:"footer_bg"`
	// FooterFg is the footer text color.
	FooterFg string `mapstructure:"footer_fg" yaml:"footer_fg"`
	// KeyBg is the background color for help key badges.
	KeyBg string `mapstructure:"key_bg" yaml:"key_bg"`
	// KeyFg is the text color for help key badges.
	KeyFg string `mapstructure:"key_fg" yaml:"key_fg"`
	// Error is the error text color.
	Error string `mapstructure:"error" yaml:"error"`
	// Label is the field label text color.
	Label string `mapstructure:"label" yaml:"label"`
}

// Config holds the application configuration.
type Config struct {
	// NotionToken is the API token for Notion integration.
	NotionToken string `mapstructure:"notion_token" yaml:"notion_token"`
	// DefaultDatabaseID is the default database ID to load.
	DefaultDatabaseID string `mapstructure:"default_database_id" yaml:"default_database_id"`
	// Theme is the custom color theme configuration.
	Theme ThemeConfig `mapstructure:"theme" yaml:"theme"`
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
