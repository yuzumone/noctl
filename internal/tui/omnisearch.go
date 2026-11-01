// Package tui provides terminal user interface components.
package tui

import (
	"context"
	"fmt"

	"noctl/internal/notion"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	setListTheme(&l, true, true, true)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

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

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "ctrl+n":
			msg = tea.KeyPressMsg{Code: tea.KeyDown}
		case "ctrl+p":
			msg = tea.KeyPressMsg{Code: tea.KeyUp}
		}
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
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
		case "ctrl+o":
			if i, ok := m.list.SelectedItem().(searchItem); ok {
				_ = openBrowser(i.url)
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
		{"Ctrl+o", "Browser"},
		{"Esc", "Cancel"},
	})

	searchBar := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color(ActiveTheme.Border)).
		Width(popupWidth-4).
		Padding(0, 1).
		Render(m.input.View())

	content := lipgloss.JoinVertical(lipgloss.Left,
		searchBar,
		m.list.View(),
		footer,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ActiveTheme.Border)).
		Width(popupWidth - 2).
		Height(popupHeight - 2).
		Render(content)
}
