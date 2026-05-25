package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jomei/notionapi"
)

func TestCalendarModel_getPagesByDay(t *testing.T) {
	m := NewCalendarModel(nil)

	// Create some dummy pages
	date1 := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)
	date3 := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	dateOtherMonth := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	createPage := func(id string, dt *time.Time) notionapi.Page {
		p := notionapi.Page{
			ID: notionapi.ObjectID(id),
			Properties: notionapi.Properties{
				"Title": &notionapi.TitleProperty{
					Title: []notionapi.RichText{{PlainText: "Page " + id}},
				},
			},
		}
		if dt != nil {
			ndt := notionapi.Date(*dt)
			p.Properties["Date"] = &notionapi.DateProperty{
				Date: &notionapi.DateObject{Start: &ndt},
			}
		}
		return p
	}

	m.allPages = []notionapi.Page{
		createPage("1", &date1),
		createPage("2", &date2),
		createPage("3", &date3),
		createPage("4", &dateOtherMonth),
		createPage("5", nil), // No date
	}

	pagesByDay := m.getPagesByDay(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))

	if len(pagesByDay[24]) != 2 {
		t.Errorf("expected 2 pages on day 24, got %d", len(pagesByDay[24]))
	}
	if len(pagesByDay[25]) != 1 {
		t.Errorf("expected 1 page on day 25, got %d", len(pagesByDay[25]))
	}
	if len(pagesByDay) != 2 {
		t.Errorf("expected 2 days with pages, got %d", len(pagesByDay))
	}
}

func TestCalendarModel_Navigation(t *testing.T) {
	m := NewCalendarModel(nil)
	initialDate := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	m.selectedDate = initialDate
	m.currentDate = initialDate

	// Test Move Right
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.selectedDate.Day() != 11 {
		t.Errorf("expected day 11 after right, got %d", m.selectedDate.Day())
	}

	// Test Move Left
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.selectedDate.Day() != 10 {
		t.Errorf("expected day 10 after left, got %d", m.selectedDate.Day())
	}

	// Test Move Down (1 week)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.selectedDate.Day() != 17 {
		t.Errorf("expected day 17 after down, got %d", m.selectedDate.Day())
	}

	// Test Month Jump (L)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	if m.selectedDate.Month() != time.June {
		t.Errorf("expected June after L, got %s", m.selectedDate.Month())
	}
}
