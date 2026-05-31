// Package tui provides terminal user interface components.
package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"noctl/internal/notion"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"
)

type RecordsModel struct {
	table       table.Model
	filterInput textinput.Model
	client      *notion.Client
	dbID        string
	dbName      string
	allPages    []notionapi.Page
	pages       []notionapi.Page
	filtering   bool
	loading     bool
	err         error
	width       int
	height      int
}

type recordsMsg []notionapi.Page
type SelectPageMsg struct {
	Page *notionapi.Page
}
type CreateRecordMsg struct {
	DatabaseID string
}
type EditRecordMsg struct {
	Page *notionapi.Page
}

func NewRecordsModel(client *notion.Client) RecordsModel {
	columns := []table.Column{
		{Title: "Title", Width: 30},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
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

	ti := textinput.New()
	ti.Placeholder = "Filter records..."
	ti.Prompt = "/ "

	return RecordsModel{
		table:       t,
		filterInput: ti,
		client:      client,
	}
}

func (m RecordsModel) Init() tea.Cmd {
	return nil
}

// SetDatabase sets the database for the records view and triggers fetching of all pages.
func (m *RecordsModel) SetDatabase(id string, name string, cache []notionapi.Page) tea.Cmd {
	m.dbID = id
	m.dbName = name
	m.loading = true
	m.allPages = nil
	m.pages = nil
	m.filterInput.SetValue("")
	m.table.SetRows([]table.Row{})

	if len(cache) > 0 {
		m.loading = false
		m.allPages = cache
		m.updateTable()
		m.table.SetCursor(0)
		return nil
	}

	m.table.SetColumns([]table.Column{{Title: "Loading all records...", Width: m.width}})
	return m.fetchAllRecords
}

func (m RecordsModel) fetchAllRecords() tea.Msg {
	pages, err := m.client.QueryDatabaseAll(context.Background(), m.dbID)
	if err != nil {
		return errMsg(fmt.Errorf("failed to fetch records for %s: %w", m.dbName, err))
	}
	return recordsMsg(pages)
}

func (m *RecordsModel) updateTable() {
	if len(m.allPages) == 0 {
		m.table.SetRows([]table.Row{})
		m.table.SetColumns([]table.Column{{Title: "No results", Width: 20}})
		return
	}

	// Dynamically create columns based on properties of the first page
	propertyKeys := []string{}

	// Find title property first to put it in the first column
	titleKey := ""
	for k, p := range m.allPages[0].Properties {
		if _, ok := p.(*notionapi.TitleProperty); ok {
			titleKey = k
			break
		}
	}

	if titleKey != "" {
		propertyKeys = append(propertyKeys, titleKey)
	}

	// Add other properties sorted alphabetically
	var otherKeys []string
	for k := range m.allPages[0].Properties {
		if k == titleKey {
			continue
		}
		otherKeys = append(otherKeys, k)
	}
	sort.Strings(otherKeys)
	propertyKeys = append(propertyKeys, otherKeys...)

	// Calculate column widths to fill terminal and minimize truncation
	numCols := len(propertyKeys)
	if numCols > 0 {
		// 1. Calculate max content width for each column (sample first 10 rows)
		maxContentWidths := make(map[string]int)
		for _, k := range propertyKeys {
			maxW := len(k) // Start with header length
			sampleCount := 10
			if len(m.allPages) < sampleCount {
				sampleCount = len(m.allPages)
			}
			for i := 0; i < sampleCount; i++ {
				valLen := len(notion.PropertyToString(m.allPages[i].Properties[k]))
				if valLen > maxW {
					maxW = valLen
				}
			}
			maxContentWidths[k] = maxW
		}

		// 2. Initial allocation based on content
		totalContentWidth := 0
		for _, w := range maxContentWidths {
			totalContentWidth += w + 2 // +2 for padding
		}

		columns := []table.Column{}
		remainingWidth := m.width

		for i, k := range propertyKeys {
			var w int
			if totalContentWidth > 0 {
				// Proportional allocation based on content, but ensure it sums to terminal width
				contentW := maxContentWidths[k] + 2
				if i == numCols-1 {
					w = remainingWidth // Take all remaining for last column
				} else {
					w = int(float64(contentW) / float64(totalContentWidth) * float64(m.width))
					if w < 5 {
						w = 5
					}
				}
			} else {
				w = m.width / numCols
			}

			columns = append(columns, table.Column{Title: k, Width: w})
			remainingWidth -= w
		}
		m.table.SetRows([]table.Row{}) // Clear rows before setting columns to avoid panic
		m.table.SetColumns(columns)
	}

	rows := []table.Row{}
	filter := strings.ToLower(m.filterInput.Value())
	m.pages = nil

	for _, page := range m.allPages {
		row := table.Row{}
		match := filter == ""
		for _, k := range propertyKeys {
			val := notion.PropertyToString(page.Properties[k])
			row = append(row, val)
			if !match && strings.Contains(strings.ToLower(val), filter) {
				match = true
			}
		}
		if match {
			rows = append(rows, row)
			m.pages = append(m.pages, page)
		}
	}
	m.table.SetRows(rows)
}

func (m RecordsModel) Update(msg tea.Msg) (RecordsModel, tea.Cmd) {
	var cmd tea.Cmd

	if m.filtering {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+n", "down":
				m.table, cmd = m.table.Update(tea.KeyMsg{Type: tea.KeyDown})
				return m, cmd
			case "ctrl+p", "up":
				m.table, cmd = m.table.Update(tea.KeyMsg{Type: tea.KeyUp})
				return m, cmd
			case "enter", "esc":
				m.filtering = false
				m.filterInput.Blur()
				return m, nil
			}
		}
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.updateTable()
		return m, cmd
	}

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
		case "b":
			idx := m.table.Cursor()
			if idx >= 0 && idx < len(m.pages) {
				_ = openBrowser(m.pages[idx].URL)
			}
		case "a":
			return m, func() tea.Msg {
				return CreateRecordMsg{DatabaseID: m.dbID}
			}
		case "e":
			idx := m.table.Cursor()
			if idx >= 0 && idx < len(m.pages) {
				page := m.pages[idx]
				return m, func() tea.Msg {
					return EditRecordMsg{Page: &page}
				}
			}
		case "/":
			m.filtering = true
			m.filterInput.Focus()
			return m, nil
		case "enter", "l":
			idx := m.table.Cursor()
			if idx >= 0 && idx < len(m.pages) {
				page := m.pages[idx]
				return m, func() tea.Msg {
					return SelectPageMsg{Page: &page}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(m.width)
		m.table.SetHeight(m.height - 4)
		if m.allPages != nil {
			m.updateTable()
		}

	case recordsMsg:
		m.loading = false
		m.allPages = msg
		m.updateTable()
		m.table.SetCursor(0)

	case errMsg:
		m.err = msg
		m.loading = false
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m RecordsModel) View() string {
	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Records View Error: %v", m.err))
	}

	var header string
	if m.filtering {
		header = m.filterInput.View() + "\n\n"
	} else {
		header = TitleStyle.Width(m.width).Padding(0, 1).Render("Database: "+m.dbName) + "\n\n"
		if m.loading {
			header += "Loading records...\n"
		}
	}

	footer := renderFooter(m.width, []keyHelp{
		{"b", "Open"},
		{"o", "Omnisearch"},
		{"a", "New"},
		{"r", "Refresh"},
		{"/", "Filter"},
		{"Enter", "Detail"},
		{"e", "Edit"},
		{"Esc", "Back"},
		{"q", "Quit"},
	})

	mainView := header + m.table.View()

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Height(m.height-1).Render(mainView),
		footer,
	)
}
