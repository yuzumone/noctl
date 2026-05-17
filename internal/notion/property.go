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

func TextItemsToString(items []notionapi.RichText) string {
	var s []string
	for _, item := range items {
		s = append(s, item.PlainText)
	}
	return strings.Join(s, "")
}
