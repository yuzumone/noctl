// Package tui provides terminal user interface components.
package tui

import (
	"context"
	"fmt"

	"noctl/internal/notion"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"
)

type searchItem struct {
	id     string
	title  string
	url    string
	object string // "page" or "database"
}

func (i searchItem) Title() string {
	icon := "󰈔"
	if i.object == "database" {
		icon = "󰆼"
	}
	return icon + " " + i.title
}
func (i searchItem) Description() string { return i.object + ": " + i.id }
func (i searchItem) FilterValue() string { return i.title }

// OmnisearchModel is a Bubble Tea model for global search.
type OmnisearchModel struct {
	input   textinput.Model
	list    list.Model
	client  *notion.Client
	loading bool
	err     error
	width   int
	height  int
}

type searchResultsMsg *notionapi.SearchResponse

// CancelOmnisearchMsg is a message sent when the search is canceled.
type CancelOmnisearchMsg struct{}

// NewOmnisearchModel creates a new OmnisearchModel.
func NewOmnisearchModel(client *notion.Client) OmnisearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search pages and databases..."
	ti.Focus()

	delegate := list.NewDefaultDelegate()
	// Customize delegate to be more minimal by unsetting backgrounds
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("229")).
		UnsetBackground().
		BorderLeftForeground(lipgloss.Color("57")).
		Bold(false)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("240")).
		UnsetBackground().
		BorderLeftForeground(lipgloss.Color("57"))
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(lipgloss.Color("252"))
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.
		Foreground(lipgloss.Color("245"))

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.NoItems = l.Styles.NoItems.Padding(1, 2)

	return OmnisearchModel{
		input:  ti,
		list:   l,
		client: client,
	}
}

// Init initializes the OmnisearchModel.
func (m OmnisearchModel) Init() tea.Cmd {
	return nil
}

func (m OmnisearchModel) performSearch() tea.Msg {
	res, err := m.client.GlobalSearch(context.Background(), m.input.Value(), "")
	if err != nil {
		return errMsg(err)
	}
	return searchResultsMsg(res)
}

// Update updates the OmnisearchModel.
func (m OmnisearchModel) Update(msg tea.Msg) (OmnisearchModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

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
			return m, func() tea.Msg { return CancelOmnisearchMsg{} }
		case "enter":
			if i, ok := m.list.SelectedItem().(searchItem); ok {
				if i.object == "database" {
					return m, func() tea.Msg {
						return SelectDBMsg{ID: i.id, Title: i.title}
					}
				}
				// It's a page, but search result doesn't give us the full page object.
				// We need to fetch it.
				return m, func() tea.Msg {
					p, err := m.client.Page.Get(context.Background(), notionapi.PageID(i.id))
					if err != nil {
						return errMsg(err)
					}
					return SelectPageMsg{Page: p}
				}
			}
		case "up", "down":
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		popupWidth := int(float64(m.width) * 0.8)
		if popupWidth < 40 {
			popupWidth = m.width
		}
		popupHeight := int(float64(m.height) * 0.6)
		if popupHeight < 10 {
			popupHeight = m.height
		}
		m.list.SetSize(popupWidth-2, popupHeight-6)

	case searchResultsMsg:
		m.loading = false
		items := make([]list.Item, len(msg.Results))
		for i, res := range msg.Results {
			title := "Untitled"
			object := res.GetObject().String()
			id := ""
			url := ""

			switch b := res.(type) {
			case *notionapi.Page:
				id = string(b.ID)
				url = b.URL
				title = notion.PropertyToString(b.Properties["title"])
				if title == "" {
					for _, p := range b.Properties {
						if _, ok := p.(*notionapi.TitleProperty); ok {
							title = notion.PropertyToString(p)
							break
						}
					}
				}
			case *notionapi.Database:
				id = string(b.ID)
				url = b.URL
				if len(b.Title) > 0 {
					title = b.Title[0].PlainText
				}
			}
			items[i] = searchItem{
				id:     id,
				title:  title,
				url:    url,
				object: object,
			}
		}
		m.list.SetItems(items)
		return m, nil

	case errMsg:
		m.err = msg
		m.loading = false
		return m, nil
	}

	oldVal := m.input.Value()
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	if m.input.Value() != oldVal {
		m.loading = true
		cmds = append(cmds, m.performSearch)
	}

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the OmnisearchModel as a popup.
func (m OmnisearchModel) View() string {
	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Search Error: %v", m.err))
	}

	popupWidth := int(float64(m.width) * 0.8)
	if popupWidth < 40 {
		popupWidth = m.width
	}
	popupHeight := int(float64(m.height) * 0.6)
	if popupHeight < 10 {
		popupHeight = m.height
	}

	footer := renderFooter(popupWidth-2, []keyHelp{
		{"Enter", "Select"},
		{"Esc", "Cancel"},
	})

	searchBar := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("57")).
		Width(popupWidth - 4).
		Padding(0, 1).
		Render(m.input.View())

	content := lipgloss.JoinVertical(lipgloss.Left,
		searchBar,
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
