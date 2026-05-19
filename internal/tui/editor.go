package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"noctl/internal/notion"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"
)

type editorMode uint

const (
	modeView editorMode = iota
	modeCreate
	modeEdit
)

type EditorModel struct {
	viewport    viewport.Model
	inputs      []textinput.Model
	propKeys    []string
	propConfigs map[string]notionapi.PropertyConfig
	mode        editorMode
	client      *notion.Client
	dbID        string
	page         *notionapi.Page
	blocks       []notionapi.Block
	blocksLoaded bool
	focusedIdx   int
	tempFile     string
	err          error
	ready       bool
	loading     bool
	width       int
	height      int
}

func NewEditorModel(client *notion.Client) EditorModel {
	return EditorModel{
		client: client,
	}
}

func (m EditorModel) Init() tea.Cmd {
	return nil
}

type PageCreatedMsg *notionapi.Page
type PageUpdatedMsg *notionapi.Page
type blocksMsg []notionapi.Block
type editorFinishedMsg struct{ err error }

func (m *EditorModel) SetPage(page *notionapi.Page) tea.Cmd {
	m.mode = modeView
	m.page = page
	m.blocks = nil
	m.blocksLoaded = false
	m.updateContent()
	return m.fetchBlocks
}

func (m EditorModel) fetchBlocks() tea.Msg {
	if m.page == nil {
		return nil
	}
	blocks, err := m.client.ListBlocksExpanded(context.Background(), string(m.page.ID))
	if err != nil {
		return errMsg(fmt.Errorf("failed to fetch page content: %w", err))
	}
	return blocksMsg(blocks)
}

func (m *EditorModel) SetCreateMode(dbID string) tea.Cmd {
	m.mode = modeCreate
	m.dbID = dbID
	m.loading = true
	m.inputs = nil
	m.propKeys = nil
	m.focusedIdx = 0

	return func() tea.Msg {
		db, err := m.client.GetDatabase(context.Background(), dbID)
		if err != nil {
			return errMsg(err)
		}
		return db
	}
}

func (m *EditorModel) SetEditMode(page *notionapi.Page) tea.Cmd {
	m.mode = modeEdit
	m.page = page
	m.dbID = string(page.Parent.DatabaseID)
	m.loading = true
	m.inputs = nil
	m.propKeys = nil
	m.focusedIdx = 0

	return func() tea.Msg {
		db, err := m.client.GetDatabase(context.Background(), m.dbID)
		if err != nil {
			return errMsg(err)
		}
		return db
	}
}

func (m *EditorModel) initCreateInputs(db *notionapi.Database) {
	m.loading = false
	m.propKeys = []string{}
	m.propConfigs = db.Properties

	// Find title property first
	titleKey := ""
	for k, p := range db.Properties {
		if p.GetType() == notionapi.PropertyConfigTypeTitle {
			titleKey = k
			break
		}
	}

	if titleKey != "" {
		m.propKeys = append(m.propKeys, titleKey)
	}

	for k, p := range db.Properties {
		if k == titleKey {
			continue
		}
		// Support simple types for now
		switch p.GetType() {
		case notionapi.PropertyConfigTypeRichText,
			notionapi.PropertyConfigTypeNumber,
			notionapi.PropertyConfigTypeURL,
			notionapi.PropertyConfigTypeEmail,
			notionapi.PropertyConfigTypePhoneNumber:
			m.propKeys = append(m.propKeys, k)
		}
	}

	for _, k := range m.propKeys {
		ti := textinput.New()
		ti.Placeholder = k
		m.inputs = append(m.inputs, ti)
	}

	if len(m.inputs) > 0 {
		m.inputs[0].Focus()
	}
}

func (m *EditorModel) initEditInputs(db *notionapi.Database) {
	m.loading = false
	m.propKeys = []string{}
	m.propConfigs = db.Properties

	// Find title property first
	titleKey := ""
	for k, p := range db.Properties {
		if p.GetType() == notionapi.PropertyConfigTypeTitle {
			titleKey = k
			break
		}
	}

	if titleKey != "" {
		m.propKeys = append(m.propKeys, titleKey)
	}

	for k, p := range db.Properties {
		if k == titleKey {
			continue
		}
		// Support simple types
		switch p.GetType() {
		case notionapi.PropertyConfigTypeRichText,
			notionapi.PropertyConfigTypeNumber,
			notionapi.PropertyConfigTypeURL,
			notionapi.PropertyConfigTypeEmail,
			notionapi.PropertyConfigTypePhoneNumber:
			m.propKeys = append(m.propKeys, k)
		}
	}

	for _, k := range m.propKeys {
		ti := textinput.New()
		ti.Placeholder = k
		// Set current value
		if m.page != nil {
			if prop, ok := m.page.Properties[k]; ok {
				ti.SetValue(notion.PropertyToString(prop))
			}
		}
		m.inputs = append(m.inputs, ti)
	}

	if len(m.inputs) > 0 {
		m.inputs[0].Focus()
	}
}

func (m EditorModel) createPage() tea.Cmd {
	return func() tea.Msg {
		props := notionapi.Properties{}
		for i, k := range m.propKeys {
			val := m.inputs[i].Value()
			if val == "" {
				continue
			}

			config := m.propConfigs[k]
			switch config.GetType() {
			case notionapi.PropertyConfigTypeTitle:
				props[k] = notionapi.TitleProperty{
					Title: []notionapi.RichText{{Text: &notionapi.Text{Content: val}}},
				}
			case notionapi.PropertyConfigTypeRichText:
				props[k] = notionapi.RichTextProperty{
					RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: val}}},
				}
			case notionapi.PropertyConfigTypeNumber:
				num, err := strconv.ParseFloat(val, 64)
				if err == nil {
					props[k] = notionapi.NumberProperty{Number: num}
				}
			case notionapi.PropertyConfigTypeURL:
				props[k] = notionapi.URLProperty{URL: val}
			case notionapi.PropertyConfigTypeEmail:
				props[k] = notionapi.EmailProperty{Email: val}
			case notionapi.PropertyConfigTypePhoneNumber:
				props[k] = notionapi.PhoneNumberProperty{PhoneNumber: val}
			}
		}

		page, err := m.client.CreatePage(context.Background(), m.dbID, props)
		if err != nil {
			return errMsg(fmt.Errorf("failed to create page: %w", err))
		}
		return PageCreatedMsg(page)
	}
}

func (m EditorModel) updatePage() tea.Cmd {
	return func() tea.Msg {
		props := notionapi.Properties{}
		for i, k := range m.propKeys {
			val := m.inputs[i].Value()

			config := m.propConfigs[k]
			switch config.GetType() {
			case notionapi.PropertyConfigTypeTitle:
				props[k] = notionapi.TitleProperty{
					Title: []notionapi.RichText{{Text: &notionapi.Text{Content: val}}},
				}
			case notionapi.PropertyConfigTypeRichText:
				props[k] = notionapi.RichTextProperty{
					RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: val}}},
				}
			case notionapi.PropertyConfigTypeNumber:
				num, err := strconv.ParseFloat(val, 64)
				if err == nil {
					props[k] = notionapi.NumberProperty{Number: num}
				}
			case notionapi.PropertyConfigTypeURL:
				props[k] = notionapi.URLProperty{URL: val}
			case notionapi.PropertyConfigTypeEmail:
				props[k] = notionapi.EmailProperty{Email: val}
			case notionapi.PropertyConfigTypePhoneNumber:
				props[k] = notionapi.PhoneNumberProperty{PhoneNumber: val}
			}
		}

		page, err := m.client.UpdatePage(context.Background(), string(m.page.ID), props)
		if err != nil {
			return errMsg(fmt.Errorf("failed to update page: %w", err))
		}
		return PageUpdatedMsg(page)
	}
}

func (m *EditorModel) updateContent() {
	if m.page == nil {
		return
	}

	var content strings.Builder

	// Page Title
	titleKey := ""
	for k, p := range m.page.Properties {
		if _, ok := p.(*notionapi.TitleProperty); ok {
			titleKey = k
			break
		}
	}

	content.WriteString(TitleStyle.Width(m.width).Padding(0, 1).Render("Page: "+notion.PropertyToString(m.page.Properties[titleKey])) + "\n\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("ID: "+string(m.page.ID)) + "\n\n")

	// Properties
	content.WriteString(lipgloss.NewStyle().Bold(true).Underline(true).Render("Properties:") + "\n")

	var propNames []string
	for name := range m.page.Properties {
		if name != titleKey {
			propNames = append(propNames, name)
		}
	}
	sort.Strings(propNames)

	for _, name := range propNames {
		prop := m.page.Properties[name]
		label := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true).Render(name + ": ")
		value := notion.PropertyToString(prop)
		if value == "" {
			value = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("(empty)")
		}
		content.WriteString(fmt.Sprintf("%s %s\n", label, value))
	}

	// Content Blocks
	if len(m.blocks) > 0 {
		md := notion.BlocksToMarkdown(m.blocks)
		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(m.width-4),
		)
		if err == nil {
			rendered, err := renderer.Render(md)
			if err == nil {
				content.WriteString("\n" + rendered)
			} else {
				content.WriteString("\n" + md)
			}
		} else {
			content.WriteString("\n" + md)
		}
	} else if m.mode == modeView {
		if m.blocksLoaded {
			content.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("(empty content)") + "\n")
		} else {
			content.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Loading content...") + "\n")
		}
	}

	m.viewport.SetContent(content.String())
}

func (m *EditorModel) openInEditor() tea.Cmd {
	if m.page == nil {
		return nil
	}

	md := notion.BlocksToMarkdownExtended(m.blocks, true)
	
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "noctl-*.md")
	if err != nil {
		return func() tea.Msg { return errMsg(fmt.Errorf("failed to create temp file: %w", err)) }
	}
	m.tempFile = tmpFile.Name()

	if _, err := tmpFile.WriteString(md); err != nil {
		tmpFile.Close()
		os.Remove(m.tempFile)
		return func() tea.Msg { return errMsg(fmt.Errorf("failed to write to temp file: %w", err)) }
	}
	tmpFile.Close()

	// Open editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim" // Default to vim
	}

	cmd := exec.Command(editor, m.tempFile)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

func (m *EditorModel) handleEditorFinished(msg editorFinishedMsg) tea.Cmd {
	defer os.Remove(m.tempFile)

	if msg.err != nil {
		return func() tea.Msg { return errMsg(fmt.Errorf("editor failed: %w", msg.err)) }
	}

	// Read back content
	newMdBytes, err := os.ReadFile(m.tempFile)
	if err != nil {
		return func() tea.Msg { return errMsg(fmt.Errorf("failed to read temp file: %w", err)) }
	}
	newMd := string(newMdBytes)

	oldMd := notion.BlocksToMarkdown(m.blocks)
	if newMd == oldMd {
		return nil // No changes
	}

	m.loading = true
	m.updateContent()

	return func() tea.Msg {
		err := m.client.UpdatePageContent(context.Background(), string(m.page.ID), newMd)
		if err != nil {
			return errMsg(fmt.Errorf("failed to update page content: %w", err))
		}
		// Refresh blocks
		return m.fetchBlocks()
	}
}

func (m EditorModel) Update(msg tea.Msg) (EditorModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if m.mode == modeCreate || m.mode == modeEdit {
		switch msg := msg.(type) {
		case *notionapi.Database:
			if m.mode == modeCreate {
				m.initCreateInputs(msg)
			} else {
				m.initEditInputs(msg)
			}
			return m, nil

		case PageCreatedMsg:
			m.loading = false
			return m, func() tea.Msg { return msg }

		case PageUpdatedMsg:
			m.loading = false
			return m, func() tea.Msg { return msg }

		case tea.KeyMsg:
			switch msg.String() {
			case "up":
				m.inputs[m.focusedIdx].Blur()
				m.focusedIdx--
				if m.focusedIdx < 0 {
					m.focusedIdx = len(m.inputs) - 1
				}
				m.inputs[m.focusedIdx].Focus()
			case "down", "tab", "enter":
				if msg.String() == "enter" && m.focusedIdx == len(m.inputs)-1 {
					m.loading = true
					if m.mode == modeCreate {
						return m, m.createPage()
					} else {
						return m, m.updatePage()
					}
				}
				m.inputs[m.focusedIdx].Blur()
				m.focusedIdx++
				if m.focusedIdx >= len(m.inputs) {
					m.focusedIdx = 0
				}
				m.inputs[m.focusedIdx].Focus()
			case "ctrl+s":
				m.loading = true
				if m.mode == modeCreate {
					return m, m.createPage()
				} else {
					return m, m.updatePage()
				}
			}
		}

		for i := range m.inputs {
			m.inputs[i], cmd = m.inputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case editorFinishedMsg:
		return m, m.handleEditorFinished(msg)

	case errMsg:
		m.err = msg
		m.loading = false
		return m, nil

	case blocksMsg:
		m.blocks = msg
		m.blocksLoaded = true
		m.loading = false
		m.updateContent()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "o":
			if m.page != nil {
				_ = openBrowser(m.page.URL)
			}
		case "e":
			return m, m.SetEditMode(m.page)
		case "O":
			return m, m.openInEditor()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-1)
			m.viewport.HighPerformanceRendering = false
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 1
		}
		m.updateContent()
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m EditorModel) View() string {
	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Editor Error: %v", m.err))
	}

	if m.loading {
		return DocStyle.Render("Loading...")
	}

	if m.mode == modeCreate || m.mode == modeEdit {
		var s strings.Builder
		title := "New Record"
		if m.mode == modeEdit {
			title = "Edit Record"
		}
		s.WriteString(TitleStyle.Width(m.width).Padding(0, 1).Render(title) + "\n\n")
		for i := range m.inputs {
			s.WriteString(m.inputs[i].View() + "\n")
		}

		footer := renderFooter(m.width, []keyHelp{
			{"Tab", "Next"},
			{"Ctrl+S", "Save"},
			{"Esc", "Cancel"},
		})

		return lipgloss.JoinVertical(lipgloss.Left,
			DocStyle.Render(s.String()),
			footer,
		)
	}

	if !m.ready || m.page == nil {
		return "Initializing detail view..."
	}

	footer := renderFooter(m.width, []keyHelp{
		{"o", "Open"},
		{"e", "Edit Prop"},
		{"O", "Edit Content"},
		{"Esc", "Back"},
		{"q", "Quit"},
	})

	return lipgloss.JoinVertical(lipgloss.Left,
		m.viewport.View(),
		footer,
	)
}
