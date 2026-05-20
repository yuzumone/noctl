// Package tui provides terminal user interface components.
package tui

import "github.com/charmbracelet/lipgloss"

var (
	// TitleStyle is the style for section titles.
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4"))

	// StatusStyle is the style for status messages.
	StatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	// ErrorStyle is the style for error messages.
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	// DocStyle is the basic document style.
	DocStyle = lipgloss.NewStyle()

	// FooterStyle is the style for the application footer.
	FooterStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#353535")).
			Foreground(lipgloss.Color("#FFFFFF"))

	// KeyStyle is the style for key labels in help.
	KeyStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#7D56F4")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1).
			Bold(true)

	// KeyDescStyle is the style for key descriptions in help.
	KeyDescStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#353535")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)
)
