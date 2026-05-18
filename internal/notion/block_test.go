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
			expected: "Hello content\n\n",
		},
		{
			name: "Heading 1 Block",
			block: &notionapi.Heading1Block{
				Heading1: notionapi.Heading{
					RichText: []notionapi.RichText{{PlainText: "Main Title"}},
				},
			},
			expected: "# Main Title\n\n",
		},
		{
			name: "ToDo Block - Unchecked",
			block: &notionapi.ToDoBlock{
				ToDo: notionapi.ToDo{
					RichText: []notionapi.RichText{{PlainText: "Task 1"}},
					Checked:  false,
				},
			},
			expected: "- [ ] Task 1\n",
		},
		{
			name: "ToDo Block - Checked",
			block: &notionapi.ToDoBlock{
				ToDo: notionapi.ToDo{
					RichText: []notionapi.RichText{{PlainText: "Task 2"}},
					Checked:  true,
				},
			},
			expected: "- [x] Task 2\n",
		},
		{
			name: "Code Block",
			block: &notionapi.CodeBlock{
				Code: notionapi.Code{
					RichText: []notionapi.RichText{{PlainText: "fmt.Println(\"hi\")"}},
					Language: "go",
				},
			},
			expected: "```go\nfmt.Println(\"hi\")\n```\n\n",
		},
		{
			name: "Divider Block",
			block: &notionapi.DividerBlock{},
			expected: "---\n\n",
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

func TestBlocksToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		blocks   []notionapi.Block
		expected string
	}{
		{
			name: "Simple Table with Header",
			blocks: []notionapi.Block{
				&notionapi.TableBlock{
					BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeTableBlock},
					Table: notionapi.Table{
						HasRowHeader: true,
					},
				},
				&notionapi.TableRowBlock{
					BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeTableRowBlock},
					TableRow: notionapi.TableRow{
						Cells: [][]notionapi.RichText{
							{{PlainText: "Header 1"}},
							{{PlainText: "Header 2"}},
						},
					},
				},
				&notionapi.TableRowBlock{
					BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeTableRowBlock},
					TableRow: notionapi.TableRow{
						Cells: [][]notionapi.RichText{
							{{PlainText: "Data 1"}},
							{{PlainText: "Data 2"}},
						},
					},
				},
			},
			expected: "| Header 1 | Header 2 |\n| --- | --- |\n| Data 1 | Data 2 |\n\n",
		},
		{
			name: "Table without Header",
			blocks: []notionapi.Block{
				&notionapi.TableBlock{
					BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeTableBlock},
					Table: notionapi.Table{
						HasRowHeader: false,
					},
				},
				&notionapi.TableRowBlock{
					BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeTableRowBlock},
					TableRow: notionapi.TableRow{
						Cells: [][]notionapi.RichText{
							{{PlainText: "Data 1"}},
							{{PlainText: "Data 2"}},
						},
					},
				},
			},
			expected: "|   |   |\n| --- | --- |\n| Data 1 | Data 2 |\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BlocksToMarkdown(tt.blocks)
			if got != tt.expected {
				t.Errorf("BlocksToMarkdown() = %q, want %q", got, tt.expected)
			}
		})
	}
}
