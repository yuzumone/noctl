package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4"))

	StatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	DocStyle = lipgloss.NewStyle()

	FooterStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#353535")).
			Foreground(lipgloss.Color("#FFFFFF"))

	KeyStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#7D56F4")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1).
			Bold(true)

	KeyDescStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#353535")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)
)
