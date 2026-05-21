// Package notion handles Notion block conversions and API interactions.
package notion

import (
	"fmt"
	"strings"
	"time"

	"github.com/jomei/notionapi"
)

// PropertyToString converts a Notion property to a string representation for display.
func PropertyToString(prop notionapi.Property) string {
	switch p := prop.(type) {
	case *notionapi.TitleProperty:
		return TextItemsToString(p.Title)
	case *notionapi.RichTextProperty:
		return TextItemsToString(p.RichText)
	case *notionapi.SelectProperty:
		if p.Select.Name == "" {
			return ""
		}
		return p.Select.Name
	case *notionapi.MultiSelectProperty:
		var names []string
		for _, o := range p.MultiSelect {
			names = append(names, o.Name)
		}
		return strings.Join(names, ", ")
	case *notionapi.NumberProperty:
		return fmt.Sprintf("%v", p.Number)
	case *notionapi.CheckboxProperty:
		if p.Checkbox {
			return "✅"
		}
		return "☐"
	case *notionapi.DateProperty:
		if p.Date == nil {
			return ""
		}
		formatDate := func(dt *notionapi.Date) string {
			if dt == nil {
				return ""
			}
			t := time.Time(*dt)
			s := t.Format("2006-01-02")
			if t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0 {
				s = t.Format("2006-01-02 15:04")
			}
			return s
		}
		s := formatDate(p.Date.Start)
		if p.Date.End != nil {
			s += " -> " + formatDate(p.Date.End)
		}
		return s
	case *notionapi.CreatedTimeProperty:
		return p.CreatedTime.Format("2006-01-02 15:04")
	case *notionapi.LastEditedTimeProperty:
		return p.LastEditedTime.Format("2006-01-02 15:04")
	case *notionapi.PeopleProperty:
		var names []string
		for _, person := range p.People {
			names = append(names, person.Name)
		}
		return strings.Join(names, ", ")
	case *notionapi.CreatedByProperty:
		return p.CreatedBy.Name
	case *notionapi.LastEditedByProperty:
		return p.LastEditedBy.Name
	case *notionapi.RelationProperty:
		count := len(p.Relation)
		if count == 0 {
			return "🔗 (none)"
		}
		if count == 1 {
			return "🔗 1 relation"
		}
		return fmt.Sprintf("🔗 %d relations", count)
	case *notionapi.RollupProperty:
		switch p.Rollup.Type {
		case notionapi.RollupTypeNumber:
			return fmt.Sprintf("%v", p.Rollup.Number)
		case notionapi.RollupTypeDate:
			if p.Rollup.Date != nil {
				return time.Time(*p.Rollup.Date.Start).Format("2006-01-02")
			}
		case notionapi.RollupTypeArray:
			return fmt.Sprintf("%d items", len(p.Rollup.Array))
		}
		return string(p.Rollup.Type)
	case *notionapi.URLProperty:
		return p.URL
	case *notionapi.EmailProperty:
		return p.Email
	case *notionapi.PhoneNumberProperty:
		return p.PhoneNumber
	case *notionapi.StatusProperty:
		return p.Status.Name
	case *notionapi.FilesProperty:
		var names []string
		for _, f := range p.Files {
			names = append(names, f.Name)
		}
		return strings.Join(names, ", ")
	case *notionapi.FormulaProperty:
		switch p.Formula.Type {
		case "string":
			return p.Formula.String
		case "number":
			return fmt.Sprintf("%v", p.Formula.Number)
		case "boolean":
			return fmt.Sprintf("%v", p.Formula.Boolean)
		case "date":
			if p.Formula.Date == nil {
				return ""
			}
			return time.Time(*p.Formula.Date.Start).Format("2006-01-02")
		}
		return string(p.Formula.Type)
	}
	return ""
}

// PropertyToIcon returns a Nerd Font icon for a given Notion property.
func PropertyToIcon(prop notionapi.Property) string {
	switch prop.(type) {
	case *notionapi.TitleProperty:
		return "󰗚"
	case *notionapi.RichTextProperty:
		return "󰦨"
	case *notionapi.SelectProperty:
		return "󰦪"
	case *notionapi.MultiSelectProperty:
		return "󰦫"
	case *notionapi.NumberProperty:
		return "󰎠"
	case *notionapi.CheckboxProperty:
		return "󰄬"
	case *notionapi.DateProperty:
		return "󰃭"
	case *notionapi.URLProperty:
		return "󰖟"
	case *notionapi.EmailProperty:
		return "󰇮"
	case *notionapi.PhoneNumberProperty:
		return "󰏲"
	case *notionapi.StatusProperty:
		return "󰗡"
	case *notionapi.FilesProperty:
		return "󰈔"
	case *notionapi.FormulaProperty:
		return "󰘚"
	case *notionapi.CreatedTimeProperty, *notionapi.LastEditedTimeProperty:
		return "󰃰"
	case *notionapi.CreatedByProperty, *notionapi.LastEditedByProperty, *notionapi.PeopleProperty:
		return "󰇔"
	case *notionapi.RelationProperty:
		return "󰌷"
	case *notionapi.RollupProperty:
		return "󰪚"
	}
	return "󰇗" // Default icon
}

// TextItemsToString converts a slice of Notion RichText items into a single plain text string.
func TextItemsToString(items []notionapi.RichText) string {
	var s []string
	for _, item := range items {
		s = append(s, item.PlainText)
	}
	return strings.Join(s, "")
}

// RichTextToMarkdown converts a slice of Notion RichText items into a markdown string, preserving basic formatting.
func RichTextToMarkdown(items []notionapi.RichText) string {
	var sb strings.Builder
	for _, item := range items {
		text := item.PlainText
		if item.Annotations != nil {
			if item.Annotations.Code {
				text = "`" + text + "`"
			} else {
				if item.Annotations.Bold {
					text = "**" + text + "**"
				}
				if item.Annotations.Italic {
					text = "_" + text + "_"
				}
				if item.Annotations.Strikethrough {
					text = "~~" + text + "~~"
				}
			}
			// Note: Underline and Color are not easily represented in standard Markdown
		}
		if item.Href != "" {
			text = "[" + text + "](" + item.Href + ")"
		}
		sb.WriteString(text)
	}
	return sb.String()
}
