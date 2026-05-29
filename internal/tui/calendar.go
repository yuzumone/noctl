package tui

import (
	"context"
	"fmt"
	"time"

	"noctl/internal/notion"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"
)

type CalendarModel struct {
	client       *notion.Client
	dbIDs        []string
	currentDate  time.Time
	selectedDate time.Time
	allPages     []notionapi.Page
	loading      bool
	err          error
	width        int
	height       int
}

type OpenCalendarDetailsMsg struct {
	Date  time.Time
	Pages []notionapi.Page
}

func NewCalendarModel(client *notion.Client) CalendarModel {
	now := time.Now()
	return CalendarModel{
		client:       client,
		currentDate:  now,
		selectedDate: now,
	}
}

func (m CalendarModel) Init() tea.Cmd {
	return nil
}

func (m *CalendarModel) SetDatabases(ids []string) tea.Cmd {
	m.dbIDs = ids
	m.loading = true
	m.allPages = nil
	return m.fetchAllRecords
}

func (m CalendarModel) fetchAllRecords() tea.Msg {
	var allPages []notionapi.Page
	for _, id := range m.dbIDs {
		pages, err := m.client.QueryDatabaseAll(context.Background(), id)
		if err != nil {
			return errMsg(fmt.Errorf("failed to fetch records for %s: %w", id, err))
		}
		allPages = append(allPages, pages...)
	}
	return recordsMsg(allPages)
}

type SwitchToTableMsg struct {
	ID    string
	Title string
}

func (m CalendarModel) getPagesByDay(targetMonth time.Time) map[int][]notionapi.Page {
	pagesByDay := make(map[int][]notionapi.Page)
	for _, page := range m.allPages {
		var recordDate *time.Time
		for _, prop := range page.Properties {
			if dateProp, ok := prop.(*notionapi.DateProperty); ok && dateProp.Date != nil && dateProp.Date.Start != nil {
				t := time.Time(*dateProp.Date.Start)
				recordDate = &t
				break
			}
		}

		if recordDate != nil && recordDate.Year() == targetMonth.Year() && recordDate.Month() == targetMonth.Month() {
			pagesByDay[recordDate.Day()] = append(pagesByDay[recordDate.Day()], page)
		}
	}
	return pagesByDay
}

func (m CalendarModel) Update(msg tea.Msg) (CalendarModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "right", "l":
			m.selectedDate = m.selectedDate.AddDate(0, 0, 1)
			m.currentDate = m.selectedDate
		case "left", "h":
			m.selectedDate = m.selectedDate.AddDate(0, 0, -1)
			m.currentDate = m.selectedDate
		case "up", "k":
			m.selectedDate = m.selectedDate.AddDate(0, 0, -7)
			m.currentDate = m.selectedDate
		case "down", "j":
			m.selectedDate = m.selectedDate.AddDate(0, 0, 7)
			m.currentDate = m.selectedDate
		case "L":
			m.selectedDate = m.selectedDate.AddDate(0, 1, 0)
			m.currentDate = m.selectedDate
		case "H":
			m.selectedDate = m.selectedDate.AddDate(0, -1, 0)
			m.currentDate = m.selectedDate
		case "enter":
			pagesByDay := m.getPagesByDay(m.selectedDate)
			if pages, ok := pagesByDay[m.selectedDate.Day()]; ok && len(pages) > 0 {
				return m, func() tea.Msg {
					return OpenCalendarDetailsMsg{
						Date:  m.selectedDate,
						Pages: pages,
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case recordsMsg:
		m.loading = false
		m.allPages = msg

	case errMsg:
		m.err = msg
		m.loading = false
	}

	return m, nil
}

func (m CalendarModel) View() string {
	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Calendar View Error: %v", m.err))
	}

	header := TitleStyle.Width(m.width).Padding(0, 1).Render("Calendar") + "\n\n"
	if m.loading {
		header += "Loading records from all databases...\n"
	}

	calendar := m.renderCalendar()

	footer := renderFooter(m.width, []keyHelp{
		{"H/L", "Prev/Next Month"},
		{"o", "Omnisearch"},
		{"Esc", "Back"},
		{"q", "Quit"},
	})

	mainView := lipgloss.JoinVertical(lipgloss.Left,
		header,
		calendar,
	)

	// Ensure the view takes full height and push footer to the bottom
	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Height(m.height-1).Render(mainView),
		footer,
	)
}

func (m CalendarModel) renderCalendar() string {
	// Calculate dynamic dimensions
	// header (2) + monthYear (2) + dayHeaders (1) + footer (1) + margins/padding
	staticHeight := 6 // Reduced from 8
	availableHeight := m.height - staticHeight
	if availableHeight < 0 {
		availableHeight = 0
	}

	firstDayOfMonth := time.Date(m.currentDate.Year(), m.currentDate.Month(), 1, 0, 0, 0, 0, m.currentDate.Location())
	startOffset := int(firstDayOfMonth.Weekday())
	daysInMonth := time.Date(m.currentDate.Year(), m.currentDate.Month()+1, 0, 0, 0, 0, 0, m.currentDate.Location()).Day()
	numWeeks := (startOffset + daysInMonth + 6) / 7

	// Lipgloss Width/Height sets the content size. Borders add +2 to each dimension.
	cellWidth := (m.width / 7) - 2
	if cellWidth < 4 {
		cellWidth = 4
	}
	cellHeight := (availableHeight / numWeeks) - 2
	if cellHeight < 1 {
		cellHeight = 1
	}

	// Month and Year
	monthYear := m.currentDate.Format("January 2006")
	monthYearStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ActiveTheme.Accent)).MarginBottom(0) // Reduced margin

	s := monthYearStyle.Render(monthYear) + "\n"

	// Days of week
	daysOfWeek := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	var dayHeaders []string
	for _, day := range daysOfWeek {
		// Day header width should match the total cell width (content + borders)
		dayHeaders = append(dayHeaders, lipgloss.NewStyle().Width(cellWidth+2).Align(lipgloss.Center).Render(day))
	}
	s += lipgloss.JoinHorizontal(lipgloss.Top, dayHeaders...) + "\n"

	// Prepare records map
	pagesByDay := m.getPagesByDay(m.currentDate)

	cellStyle := lipgloss.NewStyle().
		Width(cellWidth).
		Height(cellHeight).
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(lipgloss.Color(ActiveTheme.Dimmed))

	today := time.Now()
	isTodayMonth := today.Year() == m.currentDate.Year() && today.Month() == m.currentDate.Month()
	isSelectedMonth := m.selectedDate.Year() == m.currentDate.Year() && m.selectedDate.Month() == m.currentDate.Month()

	var rows []string
	var currentRow []string

	// Empty cells before first day
	for i := 0; i < startOffset; i++ {
		currentRow = append(currentRow, cellStyle.Render(""))
	}

	// We have cellHeight lines available. First line is always the day number.
	// So we have room for cellHeight - 1 records.
	maxRecords := cellHeight - 1

	for day := 1; day <= daysInMonth; day++ {
		if len(currentRow) == 7 {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, currentRow...))
			currentRow = nil
		}

		content := fmt.Sprintf("%d", day)
		style := cellStyle

		// Today highlight (dimmed accent or bold)
		if isTodayMonth && day == today.Day() {
			style = style.BorderForeground(lipgloss.Color(ActiveTheme.Accent)).Bold(true)
		}

		// Selection highlight (inverse colors or thicker border)
		if isSelectedMonth && day == m.selectedDate.Day() {
			style = style.
				Background(lipgloss.Color(ActiveTheme.SelectBg)).
				Foreground(lipgloss.Color(ActiveTheme.SelectFg)).
				BorderForeground(lipgloss.Color(ActiveTheme.SelectBg))
		}

		dayPages := pagesByDay[day]
		for i, page := range dayPages {
			if i >= maxRecords {
				if maxRecords > 0 {
					content += "\n..."
				}
				break
			}
			title := notion.PropertyToString(page.Properties[m.getTitleKey(page)])
			truncated := title
			runes := []rune(title)
			if len(runes) > cellWidth {
				truncated = string(runes[:cellWidth-3]) + "..."
			}
			content += "\n" + truncated
		}

		currentRow = append(currentRow, style.Render(content))
	}

	// Empty cells after last day
	for len(currentRow) < 7 {
		currentRow = append(currentRow, cellStyle.Render(""))
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, currentRow...))

	s += lipgloss.JoinVertical(lipgloss.Left, rows...)

	return s
}

func (m CalendarModel) getTitleKey(page notionapi.Page) string {
	for k, p := range page.Properties {
		if _, ok := p.(*notionapi.TitleProperty); ok {
			return k
		}
	}
	return ""
}
