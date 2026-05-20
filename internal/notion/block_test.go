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
			name:     "Divider Block",
			block:    &notionapi.DividerBlock{},
			expected: "---\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BlockToString(tt.block, false)
			if got != tt.expected {
				t.Errorf("BlockToString() = %q, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseContentToBlocks(t *testing.T) {
	c := &Client{}
	tests := []struct {
		name     string
		content  string
		expected []notionapi.BlockType
	}{
		{
			name:    "Heading and Paragraph",
			content: "# Heading\nParagraph",
			expected: []notionapi.BlockType{
				notionapi.BlockTypeHeading1,
				notionapi.BlockTypeParagraph,
			},
		},
		{
			name:    "List Items",
			content: "- Item 1\n- Item 2",
			expected: []notionapi.BlockType{
				notionapi.BlockTypeBulletedListItem,
				notionapi.BlockTypeBulletedListItem,
			},
		},
		{
			name:    "Quote and Code",
			content: "> Quote\n```go\ncode\n```",
			expected: []notionapi.BlockType{
				notionapi.BlockTypeQuote,
				notionapi.BlockTypeCode,
			},
		},
		{
			name:    "ToDo and Divider",
			content: "- [ ] Task\n---",
			expected: []notionapi.BlockType{
				notionapi.BlockTypeToDo,
				notionapi.BlockTypeDivider,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := c.ParseContentToBlocks(tt.content)
			if len(blocks) != len(tt.expected) {
				t.Fatalf("got %d blocks, want %d", len(blocks), len(tt.expected))
			}
			for i, b := range blocks {
				if b.GetType() != tt.expected[i] {
					t.Errorf("block %d: got type %s, want %s", i, b.GetType(), tt.expected[i])
				}
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
