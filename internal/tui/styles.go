// Package tui provides terminal user interface components.
package tui

import (
	"noctl/internal/config"

	"github.com/charmbracelet/lipgloss"
)

// Theme defines the color configuration structure.
type Theme struct {
	// Accent is the main accent color.
	Accent string
	// AccentText is the foreground text color on accent background.
	AccentText string
	// SelectBg is the background color for selected items.
	SelectBg string
	// SelectFg is the foreground color for selected items.
	SelectFg string
	// Border is the border color for windows and popups.
	Border string
	// Dimmed is the color for minor/dimmed text.
	Dimmed string
	// TitleBg is the title bar background color.
	TitleBg string
	// TitleFg is the title bar text color.
	TitleFg string
	// FooterBg is the footer background color.
	FooterBg string
	// FooterFg is the footer text color.
	FooterFg string
	// KeyBg is the background color for help key badges.
	KeyBg string
	// KeyFg is the text color for help key badges.
	KeyFg string
	// Error is the error text color.
	Error string
	// Label is the field label text color.
	Label string
}

// ActiveTheme holds the actual colors used by the application, defaulting to the original theme.
var ActiveTheme = Theme{
	Accent:     "#7D56F4",
	AccentText: "#FAFAFA",
	SelectBg:   "#5F00FF",
	SelectFg:   "#FFFFDF",
	Border:     "#5F00FF",
	Dimmed:     "#585858",
	TitleBg:    "#7D56F4",
	TitleFg:    "#FAFAFA",
	FooterBg:   "#353535",
	FooterFg:   "#FFFFFF",
	KeyBg:      "#7D56F4",
	KeyFg:      "#FFFFFF",
	Error:      "#FF0000",
	Label:      "#875FFF",
}

var (
	// TitleStyle is the style for section titles.
	TitleStyle lipgloss.Style
	// StatusStyle is the style for status messages.
	StatusStyle lipgloss.Style
	// ErrorStyle is the style for error messages.
	ErrorStyle lipgloss.Style
	// DocStyle is the basic document style.
	DocStyle lipgloss.Style
	// FooterStyle is the style for the application footer.
	FooterStyle lipgloss.Style
	// KeyStyle is the style for key labels in help.
	KeyStyle lipgloss.Style
	// KeyDescStyle is the style for key descriptions in help.
	KeyDescStyle lipgloss.Style
	// DimmedStyle is the style for dimmed or minor text.
	DimmedStyle lipgloss.Style
	// LabelStyle is the style for field labels in details view.
	LabelStyle lipgloss.Style
)

func init() {
	// Initialize with default theme
	InitStyles(&config.Config{})
}

// InitStyles initializes the global styles with the given configuration.
func InitStyles(cfg *config.Config) {
	// If theme configuration is present, overwrite active theme
	if cfg.Theme.Accent != "" {
		ActiveTheme.Accent = cfg.Theme.Accent
	}
	if cfg.Theme.AccentText != "" {
		ActiveTheme.AccentText = cfg.Theme.AccentText
	}
	if cfg.Theme.SelectBg != "" {
		ActiveTheme.SelectBg = cfg.Theme.SelectBg
	}
	if cfg.Theme.SelectFg != "" {
		ActiveTheme.SelectFg = cfg.Theme.SelectFg
	}
	if cfg.Theme.Border != "" {
		ActiveTheme.Border = cfg.Theme.Border
	}
	if cfg.Theme.Dimmed != "" {
		ActiveTheme.Dimmed = cfg.Theme.Dimmed
	}
	if cfg.Theme.TitleBg != "" {
		ActiveTheme.TitleBg = cfg.Theme.TitleBg
	}
	if cfg.Theme.TitleFg != "" {
		ActiveTheme.TitleFg = cfg.Theme.TitleFg
	}
	if cfg.Theme.FooterBg != "" {
		ActiveTheme.FooterBg = cfg.Theme.FooterBg
	}
	if cfg.Theme.FooterFg != "" {
		ActiveTheme.FooterFg = cfg.Theme.FooterFg
	}
	if cfg.Theme.KeyBg != "" {
		ActiveTheme.KeyBg = cfg.Theme.KeyBg
	}
	if cfg.Theme.KeyFg != "" {
		ActiveTheme.KeyFg = cfg.Theme.KeyFg
	}
	if cfg.Theme.Error != "" {
		ActiveTheme.Error = cfg.Theme.Error
	}
	if cfg.Theme.Label != "" {
		ActiveTheme.Label = cfg.Theme.Label
	}

	// Re-initialize styles
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ActiveTheme.TitleFg)).
		Background(lipgloss.Color(ActiveTheme.TitleBg))

	StatusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ActiveTheme.Dimmed))

	ErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ActiveTheme.Error))

	DocStyle = lipgloss.NewStyle()

	FooterStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(ActiveTheme.FooterBg)).
		Foreground(lipgloss.Color(ActiveTheme.FooterFg))

	KeyStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(ActiveTheme.KeyBg)).
		Foreground(lipgloss.Color(ActiveTheme.KeyFg)).
		Padding(0, 1).
		Bold(true)

	KeyDescStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(ActiveTheme.FooterBg)).
		Foreground(lipgloss.Color(ActiveTheme.FooterFg)).
		Padding(0, 1)

	DimmedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ActiveTheme.Dimmed))

	LabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ActiveTheme.Label)).
		Bold(true)
}
