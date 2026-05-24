package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmYesMsg is sent when the user confirms Yes.
type ConfirmYesMsg struct{}

// ConfirmNoMsg is sent when the user confirms No.
type ConfirmNoMsg struct{}

// ConfirmModel is a Yes/No confirmation dialog component.
type ConfirmModel struct {
	prompt  string
	cursor  int // 0 = Yes, 1 = No
	width   int
	height  int
}

// NewConfirmModel creates a new ConfirmModel with the given prompt.
func NewConfirmModel(prompt string) ConfirmModel {
	return ConfirmModel{
		prompt: prompt,
		cursor: 1, // Default to No for safety
	}
}

// Init initializes the ConfirmModel.
func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

// Update handles key events for the ConfirmModel.
func (m ConfirmModel) Update(msg tea.Msg) (ConfirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			return m, func() tea.Msg { return ConfirmYesMsg{} }
		case "n", "N", "esc":
			return m, func() tea.Msg { return ConfirmNoMsg{} }
		case "left", "h", "shift+tab":
			m.cursor = 0
		case "right", "l", "tab":
			m.cursor = 1
		case "enter":
			if m.cursor == 0 {
				return m, func() tea.Msg { return ConfirmYesMsg{} }
			}
			return m, func() tea.Msg { return ConfirmNoMsg{} }
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renders the confirmation dialog.
func (m ConfirmModel) View() string {
	borderColor := lipgloss.Color(ActiveTheme.Border)
	accentColor := lipgloss.Color(ActiveTheme.Accent)
	accentTextColor := lipgloss.Color(ActiveTheme.AccentText)
	footerBg := lipgloss.Color(ActiveTheme.FooterBg)
	footerFg := lipgloss.Color(ActiveTheme.FooterFg)

	promptStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentTextColor).
		Padding(1, 2)

	yesStyle := lipgloss.NewStyle().Padding(0, 2)
	noStyle := lipgloss.NewStyle().Padding(0, 2)

	selectedStyle := lipgloss.NewStyle().
		Background(accentColor).
		Foreground(accentTextColor).
		Padding(0, 2).
		Bold(true)

	var yesLabel, noLabel string
	if m.cursor == 0 {
		yesLabel = selectedStyle.Render("Yes")
		noLabel = noStyle.Foreground(footerFg).Background(footerBg).Render("No")
	} else {
		yesLabel = yesStyle.Foreground(footerFg).Background(footerBg).Render("Yes")
		noLabel = selectedStyle.Render("No")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Top, yesLabel, "  ", noLabel)

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ActiveTheme.Dimmed)).
		Padding(0, 2).
		Render("y/n · ←/→ to move · Enter to confirm")

	inner := lipgloss.JoinVertical(lipgloss.Center,
		promptStyle.Render(m.prompt),
		strings.Repeat(" ", 1),
		lipgloss.NewStyle().Padding(0, 2).Render(buttons),
		strings.Repeat(" ", 1),
		hint,
	)

	popupWidth := 50
	if m.width > 0 && m.width < popupWidth+4 {
		popupWidth = m.width - 4
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(popupWidth).
		Align(lipgloss.Center).
		Render(inner)
}
