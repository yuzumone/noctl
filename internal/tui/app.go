// Package tui provides terminal user interface components.
package tui

import (
	"fmt"
	"strings"

	"noctl/internal/config"
	"noctl/internal/notion"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"
)

type SessionState uint

const (
	ViewDBList SessionState = iota
	ViewRecords
	ViewEditor
	ViewOmnisearch
	ViewSelector
	ViewConfirm
	ViewCalendar
	ViewCalendarDetails
)

// AppModel is the root model for the application.
type AppModel struct {
	state   SessionState
	history []SessionState
	config  *config.Config
	client  *notion.Client

	dbList        DBListModel
	records       RecordsModel
	calendar      CalendarModel
	editor        EditorModel
	omnisearch    OmnisearchModel
	selector      SelectorModel
	tableSelector TableSelectorModel
	confirm       ConfirmModel

	width  int
	height int
	err    error
}

// NewAppModel creates a new AppModel.
func NewAppModel(cfg *config.Config) *AppModel {
	InitStyles(cfg)

	var client *notion.Client
	if cfg.NotionToken != "" {
		client = notion.NewClient(cfg.NotionToken)
	}

	return &AppModel{
		state:         ViewDBList,
		history:       []SessionState{},
		config:        cfg,
		client:        client,
		dbList:        NewDBListModel(client),
		records:       NewRecordsModel(client),
		calendar:      NewCalendarModel(client),
		editor:        NewEditorModel(client),
		omnisearch:    NewOmnisearchModel(client),
		selector:      NewSelectorModel(),
		tableSelector: NewTableSelectorModel(),
		confirm:       NewConfirmModel("Quit noctl?"),
	}
}

// SetInitialState sets the initial state of the application.
func (m *AppModel) SetInitialState(s SessionState) {
	m.state = s
}

func (m *AppModel) pushState(s SessionState) {
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

// isInInputMode returns true when the user is actively typing in an input field.
func (m *AppModel) isInInputMode() bool {
	if m.state == ViewOmnisearch {
		return true
	}
	if m.state == ViewEditor && (m.editor.mode == modeCreate || m.editor.mode == modeEdit) {
		return true
	}
	if m.state == ViewDBList && m.dbList.list.FilterState() == list.Filtering {
		return true
	}
	if m.state == ViewRecords && m.records.filtering {
		return true
	}
	return false
}

// Init initializes the application.
func (m *AppModel) Init() tea.Cmd {
	if m.state == ViewCalendar {
		return m.calendar.SetDatabases(m.config.CalendarDatabaseIDs)
	}
	return m.dbList.Init()
}

// Update handles messages and updates the AppModel.
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			// Don't intercept q when typing
			if m.isInInputMode() {
				break
			}
			// Don't stack confirm on top of confirm
			if m.state == ViewConfirm {
				break
			}
			m.confirm = NewConfirmModel("Quit noctl?")
			m.confirm.width = m.width
			m.confirm.height = m.height
			m.pushState(ViewConfirm)
			return m, nil
		case "o":
			if m.state != ViewOmnisearch {
				// Don't trigger if we are filtering in DB list or Records view
				if m.state == ViewDBList && m.dbList.list.FilterState() == list.Filtering {
					break
				}
				if m.state == ViewRecords && m.records.filtering {
					break
				}

				m.pushState(ViewOmnisearch)
				m.omnisearch.input.SetValue("")
				m.omnisearch.list.SetItems(nil)
				m.omnisearch.input.Focus()
				return m, nil
			}
		case "esc", "h":
			// Don't trigger back on 'h' if we are in an input mode or specific views that use 'h'
			if msg.String() == "h" {
				if m.state == ViewOmnisearch || m.state == ViewConfirm || m.state == ViewCalendarDetails {
					break // Let it fall through to delegation
				}
				if m.state == ViewEditor && (m.editor.mode == modeCreate || m.editor.mode == modeEdit) {
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

	case ConfirmYesMsg:
		return m, tea.Quit

	case ConfirmNoMsg:
		m.popState()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.dbList, _ = m.dbList.Update(msg)
		m.records, _ = m.records.Update(msg)
		m.calendar, _ = m.calendar.Update(msg)
		m.editor, _ = m.editor.Update(msg)
		m.omnisearch, _ = m.omnisearch.Update(msg)
		m.selector, _ = m.selector.Update(msg)
		m.tableSelector, _ = m.tableSelector.Update(msg)
		m.confirm, _ = m.confirm.Update(msg)

	case SelectDBMsg:
		m.pushState(ViewRecords)
		return m, m.records.SetDatabase(msg.ID, msg.Title)

	case SelectPageMsg:
		m.pushState(ViewEditor)
		return m, m.editor.SetPage(msg.Page)

	case CreateRecordMsg:
		m.pushState(ViewEditor)
		return m, m.editor.SetCreateMode(msg.DatabaseID)

	case EditRecordMsg:
		m.pushState(ViewEditor)
		return m, m.editor.SetEditMode(msg.Page)

	case PageCreatedMsg, PageUpdatedMsg:
		m.state = ViewRecords
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

	case OpenCalendarDetailsMsg:
		m.tableSelector.SetPages(fmt.Sprintf("Records on %s", msg.Date.Format("2006-01-02")), msg.Pages)
		m.pushState(ViewCalendarDetails)
		return m, nil

	case OpenSelectorMsg:
		m.selector.SetOptions(msg.PropName, msg.IsMulti, msg.Options, msg.CurrentValues)
		m.pushState(ViewSelector)
		return m, nil

	case SelectorDoneMsg:
		if m.state == ViewCalendarDetails {
			m.popState()
			selectedIdx := msg.SelectedIndex
			if selectedIdx >= 0 {
				pagesByDay := m.calendar.getPagesByDay(m.calendar.selectedDate)
				pages := pagesByDay[m.calendar.selectedDate.Day()]
				if selectedIdx < len(pages) {
					page := pages[selectedIdx]
					m.pushState(ViewEditor)
					return m, m.editor.SetPage(&page)
				}
			}
			return m, nil
		}
		m.popState()
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd

	case CancelSelectorMsg:
		m.popState()
		return m, nil
	}

	// Delegate to sub-models
	switch m.state {
	case ViewDBList:
		m.dbList, cmd = m.dbList.Update(msg)
		cmds = append(cmds, cmd)
	case ViewRecords:
		m.records, cmd = m.records.Update(msg)
		cmds = append(cmds, cmd)
	case ViewCalendar:
		m.calendar, cmd = m.calendar.Update(msg)
		cmds = append(cmds, cmd)
	case ViewCalendarDetails:
		m.tableSelector, cmd = m.tableSelector.Update(msg)
		cmds = append(cmds, cmd)
	case ViewEditor:
		m.editor, cmd = m.editor.Update(msg)
		cmds = append(cmds, cmd)
	case ViewOmnisearch:
		m.omnisearch, cmd = m.omnisearch.Update(msg)
		cmds = append(cmds, cmd)
	case ViewSelector:
		m.selector, cmd = m.selector.Update(msg)
		cmds = append(cmds, cmd)
	case ViewConfirm:
		m.confirm, cmd = m.confirm.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// backgroundView returns the underlying screen view for use behind popups.
func (m *AppModel) backgroundView() string {
	if len(m.history) > 0 {
		// Find the first non-popup state in history
		for i := len(m.history) - 1; i >= 0; i-- {
			switch m.history[i] {
			case ViewDBList:
				return m.dbList.View()
			case ViewRecords:
				return m.records.View()
			case ViewCalendar:
				return m.calendar.View()
			case ViewEditor:
				return m.editor.View()
			}
		}
	}
	return m.dbList.View()
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
	case ViewDBList:
		baseView = m.dbList.View()
	case ViewRecords:
		baseView = m.records.View()
	case ViewCalendar:
		baseView = m.calendar.View()
	case ViewEditor:
		baseView = m.editor.View()
	case ViewOmnisearch, ViewSelector, ViewConfirm, ViewCalendarDetails:
		baseView = m.backgroundView()
	default:
		return "Unknown state"
	}

	var popup string
	switch m.state {
	case ViewOmnisearch:
		popup = m.omnisearch.View()
	case ViewSelector:
		popup = m.selector.View()
	case ViewCalendarDetails:
		popup = m.tableSelector.View()
	case ViewConfirm:
		popup = m.confirm.View()
	}

	if popup != "" {
		// Overlay the popup on the base view
		return m.renderWithPopup(baseView, popup)
	}

	return baseView
}

// renderWithPopup overlays a popup on top of a background string.
func (m *AppModel) renderWithPopup(background, popup string) string {
	bgLines := strings.Split(background, "\n")

	// Create centered popup string of the same size as background
	popupCentered := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, popup)
	pcLines := strings.Split(popupCentered, "\n")

	finalLines := make([]string, len(bgLines))
	for i := 0; i < len(bgLines); i++ {
		// Use the background line as base
		bgLine := bgLines[i]
		if i >= len(pcLines) {
			finalLines[i] = bgLine
			continue
		}

		pcLine := pcLines[i]

		// If pcLine is just spaces, it's transparency, so use bgLine
		// We use Width to handle ANSI codes correctly
		if strings.TrimSpace(pcLine) == "" {
			finalLines[i] = bgLine
		} else {
			// This is the line with the popup content (including borders)
			finalLines[i] = pcLine
		}
	}

	return strings.Join(finalLines, "\n")
}
