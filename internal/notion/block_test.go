package notion

import (
	"testing"

	"github.com/jomei/notionapi"
)

func TestBlockToString(t *testing.T) {
	tests := []struct {
		name     string
		block    notionapi.Block
		expected string
	}{
		{
			name: "Paragraph Block",
			block: &notionapi.ParagraphBlock{
				Paragraph: notionapi.Paragraph{
					RichText: []notionapi.RichText{{PlainText: "Hello content"}},
				},
			},
			expected: "Hello content\n",
		},
		{
			name: "Heading 1 Block",
			block: &notionapi.Heading1Block{
				Heading1: notionapi.Heading{
					RichText: []notionapi.RichText{{PlainText: "Main Title"}},
				},
			},
			expected: "# Main Title\n",
		},
		{
			name: "ToDo Block - Unchecked",
			block: &notionapi.ToDoBlock{
				ToDo: notionapi.ToDo{
					RichText: []notionapi.RichText{{PlainText: "Task 1"}},
					Checked:  false,
				},
			},
			expected: "☐ Task 1\n",
		},
		{
			name: "ToDo Block - Checked",
			block: &notionapi.ToDoBlock{
				ToDo: notionapi.ToDo{
					RichText: []notionapi.RichText{{PlainText: "Task 2"}},
					Checked:  true,
				},
			},
			expected: "✅ Task 2\n",
		},
		{
			name: "Code Block",
			block: &notionapi.CodeBlock{
				Code: notionapi.Code{
					RichText: []notionapi.RichText{{PlainText: "fmt.Println(\"hi\")"}},
					Language: "go",
				},
			},
			expected: "```go\nfmt.Println(\"hi\")\n```\n",
		},
		{
			name: "Divider Block",
			block: &notionapi.DividerBlock{},
			expected: "---\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BlockToString(tt.block)
			if got != tt.expected {
				t.Errorf("BlockToString() = %q, want %v", got, tt.expected)
			}
		})
	}
}
