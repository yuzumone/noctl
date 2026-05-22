// Package tui provides terminal user interface components.
package tui

import (
	"fmt"

	"noctl/internal/config"
	"noctl/internal/notion"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"
)

type sessionState uint

const (
	viewDBList sessionState = iota
	viewRecords
	viewEditor
	viewOmnisearch
	viewSelector
)

// AppModel is the root model for the application.
type AppModel struct {
	state   sessionState
	history []sessionState
	config  *config.Config
	client  *notion.Client

	dbList     DBListModel
	records    RecordsModel
	editor     EditorModel
	omnisearch OmnisearchModel
	selector   SelectorModel

	width  int
	height int
	err    error
}

// NewAppModel creates a new AppModel.
func NewAppModel(cfg *config.Config) *AppModel {
	var client *notion.Client
	if cfg.NotionToken != "" {
		client = notion.NewClient(cfg.NotionToken)
	}

	return &AppModel{
		state:      viewDBList,
		history:    []sessionState{},
		config:     cfg,
		client:     client,
		dbList:     NewDBListModel(client),
		records:    NewRecordsModel(client),
		editor:     NewEditorModel(client),
		omnisearch: NewOmnisearchModel(client),
		selector:   NewSelectorModel(),
	}
}

func (m *AppModel) pushState(s sessionState) {
	if m.state == s {
		return
	}
	m.history = append(m.history, m.state)
	m.state = s
}

func (m *AppModel) popState() {
	if len(m.history) == 0 {
		return
	}
	last := len(m.history) - 1
	m.state = m.history[last]
	m.history = m.history[:last]
}

// Init initializes the application.
func (m *AppModel) Init() tea.Cmd {
	return m.dbList.Init()
}

// Update handles messages and updates the AppModel.
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "o":
			if m.state != viewOmnisearch {
				// Don't trigger if we are filtering in DB list or Records view
				if m.state == viewDBList && m.dbList.list.FilterState() == list.Filtering {
					break
				}
				if m.state == viewRecords && m.records.filtering {
					break
				}

				m.pushState(viewOmnisearch)
				m.omnisearch.input.SetValue("")
				m.omnisearch.list.SetItems(nil)
				m.omnisearch.input.Focus()
				return m, nil
			}
		case "esc", "h":
			// Don't trigger back on 'h' if we are in an input mode
			if msg.String() == "h" {
				if m.state == viewOmnisearch {
					break // Let it fall through to delegation
				}
				if m.state == viewEditor && (m.editor.mode == modeCreate || m.editor.mode == modeEdit) {
					break // Let it fall through to delegation
				}
			}

			// Handle 'back' action
			if msg.String() == "esc" || msg.String() == "h" {
				if len(m.history) > 0 {
					m.popState()
					return m, nil
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.dbList, _ = m.dbList.Update(msg)
		m.records, _ = m.records.Update(msg)
		m.editor, _ = m.editor.Update(msg)
		m.omnisearch, _ = m.omnisearch.Update(msg)
		m.selector, _ = m.selector.Update(msg)

	case SelectDBMsg:
		m.pushState(viewRecords)
		return m, m.records.SetDatabase(msg.ID, msg.Title)

	case SelectPageMsg:
		m.pushState(viewEditor)
		return m, m.editor.SetPage(msg.Page)

	case CreateRecordMsg:
		m.pushState(viewEditor)
		return m, m.editor.SetCreateMode(msg.DatabaseID)

	case EditRecordMsg:
		m.pushState(viewEditor)
		return m, m.editor.SetEditMode(msg.Page)

	case PageCreatedMsg, PageUpdatedMsg:
		m.state = viewRecords
		var page *notionapi.Page
		if p, ok := msg.(PageCreatedMsg); ok {
			page = (*notionapi.Page)(p)
		} else if p, ok := msg.(PageUpdatedMsg); ok {
			page = (*notionapi.Page)(p)
		}

		dbID := m.records.dbID
		dbName := m.records.dbName
		if dbID == "" && page != nil {
			dbID = string(page.Parent.DatabaseID)
		}
		return m, m.records.SetDatabase(dbID, dbName)

	case CancelEditMsg:
		if m.editor.page == nil {
			m.popState()
		} else {
			m.editor.mode = modeView
			m.editor.updateContent()
		}
		return m, nil

	case CancelOmnisearchMsg:
		m.popState()
		return m, nil

	case OpenSelectorMsg:
		m.selector.SetOptions(msg.PropName, msg.IsMulti, msg.Options, msg.CurrentValues)
		m.pushState(viewSelector)
		return m, nil

	case SelectorDoneMsg:
		m.popState()
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd

	case CancelSelectorMsg:
		m.popState()
		return m, nil
	}

	// Delegate to sub-models
	switch m.state {
	case viewDBList:
		m.dbList, cmd = m.dbList.Update(msg)
		cmds = append(cmds, cmd)
	case viewRecords:
		m.records, cmd = m.records.Update(msg)
		cmds = append(cmds, cmd)
	case viewEditor:
		m.editor, cmd = m.editor.Update(msg)
		cmds = append(cmds, cmd)
	case viewOmnisearch:
		m.omnisearch, cmd = m.omnisearch.Update(msg)
		cmds = append(cmds, cmd)
	case viewSelector:
		m.selector, cmd = m.selector.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the application.
func (m *AppModel) View() string {
	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Fatal Error: %v\n\nPress 'q' to quit.", m.err))
	}

	if m.config.NotionToken == "" {
		return TitleStyle.Render("noctl") + "\n\n" + ErrorStyle.Render("Error: NOTION_TOKEN is not set. Please set it in your environment or config file (~/.config/noctl/config.yaml).")
	}

	var baseView string
	switch m.state {
	case viewDBList:
		baseView = m.dbList.View()
	case viewRecords:
		baseView = m.records.View()
	case viewEditor:
		baseView = m.editor.View()
	case viewOmnisearch, viewSelector:
		// If search or selector is active, show the previous state in the background
		if len(m.history) > 0 {
			switch m.history[len(m.history)-1] {
			case viewDBList:
				baseView = m.dbList.View()
			case viewRecords:
				baseView = m.records.View()
			case viewEditor:
				baseView = m.editor.View()
			default:
				baseView = m.dbList.View()
			}
		} else {
			baseView = m.dbList.View()
		}
	default:
		return "Unknown state"
	}

	if m.state == viewOmnisearch {
		popup := m.omnisearch.View()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, popup)
	}

	if m.state == viewSelector {
		popup := m.selector.View()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, popup)
	}

	return baseView
}
