package tui

import (
	"sort"
	"strings"

	"noctl/internal/notion"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"
)

type TableSelectorModel struct {
	table  table.Model
	pages  []notionapi.Page
	width  int
	height int
	title  string
}

func NewTableSelectorModel() TableSelectorModel {
	t := table.New(
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(ActiveTheme.Dimmed)).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(ActiveTheme.SelectFg)).
		Background(lipgloss.Color(ActiveTheme.SelectBg)).
		Bold(false)
	t.SetStyles(s)

	return TableSelectorModel{
		table: t,
	}
}

func (m TableSelectorModel) Init() tea.Cmd {
	return nil
}

func (m TableSelectorModel) Update(msg tea.Msg) (TableSelectorModel, tea.Cmd) {
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
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CancelSelectorMsg{} }
		case "enter":
			idx := m.table.Cursor()
			return m, func() tea.Msg {
				return SelectorDoneMsg{
					SelectedIndex: idx,
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateTableSize()
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *TableSelectorModel) updateTableSize() {
	popupWidth := int(float64(m.width) * 0.8)
	if popupWidth < 60 {
		popupWidth = m.width
	}
	popupHeight := int(float64(m.height) * 0.6)
	if popupHeight < 15 {
		popupHeight = m.height
	}

	m.table.SetWidth(popupWidth - 4)
	m.table.SetHeight(popupHeight - 8)
	
	if len(m.pages) > 0 {
		m.renderPages()
	}
}

func (m *TableSelectorModel) SetPages(title string, pages []notionapi.Page) {
	m.title = title
	m.pages = pages
	m.renderPages()
}

func (m *TableSelectorModel) renderPages() {
	if len(m.pages) == 0 {
		return
	}

	popupWidth := int(float64(m.width) * 0.8)
	if popupWidth < 60 {
		popupWidth = m.width
	}
	contentWidth := popupWidth - 4

	// Logic adapted from RecordsModel.updateTable
	propertyKeys := []string{}
	titleKey := ""
	for k, p := range m.pages[0].Properties {
		if _, ok := p.(*notionapi.TitleProperty); ok {
			titleKey = k
			break
		}
	}
	if titleKey != "" {
		propertyKeys = append(propertyKeys, titleKey)
	}
	var otherKeys []string
	for k := range m.pages[0].Properties {
		if k == titleKey {
			continue
		}
		otherKeys = append(otherKeys, k)
	}
	sort.Strings(otherKeys)
	propertyKeys = append(propertyKeys, otherKeys...)

	numCols := len(propertyKeys)
	if numCols > 0 {
		maxContentWidths := make(map[string]int)
		for _, k := range propertyKeys {
			maxW := len(k)
			for i := 0; i < len(m.pages); i++ {
				valLen := len(notion.PropertyToString(m.pages[i].Properties[k]))
				if valLen > maxW {
					maxW = valLen
				}
			}
			maxContentWidths[k] = maxW
		}

		totalContentWidth := 0
		for _, w := range maxContentWidths {
			totalContentWidth += w + 2
		}

		columns := []table.Column{}
		remainingWidth := contentWidth

		for i, k := range propertyKeys {
			var w int
			if totalContentWidth > 0 {
				contentW := maxContentWidths[k] + 2
				if i == numCols-1 {
					w = remainingWidth
				} else {
					w = int(float64(contentW) / float64(totalContentWidth) * float64(contentWidth))
					if w < 5 {
						w = 5
					}
				}
			} else {
				w = contentWidth / numCols
			}
			columns = append(columns, table.Column{Title: k, Width: w})
			remainingWidth -= w
		}
		m.table.SetColumns(columns)
	}

	rows := []table.Row{}
	for _, page := range m.pages {
		row := table.Row{}
		for _, k := range propertyKeys {
			row = append(row, notion.PropertyToString(page.Properties[k]))
		}
		rows = append(rows, row)
	}
	m.table.SetRows(rows)
}

func (m TableSelectorModel) View() string {
	popupWidth := int(float64(m.width) * 0.8)
	if popupWidth < 60 {
		popupWidth = m.width
	}
	popupHeight := int(float64(m.height) * 0.6)
	if popupHeight < 15 {
		popupHeight = m.height
	}

	header := TitleStyle.Width(popupWidth - 2).Padding(0, 1).Render(m.title)
	footer := renderFooter(popupWidth-2, []keyHelp{
		{"Enter", "Detail"},
		{"Esc", "Back"},
	})

	internalHeight := popupHeight - 2
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		lipgloss.NewStyle().Height(internalHeight-headerHeight(header)-1).Render(m.table.View()),
		footer,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ActiveTheme.Border)).
		Width(popupWidth - 2).
		Height(popupHeight - 2).
		Render(content)
}

func headerHeight(s string) int {
	return len(strings.Split(s, "\n"))
}
