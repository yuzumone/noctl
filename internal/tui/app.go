package tui

import (
	"fmt"

	"noctl/internal/config"
	"noctl/internal/notion"

	tea "github.com/charmbracelet/bubbletea"
)

type sessionState uint

const (
	viewDBList sessionState = iota
	viewRecords
	viewEditor
)

// AppModel is the root model for the application.
type AppModel struct {
	state  sessionState
	config *config.Config
	client *notion.Client
	
	dbList  DBListModel
	records RecordsModel
	editor  EditorModel
	
	width  int
	height int
	err    error
}

// NewAppModel creates a new AppModel.
func NewAppModel(cfg *config.Config) AppModel {
	var client *notion.Client
	if cfg.NotionToken != "" {
		client = notion.NewClient(cfg.NotionToken)
	}

	return AppModel{
		state:   viewDBList,
		config:  cfg,
		client:  client,
		dbList:  NewDBListModel(client),
		records: NewRecordsModel(client),
		editor:  NewEditorModel(client),
	}
}

// Init initializes the application.
func (m AppModel) Init() tea.Cmd {
	return m.dbList.Init()
}

// Update handles messages.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "h":
			if m.state == viewRecords {
				m.state = viewDBList
				return m, nil
			} else if m.state == viewEditor {
				m.state = viewRecords
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.dbList, _ = m.dbList.Update(msg)
		m.records, _ = m.records.Update(msg)
		m.editor, _ = m.editor.Update(msg)

	case SelectDBMsg:
		m.state = viewRecords
		return m, m.records.SetDatabase(msg.ID, msg.Title)

	case SelectPageMsg:
		m.state = viewEditor
		return m, m.editor.SetPage(msg.Page)

	case CreateRecordMsg:
		m.state = viewEditor
		return m, m.editor.SetCreateMode(msg.DatabaseID)

	case EditRecordMsg:
		m.state = viewEditor
		return m, m.editor.SetEditMode(msg.Page)

	case PageCreatedMsg, PageUpdatedMsg:
		m.state = viewRecords
		return m, m.records.SetDatabase(m.records.dbID, m.records.dbName)
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
	}

	return m, tea.Batch(cmds...)
}

// View renders the application.
func (m AppModel) View() string {
	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Fatal Error: %v\n\nPress 'q' to quit.", m.err))
	}

	if m.config.NotionToken == "" {
		return TitleStyle.Render("noctl") + "\n\n" + ErrorStyle.Render("Error: NOTION_TOKEN is not set. Please set it in your environment or config file (~/.config/noctl/config.yaml).")
	}

	switch m.state {
	case viewDBList:
		return m.dbList.View()
	case viewRecords:
		return m.records.View()
	case viewEditor:
		return m.editor.View()
	default:
		return "Unknown state"
	}
}
