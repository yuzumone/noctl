// Package tui provides terminal user interface components.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type selectorItem struct {
	name     string
	color    string
	selected bool
}

func (i selectorItem) Title() string {
	check := " "
	if i.selected {
		check = "x"
	}
	return fmt.Sprintf("[%s] %s", check, i.name)
}
func (i selectorItem) Description() string { return "" }
func (i selectorItem) FilterValue() string { return i.name }

// SelectorModel is a Bubble Tea model for selecting options from a list.
type SelectorModel struct {
	list     list.Model
	isMulti  bool
	propName string
	width    int
	height   int
}

// SelectorDoneMsg is sent when selection is finished.
type SelectorDoneMsg struct {
	PropName string
	Values   []string
}

// CancelSelectorMsg is sent when the selector is canceled.
type CancelSelectorMsg struct{}

// OpenSelectorMsg is sent to trigger opening the selector popup.
type OpenSelectorMsg struct {
	PropName      string
	IsMulti       bool
	CurrentValues []string
	Options       []SelectorOption
}

// SelectorOption represents a choice in the selector.
type SelectorOption struct {
	Name  string
	Color string
}

// NewSelectorModel creates a new SelectorModel.
func NewSelectorModel() SelectorModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("229")).
		UnsetBackground().
		BorderLeftForeground(lipgloss.Color("57")).
		Bold(false)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(true)
	l.Styles.Title = TitleStyle
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	return SelectorModel{
		list: l,
	}
}

// Init initializes the SelectorModel.
func (m SelectorModel) Init() tea.Cmd {
	return nil
}

// Update updates the SelectorModel.
func (m SelectorModel) Update(msg tea.Msg) (SelectorModel, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+n":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case "ctrl+p":
			msg = tea.KeyMsg{Type: tea.KeyUp}
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CancelSelectorMsg{} }
		case "space":
			if m.isMulti {
				idx := m.list.Cursor()
				if idx >= 0 && idx < len(m.list.Items()) {
					item := m.list.Items()[idx].(selectorItem)
					item.selected = !item.selected
					return m, m.list.SetItem(idx, item)
				}
				return m, nil
			}
			// For single select, space acts like enter
			return m, m.confirmSelection
		case "enter":
			return m, m.confirmSelection
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		popupWidth := int(float64(m.width) * 0.6)
		if popupWidth < 40 {
			popupWidth = m.width
		}
		popupHeight := int(float64(m.height) * 0.5)
		if popupHeight < 10 {
			popupHeight = m.height
		}
		m.list.SetSize(popupWidth-2, popupHeight-6)
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m SelectorModel) confirmSelection() tea.Msg {
	var values []string
	if m.isMulti {
		for _, it := range m.list.Items() {
			item := it.(selectorItem)
			if item.selected {
				values = append(values, item.name)
			}
		}
	} else {
		if i, ok := m.list.SelectedItem().(selectorItem); ok {
			values = append(values, i.name)
		}
	}
	return SelectorDoneMsg{PropName: m.propName, Values: values}
}

// View renders the SelectorModel as a popup.
func (m SelectorModel) View() string {
	popupWidth := int(float64(m.width) * 0.6)
	if popupWidth < 40 {
		popupWidth = m.width
	}
	popupHeight := int(float64(m.height) * 0.5)
	if popupHeight < 10 {
		popupHeight = m.height
	}

	footerHelps := []keyHelp{
		{"Enter", "Confirm"},
		{"Esc", "Cancel"},
	}
	if m.isMulti {
		footerHelps = append([]keyHelp{{"Space", "Toggle"}}, footerHelps...)
	}
	footer := renderFooter(popupWidth-2, footerHelps)

	content := lipgloss.JoinVertical(lipgloss.Left,
		m.list.View(),
		footer,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("57")).
		Width(popupWidth - 2).
		Height(popupHeight - 2).
		Render(content)
}

// SetOptions populates the selector with options.
func (m *SelectorModel) SetOptions(propName string, isMulti bool, options []SelectorOption, currentValues []string) {
	m.propName = propName
	m.isMulti = isMulti
	m.list.Title = "Select: " + propName

	currentMap := make(map[string]bool)
	for _, v := range currentValues {
		currentMap[strings.TrimSpace(v)] = true
	}

	items := make([]list.Item, len(options))
	for i, opt := range options {
		selected := currentMap[opt.Name]
		items[i] = selectorItem{
			name:     opt.Name,
			color:    opt.Color,
			selected: selected,
		}
	}
	m.list.SetItems(items)
	m.list.Select(0)
}
