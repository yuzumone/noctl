package tui

import (
	"testing"

	"github.com/jomei/notionapi"
)

func TestTableSelectorModel_SetPages(t *testing.T) {
	m := NewTableSelectorModel()
	m.width = 100
	m.height = 50

	pages := []notionapi.Page{
		{
			ID: "1",
			Properties: notionapi.Properties{
				"Name": &notionapi.TitleProperty{
					Title: []notionapi.RichText{{PlainText: "Test Page"}},
				},
			},
		},
	}

	m.SetPages("Test Title", pages)

	if len(m.pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(m.pages))
	}

	if m.title != "Test Title" {
		t.Errorf("expected title 'Test Title', got %s", m.title)
	}

	// Check if table rows were populated
	if len(m.table.Rows()) != 1 {
		t.Errorf("expected 1 table row, got %d", len(m.table.Rows()))
	}
}
