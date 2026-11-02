package tui

import (
	"context"
	"fmt"
	"image/color"
	"io"
	"strings"
	"testing"
	"time"

	"noctl/internal/config"
	"noctl/internal/notion"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/jomei/notionapi"
)

func TestAppFormTypingDoesNotTriggerShortcuts(t *testing.T) {
	for _, mode := range []editorMode{modeCreate, modeEdit} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			m := NewAppModel(&config.Config{NotionToken: "test"})
			m.pushState(ViewEditor)
			m.editor.mode = mode
			m.editor.initPropertyInputs(&notionapi.Database{Properties: notionapi.PropertyConfigs{
				"Title": &notionapi.TitlePropertyConfig{Type: notionapi.PropertyConfigTypeTitle},
			}}, nil)
			for _, r := range "oqhr" {
				_, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
			}
			if m.state != ViewEditor || m.editor.inputs[0].Value() != "oqhr" || len(m.history) != 1 {
				t.Fatalf("typing triggered a shortcut: state=%v input=%q", m.state, m.editor.inputs[0].Value())
			}
		})
	}
}

func TestAppRecordsEscapeExitsFilterBeforeGoingBack(t *testing.T) {
	m := NewAppModel(&config.Config{NotionToken: "test"})
	m.pushState(ViewRecords)
	_, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	_, _ = m.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.state != ViewRecords || m.records.filtering || m.records.filterInput.Value() != "o" || len(m.history) != 1 {
		t.Fatal("Escape navigated away instead of finishing record filter")
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.state != ViewDBList {
		t.Fatal("second Escape did not navigate back")
	}
}

func TestAppSelectorFilterOwnsShortcutsAndEscape(t *testing.T) {
	m := NewAppModel(&config.Config{NotionToken: "test"})
	m.pushState(ViewEditor)
	m.selector.SetOptions("Tags", false, []SelectorOption{{Name: "oqhr"}}, nil)
	m.pushState(ViewSelector)
	_, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "oqhr" {
		_, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if m.state != ViewSelector || m.selector.list.FilterInput.Value() != "oqhr" {
		t.Fatal("selector filter typing triggered a global shortcut")
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.state != ViewSelector || m.selector.list.FilterState() == list.Filtering {
		t.Fatal("Escape closed selector instead of exiting its filter")
	}
	m.selector.list.SetFilterText("oqhr")
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.state != ViewSelector || m.selector.list.FilterState() != list.Unfiltered {
		t.Fatal("Escape closed selector instead of clearing applied filter")
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("Escape did not cancel selector")
	}
	_, _ = m.Update(cmd())
	if m.state != ViewEditor {
		t.Fatal("selector cancel did not return to editor")
	}
}

func TestAppDatabaseFilterOwnsKeysAndEscape(t *testing.T) {
	m := NewAppModel(&config.Config{NotionToken: "test"})
	m.state = ViewCalendar
	m.pushState(ViewDBList)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	_, _ = m.Update(databasesMsg{{ID: "database", Title: []notionapi.RichText{{PlainText: "Tasks"}}}})
	_, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "loqhr" {
		_, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if m.state != ViewDBList || m.dbList.list.FilterInput.Value() != "loqhr" {
		t.Fatal("database filter typing triggered navigation or a shortcut")
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.state != ViewDBList || m.dbList.list.FilterState() != list.Unfiltered || len(m.history) != 1 {
		t.Fatal("Escape navigated away instead of canceling database filter")
	}
	m.dbList.list.SetFilterText("Tasks")
	m.dbList.list.SetFilterState(list.Filtering)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.state != ViewDBList || m.dbList.list.FilterState() != list.FilterApplied {
		t.Fatal("Enter did not apply database filter")
	}
	if cmd != nil {
		if _, selected := cmd().(SelectDBMsg); selected {
			t.Fatal("Enter selected database while filtering")
		}
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.state != ViewDBList || m.dbList.list.FilterState() != list.Unfiltered {
		t.Fatal("Escape did not clear applied database filter")
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.state != ViewCalendar {
		t.Fatal("Escape did not navigate back after clearing filter")
	}
}

func TestAppOmnisearchTypingAndCancellation(t *testing.T) {
	m := NewAppModel(&config.Config{NotionToken: "test"})
	_, _ = m.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
	for _, r := range "oqhr" {
		_, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if m.state != ViewOmnisearch || m.omnisearch.input.Value() != "oqhr" {
		t.Fatal("search typing triggered a shortcut")
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("Escape bypassed search cancellation")
	}
	_, _ = m.Update(cmd())
	if m.state != ViewDBList {
		t.Fatal("search cancellation did not return to previous screen")
	}
}

func TestAppControlCStillQuitsWhileTyping(t *testing.T) {
	m := NewAppModel(&config.Config{NotionToken: "test"})
	m.pushState(ViewEditor)
	m.editor.mode = modeCreate
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("ctrl+c was swallowed by input mode")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("ctrl+c did not quit")
	}
}

func TestAppEditorEscapeUsesCancellation(t *testing.T) {
	for _, mode := range []editorMode{modeCreate, modeEdit} {
		m := NewAppModel(&config.Config{NotionToken: "test"})
		m.pushState(ViewRecords)
		m.pushState(ViewEditor)
		m.editor.mode = mode
		// A previous detail page must not turn create cancellation into edit cancellation.
		m.editor.page = &notionapi.Page{ID: "previous", Properties: notionapi.Properties{}}
		_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
		if cmd == nil {
			t.Fatal("Escape bypassed editor cancellation")
		}
		_, _ = m.Update(cmd())
		if mode == modeCreate && m.state != ViewRecords {
			t.Fatal("create cancel did not return to records")
		}
		if mode == modeEdit && (m.state != ViewEditor || m.editor.mode != modeView) {
			t.Fatal("edit cancel did not return to detail view")
		}
	}
}

func TestAppRunsWithBubbleTea(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// An empty token exercises startup and rendering without contacting Notion.
	m := NewAppModel(&config.Config{})
	p := tea.NewProgram(m, tea.WithContext(ctx), tea.WithInput(nil),
		tea.WithOutput(io.Discard), tea.WithoutRenderer(), tea.WithoutSignalHandler(),
		tea.WithWindowSize(100, 40))
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	p.Send(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if err := <-done; err != nil {
		t.Fatalf("program failed to start or quit: %v", err)
	}
}

func TestAppViewUsesAlternateScreen(t *testing.T) {
	m := NewAppModel(&config.Config{})
	view := m.View()
	if !view.AltScreen || !strings.Contains(view.Content, "NOTION_TOKEN") {
		t.Fatal("missing alternate screen or configuration error")
	}
	if view.BackgroundColor != nil || view.ForegroundColor != nil {
		t.Fatal("view overrides terminal colors")
	}
}

func TestAppKeyPressAndRelease(t *testing.T) {
	m := NewAppModel(&config.Config{NotionToken: "test"})
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	_, _ = m.Update(tea.KeyReleaseMsg{Code: 'q', Text: "q"})
	if m.state != ViewDBList {
		t.Fatal("key release opened quit dialog")
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if m.state != ViewConfirm || !strings.Contains(m.View().Content, "Quit noctl?") {
		t.Fatal("quit popup was not rendered")
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if cmd == nil {
		t.Fatal("confirmation cancel produced no message")
	}
	_, _ = m.Update(cmd())
	if m.state != ViewDBList {
		t.Fatal("cancel did not restore previous screen")
	}
}

func TestAppFilteringAcceptsText(t *testing.T) {
	m := NewAppModel(&config.Config{NotionToken: "test"})
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	_, _ = m.Update(databasesMsg{{ID: "database", Title: []notionapi.RichText{{PlainText: "Tasks"}}}})
	_, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	_, _ = m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if m.state != ViewDBList || m.dbList.list.FilterInput.Value() != "q" {
		t.Fatalf("filter typing was treated as a global shortcut: state=%v input=%q", m.state, m.dbList.list.FilterInput.Value())
	}
}

func TestAppBackgroundColorUpdatesComponents(t *testing.T) {
	m := NewAppModel(&config.Config{NotionToken: "test"})
	for _, tc := range []struct {
		color color.Color
		dark  bool
	}{
		{color.White, false}, {color.Black, true},
	} {
		_, _ = m.Update(tea.BackgroundColorMsg{Color: tc.color})
		if m.editor.isDark != tc.dark {
			t.Fatalf("background detection: want dark=%v", tc.dark)
		}
		if m.dbList.list.Styles.Title.GetBackground() != TitleStyle.GetBackground() {
			t.Fatal("background detection replaced configured title style")
		}
	}
}

func TestRecordsControlNavigation(t *testing.T) {
	m := NewRecordsModel(nil)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m, _ = m.Update(recordsMsg{
		{ID: "first", Properties: notionapi.Properties{"Title": &notionapi.TitleProperty{Title: []notionapi.RichText{{PlainText: "First"}}}}},
		{ID: "second", Properties: notionapi.Properties{"Title": &notionapi.TitleProperty{Title: []notionapi.RichText{{PlainText: "Second"}}}}},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	if m.table.Cursor() != 1 {
		t.Fatal("ctrl+n did not move down")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	if m.table.Cursor() != 0 {
		t.Fatal("ctrl+p did not move up")
	}
}

func TestSelectorSpaceTogglesOption(t *testing.T) {
	m := NewSelectorModel()
	m.SetOptions("Tags", true, []SelectorOption{{Name: "selected"}}, nil)
	m, _ = m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter produced no selection")
	}
	selection := cmd().(SelectorDoneMsg)
	if len(selection.Values) != 1 || selection.Values[0] != "selected" {
		t.Fatalf("space did not toggle option: %+v", selection)
	}
}

func TestEditorRendersBodyForBothBackgrounds(t *testing.T) {
	for _, dark := range []bool{true, false} {
		m := NewEditorModel(nil)
		m.isDark = dark
		m.page = &notionapi.Page{ID: "page", Properties: notionapi.Properties{
			"Title": &notionapi.TitleProperty{Title: []notionapi.RichText{{PlainText: "Page title"}}},
		}}
		m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
		m, _ = m.Update(markdownMsg{pageID: "page", content: &notion.PageMarkdown{Markdown: "Body text"}})
		if !strings.Contains(ansi.Strip(m.View()), "Body text") {
			t.Fatalf("body missing for dark=%v", dark)
		}
		m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
		if m.viewport.Width() != 80 || m.viewport.Height() != 29 {
			t.Fatal("viewport did not resize")
		}
	}
}
