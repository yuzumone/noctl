package notion

import (
	"testing"

	"github.com/jomei/notionapi"
)

func TestPropertyToString(t *testing.T) {
	tests := []struct {
		name     string
		property notionapi.Property
		expected string
	}{
		{
			name: "Title Property",
			property: &notionapi.TitleProperty{
				Title: []notionapi.RichText{
					{PlainText: "Hello "},
					{PlainText: "World"},
				},
			},
			expected: "Hello World",
		},
		{
			name: "RichText Property",
			property: &notionapi.RichTextProperty{
				RichText: []notionapi.RichText{
					{PlainText: "Some "},
					{PlainText: "Text"},
				},
			},
			expected: "Some Text",
		},
		{
			name: "Select Property",
			property: &notionapi.SelectProperty{
				Select: notionapi.Option{Name: "Option A"},
			},
			expected: "Option A",
		},
		{
			name: "MultiSelect Property",
			property: &notionapi.MultiSelectProperty{
				MultiSelect: []notionapi.Option{
					{Name: "Tag1"},
					{Name: "Tag2"},
				},
			},
			expected: "Tag1, Tag2",
		},
		{
			name: "Checkbox Property - True",
			property: &notionapi.CheckboxProperty{
				Checkbox: true,
			},
			expected: "✅",
		},
		{
			name: "Checkbox Property - False",
			property: &notionapi.CheckboxProperty{
				Checkbox: false,
			},
			expected: "☐",
		},
		{
			name: "Number Property",
			property: &notionapi.NumberProperty{
				Number: 123.45,
			},
			expected: "123.45",
		},
		{
			name: "URL Property",
			property: &notionapi.URLProperty{
				URL: "https://example.com",
			},
			expected: "https://example.com",
		},
		{
			name: "Files Property",
			property: &notionapi.FilesProperty{
				Files: []notionapi.File{
					{Name: "image.png"},
					{Name: "doc.pdf"},
				},
			},
			expected: "image.png, doc.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PropertyToString(tt.property)
			if got != tt.expected {
				t.Errorf("PropertyToString() = %v, want %v", got, tt.expected)
			}
		})
	}
}
