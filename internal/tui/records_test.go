package tui

import (
	"errors"
	"testing"

	"noctl/internal/config"

	tea "charm.land/bubbletea/v2"
	"github.com/jomei/notionapi"
)

func recordPages(id string) []notionapi.Page {
	return []notionapi.Page{{ID: notionapi.ObjectID(id), Properties: notionapi.Properties{
		"Title": &notionapi.TitleProperty{Title: []notionapi.RichText{{PlainText: id}}},
	}}}
}

func TestAppIgnoresRecordsFromPreviousDatabase(t *testing.T) {
	m := NewAppModel(&config.Config{NotionToken: "test"})
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	_, _ = m.Update(SelectDBMsg{ID: "first", Title: "First"})
	old := recordsMsg{dbID: "first", requestID: m.records.requestID, pages: recordPages("old")}
	_, _ = m.Update(SelectDBMsg{ID: "second", Title: "Second"})
	_, _ = m.Update(old)
	if len(m.recordsCache) != 0 || !m.records.loading || len(m.records.allPages) != 0 {
		t.Fatal("previous database result polluted current records or cache")
	}
	_, _ = m.Update(recordsMsg{dbID: "second", requestID: m.records.requestID, pages: recordPages("new")})
	if m.records.loading || m.records.pages[0].ID != "new" || m.recordsCache["second"][0].ID != "new" {
		t.Fatal("current database result was not applied")
	}
	old.err = errors.New("late error")
	_, _ = m.Update(old)
	if m.records.err != nil || m.recordsCache["second"][0].ID != "new" {
		t.Fatal("late error affected current database")
	}
}

func TestAppIgnoresOlderRefreshOfSameDatabase(t *testing.T) {
	m := NewAppModel(&config.Config{NotionToken: "test"})
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	_, _ = m.Update(SelectDBMsg{ID: "database", Title: "Database"})
	old := recordsMsg{dbID: "database", requestID: m.records.requestID, pages: recordPages("old")}
	_, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	_, _ = m.Update(recordsMsg{dbID: "database", requestID: m.records.requestID, pages: recordPages("new")})
	_, _ = m.Update(old)
	if m.recordsCache["database"][0].ID != "new" || m.records.allPages[0].ID != "new" {
		t.Fatal("older refresh overwrote newer data")
	}
	old.err = errors.New("old refresh failed")
	_, _ = m.Update(old)
	if m.records.err != nil {
		t.Fatal("older refresh error replaced newer success")
	}
}

func TestAppRoutesRecordErrorsToOwner(t *testing.T) {
	m := NewAppModel(&config.Config{NotionToken: "test"})
	_, _ = m.Update(SelectDBMsg{ID: "database", Title: "Database"})
	m.pushState(ViewEditor)
	want := errors.New("query failed")
	_, _ = m.Update(recordsMsg{dbID: "database", requestID: m.records.requestID, err: want})
	if m.records.err != want || m.records.loading || m.editor.err != nil || m.state != ViewEditor || len(m.recordsCache) != 0 {
		t.Fatal("record error was routed to the active view instead of its owner")
	}
	_, _ = m.Update(SelectDBMsg{ID: "database", Title: "Database"})
	if m.records.err != nil {
		t.Fatal("retry retained the previous query error")
	}
}

func TestRecordsResultArrivesWhileFiltering(t *testing.T) {
	m := NewRecordsModel(nil)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	_ = m.SetDatabase("database", "Database", nil)
	m, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	m, _ = m.Update(recordsMsg{dbID: "database", requestID: m.requestID, pages: recordPages("x-page")})
	if m.loading || !m.filtering || len(m.pages) != 1 || m.filterInput.Value() != "x" {
		t.Fatal("filtering swallowed the query response")
	}
}

func TestAppCachesEmptyDatabase(t *testing.T) {
	m := NewAppModel(&config.Config{NotionToken: "test"})
	_, _ = m.Update(SelectDBMsg{ID: "empty", Title: "Empty"})
	_, _ = m.Update(recordsMsg{dbID: "empty", requestID: m.records.requestID})
	cache, ok := m.recordsCache["empty"]
	if !ok || cache == nil {
		t.Fatal("empty database was not cached")
	}
	_, cmd := m.Update(SelectDBMsg{ID: "empty", Title: "Empty"})
	if cmd != nil || m.records.loading {
		t.Fatal("empty cache hit triggered another request")
	}
}

func TestCalendarAndTableResultsAreSeparate(t *testing.T) {
	m := NewAppModel(&config.Config{NotionToken: "test", CalendarDatabaseIDs: []string{"calendar"}})
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	_ = m.calendar.SetDatabases([]string{"calendar"}, nil)
	calendarResult := calendarRecordsMsg{requestID: m.calendar.requestID, pages: recordPages("event")}
	_, _ = m.Update(SelectDBMsg{ID: "table", Title: "Table"})
	_, _ = m.Update(calendarResult)
	if m.calendar.loading || len(m.calendar.allPages) != 1 || !m.records.loading || len(m.recordsCache) != 0 {
		t.Fatal("calendar result leaked into table")
	}
	m.state = ViewCalendar
	_, _ = m.Update(recordsMsg{dbID: "table", requestID: m.records.requestID, pages: recordPages("row")})
	if m.calendar.allPages[0].ID != "event" || m.recordsCache["table"][0].ID != "row" {
		t.Fatal("table result leaked into calendar")
	}
	_ = m.calendar.SetDatabases([]string{"calendar"}, nil)
	calendarResult.err = errors.New("old calendar failure")
	_, _ = m.Update(calendarResult)
	if m.calendar.err != nil || !m.calendar.loading {
		t.Fatal("old calendar request affected refresh")
	}
	want := errors.New("calendar failed")
	_, _ = m.Update(calendarRecordsMsg{requestID: m.calendar.requestID, err: want})
	if m.calendar.err != want || m.calendar.loading || m.records.err != nil {
		t.Fatal("calendar error was lost or misrouted")
	}
}

func TestCalendarRequiresCompleteRelevantCache(t *testing.T) {
	m := NewAppModel(&config.Config{CalendarDatabaseIDs: []string{"first", "second"}})
	m.recordsCache["unrelated"] = recordPages("unrelated")
	m.recordsCache["first"] = recordPages("first-page")
	if m.getAllCachedRecords() != nil {
		t.Fatal("partial cache skipped fetching missing calendar database")
	}
	m.recordsCache["second"] = []notionapi.Page{}
	cache := m.getAllCachedRecords()
	if len(cache) != 1 || cache[0].ID != "first-page" {
		t.Fatal("calendar cache included unrelated records")
	}
	m.recordsCache["first"] = []notionapi.Page{}
	if cmd := m.calendar.SetDatabases(m.config.CalendarDatabaseIDs, m.getAllCachedRecords()); cmd != nil || m.calendar.loading {
		t.Fatal("complete empty calendar cache triggered another request")
	}
}

func TestAppRecordsCompleteBehindPopup(t *testing.T) {
	m := NewAppModel(&config.Config{NotionToken: "test"})
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	_, _ = m.Update(SelectDBMsg{ID: "database", Title: "Database"})
	_, _ = m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	_, _ = m.Update(recordsMsg{dbID: "database", requestID: m.records.requestID, pages: recordPages("page")})
	if m.state != ViewConfirm || m.records.loading || len(m.recordsCache["database"]) != 1 {
		t.Fatal("record result was lost while the quit dialog was open")
	}
}
