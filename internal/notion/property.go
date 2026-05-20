package notion

import (
	"fmt"
	"strings"

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
		s := p.Date.Start.String()
		if p.Date.End != nil {
			s += " -> " + p.Date.End.String()
		}
		return s
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
			return p.Formula.Date.Start.String()
		}
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
