// Package tui provides terminal user interface components.
package tui

import (
	"context"
	"fmt"

	"noctl/internal/notion"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"
)

type dbItem struct {
	id    string
	title string
	url   string
}

func (i dbItem) Title() string       { return i.title }
func (i dbItem) Description() string { return i.id }
func (i dbItem) FilterValue() string { return i.title }

type DBListModel struct {
	list    list.Model
	client  *notion.Client
	loading bool
	err     error
	width   int
	height  int
}

type databasesMsg []notionapi.Database
type errMsg error
type SelectDBMsg struct {
	ID    string
	Title string
}

func NewDBListModel(client *notion.Client) DBListModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select a Database"
	l.Styles.Title = TitleStyle
	l.SetShowHelp(false)

	return DBListModel{
		list:   l,
		client: client,
	}
}

func (m DBListModel) Init() tea.Cmd {
	return m.fetchDatabases
}

func (m DBListModel) fetchDatabases() tea.Msg {
	if m.client == nil {
		return errMsg(fmt.Errorf("notion client is not initialized"))
	}
	dbs, err := m.client.ListDatabases(context.Background())
	if err != nil {
		return errMsg(fmt.Errorf("failed to fetch databases: %w", err))
	}
	return databasesMsg(dbs)
}

func (m DBListModel) Update(msg tea.Msg) (DBListModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "b":
			if i, ok := m.list.SelectedItem().(dbItem); ok {
				_ = openBrowser(i.url)
			}
		case "enter", "l":
			if i, ok := m.list.SelectedItem().(dbItem); ok {
				return m, func() tea.Msg {
					return SelectDBMsg{ID: i.id, Title: i.title}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-1)
		m.list.Styles.Title = TitleStyle.Width(msg.Width).Padding(0, 1)

	case databasesMsg:
		m.loading = false
		items := make([]list.Item, len(msg))
		for i, db := range msg {
			title := "Untitled"
			if len(db.Title) > 0 {
				title = db.Title[0].PlainText
			}
			items[i] = dbItem{id: string(db.ID), title: title, url: db.URL}
		}
		return m, m.list.SetItems(items)

	case errMsg:
		m.err = msg
		m.loading = false
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m DBListModel) View() string {
	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Database List Error: %v", m.err))
	}

	footer := renderFooter(m.width, []keyHelp{
		{"Enter", "Select"},
		{"o", "Omnisearch"},
		{"b", "Open"},
		{"/", "Filter"},
		{"q", "Quit"},
	})

	return lipgloss.JoinVertical(lipgloss.Left,
		m.list.View(),
		footer,
	)
}
