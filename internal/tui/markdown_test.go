package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jomei/notionapi"
	"noctl/internal/notion"
)

func TestMarkdownForDisplay(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"<b>bold</b> <i>italic</i> <code>x</code>", "**bold** *italic* `x`"},
		{"| A | B |\n| --- | --- |\n| 1 | 2 |", "| A | B |\n| --- | --- |\n| 1 | 2 |"},
		{"```html\n<b>literal</b>\n```\n<b>bold</b>", "```html\n<b>literal</b>\n```\n**bold**"},
		{"`<b>literal</b>` <b>bold</b>", "`<b>literal</b>` **bold**"},
		{`<bookmark source="https://example.com" />`, "[bookmark](https://example.com) "},
		{"<callout>\n本文\n</callout>", "💡 \n本文\n\n\n"},
	} {
		if got := markdownForDisplay(tc.input); got != tc.want {
			t.Errorf("display(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestEditorUnchangedMarkdownDoesNotSave(t *testing.T) {
	file := filepath.Join(t.TempDir(), "page.md")
	const original = "<b>Original</b>\n"
	if err := os.WriteFile(file, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	m := NewEditorModel(nil)
	m.tempFile, m.editOriginal = file, original
	if cmd := m.handleEditorFinished(editorFinishedMsg{}); cmd != nil {
		t.Fatal("unchanged content scheduled a save")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("unchanged temp file not removed: %v", err)
	}
}

func TestEditorPreservesFailedAndWarningSaves(t *testing.T) {
	for _, warning := range []bool{false, true} {
		file := filepath.Join(t.TempDir(), "page.md")
		if err := os.WriteFile(file, []byte("edited content"), 0600); err != nil {
			t.Fatal(err)
		}
		m := NewEditorModel(nil)
		m.page = &notionapi.Page{ID: "page-id"}
		msg := contentSavedMsg{pageID: "page-id", file: file, err: errors.New("failed")}
		if warning {
			msg.err = nil
			msg.content = &notion.PageMarkdown{Warnings: []notion.MarkdownWarning{{Code: "unsupported_tag", Message: "warning"}}}
		}
		m, cmd := m.Update(msg)
		if cmd != nil || m.err == nil || !strings.Contains(m.err.Error(), file) {
			t.Fatalf("missing recoverable error: %v", m.err)
		}
		if data, err := os.ReadFile(file); err != nil || string(data) != "edited content" {
			t.Fatalf("edited file lost: %v", err)
		}
	}
}

func TestEditorIgnoresMarkdownForAnotherPage(t *testing.T) {
	m := NewEditorModel(nil)
	m.page = &notionapi.Page{ID: "current"}
	m, _ = m.Update(markdownMsg{pageID: "old", content: &notion.PageMarkdown{Markdown: "old body"}})
	if m.markdown != nil {
		t.Fatal("stale result replaced current content")
	}
}

func TestEditorBlocksIncompleteContentEditing(t *testing.T) {
	m := NewEditorModel(nil)
	m.page = &notionapi.Page{ID: "current"}
	m.markdown = &notion.PageMarkdown{Truncated: true}
	if cmd := m.openInEditor(); cmd != nil || m.err == nil || m.tempFile != "" {
		t.Fatal("incomplete content allowed editing")
	}
}
