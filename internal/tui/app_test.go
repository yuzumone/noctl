package tui

import (
	"context"
	"image/color"
	"io"
	"strings"
	"testing"
	"time"

	"noctl/internal/config"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/jomei/notionapi"
)

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
		m, _ = m.Update(blocksMsg{&notionapi.ParagraphBlock{
			Paragraph: notionapi.Paragraph{RichText: []notionapi.RichText{{PlainText: "Body text"}}},
		}})
		if !strings.Contains(ansi.Strip(m.View()), "Body text") {
			t.Fatalf("body missing for dark=%v", dark)
		}
		m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
		if m.viewport.Width() != 80 || m.viewport.Height() != 29 {
			t.Fatal("viewport did not resize")
		}
	}
}
