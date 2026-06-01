package tui

import (
	"reflect"
	"testing"

	"github.com/jomei/notionapi"
)

func TestEditablePropertyKeys(t *testing.T) {
	configs := map[string]notionapi.PropertyConfig{
		"URL":    &notionapi.URLPropertyConfig{Type: notionapi.PropertyConfigTypeURL},
		"Title":  &notionapi.TitlePropertyConfig{Type: notionapi.PropertyConfigTypeTitle},
		"Status": &notionapi.SelectPropertyConfig{Type: notionapi.PropertyConfigTypeSelect},
		"Created": &notionapi.CreatedTimePropertyConfig{
			ID:   "created",
			Type: notionapi.PropertyConfigCreatedTime,
		},
		"Notes": &notionapi.RichTextPropertyConfig{Type: notionapi.PropertyConfigTypeRichText},
	}

	got := editablePropertyKeys(configs)
	want := []string{"Title", "Notes", "Status", "URL"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("editablePropertyKeys() = %v, want %v", got, want)
	}
}

func TestInputProperty(t *testing.T) {
	tests := []struct {
		name   string
		config notionapi.PropertyConfig
		value  string
		wantOK bool
		assert func(t *testing.T, prop notionapi.Property)
	}{
		{
			name:   "title",
			config: &notionapi.TitlePropertyConfig{Type: notionapi.PropertyConfigTypeTitle},
			value:  "Task",
			wantOK: true,
			assert: func(t *testing.T, prop notionapi.Property) {
				p, ok := prop.(notionapi.TitleProperty)
				if !ok {
					t.Fatalf("prop type = %T, want notionapi.TitleProperty", prop)
				}
				if got := p.Title[0].Text.Content; got != "Task" {
					t.Fatalf("title content = %q, want %q", got, "Task")
				}
			},
		},
		{
			name:   "invalid number",
			config: &notionapi.NumberPropertyConfig{Type: notionapi.PropertyConfigTypeNumber},
			value:  "not-a-number",
			wantOK: false,
		},
		{
			name:   "multi select trims values",
			config: &notionapi.MultiSelectPropertyConfig{Type: notionapi.PropertyConfigTypeMultiSelect},
			value:  "bug,  feature, ",
			wantOK: true,
			assert: func(t *testing.T, prop notionapi.Property) {
				p, ok := prop.(notionapi.MultiSelectProperty)
				if !ok {
					t.Fatalf("prop type = %T, want notionapi.MultiSelectProperty", prop)
				}
				got := []string{}
				for _, option := range p.MultiSelect {
					got = append(got, option.Name)
				}
				want := []string{"bug", "feature"}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("multi select options = %v, want %v", got, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prop, ok := inputProperty(tt.config, tt.value)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if tt.assert != nil {
				tt.assert(t, prop)
			}
		})
	}
}
