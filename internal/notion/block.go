package notion

import (
	"context"
	"fmt"
	"strings"

	"github.com/jomei/notionapi"
)

// BlockToString converts a Notion block to its markdown representation.
func BlockToString(block notionapi.Block) string {
	switch b := block.(type) {
	case *notionapi.ParagraphBlock:
		return RichTextToMarkdown(b.Paragraph.RichText) + "\n\n"
	case *notionapi.Heading1Block:
		return "# " + RichTextToMarkdown(b.Heading1.RichText) + "\n\n"
	case *notionapi.Heading2Block:
		return "## " + RichTextToMarkdown(b.Heading2.RichText) + "\n\n"
	case *notionapi.Heading3Block:
		return "### " + RichTextToMarkdown(b.Heading3.RichText) + "\n\n"
	case *notionapi.BulletedListItemBlock:
		return "- " + RichTextToMarkdown(b.BulletedListItem.RichText) + "\n"
	case *notionapi.NumberedListItemBlock:
		return "1. " + RichTextToMarkdown(b.NumberedListItem.RichText) + "\n"
	case *notionapi.ToDoBlock:
		check := "[ ]"
		if b.ToDo.Checked {
			check = "[x]"
		}
		return "- " + check + " " + RichTextToMarkdown(b.ToDo.RichText) + "\n"
	case *notionapi.ToggleBlock:
		return "> " + RichTextToMarkdown(b.Toggle.RichText) + "\n\n"
	case *notionapi.CodeBlock:
		return "```" + b.Code.Language + "\n" + RichTextToMarkdown(b.Code.RichText) + "\n```\n\n"
	case *notionapi.QuoteBlock:
		return "> " + RichTextToMarkdown(b.Quote.RichText) + "\n\n"
	case *notionapi.CalloutBlock:
		return "> 💡 " + RichTextToMarkdown(b.Callout.RichText) + "\n\n"
	case *notionapi.DividerBlock:
		return "---\n\n"
	}
	return ""
}

// RenderTable converts table rows into a Markdown table string.
func RenderTable(rows []notionapi.Block, hasHeader bool) string {
	if len(rows) == 0 {
		return ""
	}

	var sb strings.Builder
	columnCount := 0

	// Get column count from first row
	if firstRow, ok := rows[0].(*notionapi.TableRowBlock); ok {
		columnCount = len(firstRow.TableRow.Cells)
	} else {
		return ""
	}

	if !hasHeader {
		// Add an empty header row if Notion table doesn't have one,
		// because Markdown requires a header row for tables.
		sb.WriteString("|")
		for i := 0; i < columnCount; i++ {
			sb.WriteString("   |")
		}
		sb.WriteString("\n|")
		for i := 0; i < columnCount; i++ {
			sb.WriteString(" --- |")
		}
		sb.WriteString("\n")
	}

	for i, rowBlock := range rows {
		row, ok := rowBlock.(*notionapi.TableRowBlock)
		if !ok {
			continue
		}

		sb.WriteString("|")
		for _, cell := range row.TableRow.Cells {
			// Replace newlines with spaces to avoid breaking MD table
			text := strings.ReplaceAll(RichTextToMarkdown(cell), "\n", " ")
			// Escape pipe characters
			text = strings.ReplaceAll(text, "|", "\\|")
			sb.WriteString(" " + text + " |")
		}
		sb.WriteString("\n")

		// Add separator after first row if it's a header
		if hasHeader && i == 0 {
			sb.WriteString("|")
			for j := 0; j < columnCount; j++ {
				sb.WriteString(" --- |")
			}
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")
	return sb.String()
}

// BlocksToMarkdown converts a slice of Notion blocks to a single markdown string.
func BlocksToMarkdown(blocks []notionapi.Block) string {
	var sb strings.Builder
	var tableRows []notionapi.Block
	inTable := false
	hasHeader := true

	for _, b := range blocks {
		if b.GetType() == notionapi.BlockTypeTableRowBlock {
			tableRows = append(tableRows, b)
			inTable = true
			continue
		}

		if inTable {
			sb.WriteString(RenderTable(tableRows, hasHeader))
			tableRows = nil
			inTable = false
		}

		if tb, ok := b.(*notionapi.TableBlock); ok {
			hasHeader = tb.Table.HasRowHeader
			continue
		}

		sb.WriteString(BlockToString(b))
	}

	// Catch any trailing table
	if inTable {
		sb.WriteString(RenderTable(tableRows, hasHeader))
	}

	return sb.String()
}

// ListBlocks fetches blocks (content) of a page or block.
func (c *Client) ListBlocks(ctx context.Context, id string, cursor notionapi.Cursor) (*notionapi.GetChildrenResponse, error) {
	res, err := c.Block.GetChildren(ctx, notionapi.BlockID(id), &notionapi.Pagination{
		StartCursor: cursor,
		PageSize:    100,
	})
	if err != nil {
		return nil, fmt.Errorf("getting block children for %s: %w", id, err)
	}
	return res, nil
}

// ListBlocksExpanded fetches blocks and expands those that need additional data (like Tables).
func (c *Client) ListBlocksExpanded(ctx context.Context, id string) ([]notionapi.Block, error) {
	var allBlocks []notionapi.Block
	cursor := notionapi.Cursor("")

	for {
		res, err := c.ListBlocks(ctx, id, cursor)
		if err != nil {
			return nil, err
		}
		allBlocks = append(allBlocks, res.Results...)
		if !res.HasMore {
			break
		}
		cursor = notionapi.Cursor(res.NextCursor)
	}

	var expanded []notionapi.Block
	for _, block := range allBlocks {
		expanded = append(expanded, block)

		if block.GetType() == notionapi.BlockTypeTableBlock {
			// Fetch all rows for this table
			rowCursor := notionapi.Cursor("")
			for {
				rowsRes, err := c.ListBlocks(ctx, string(block.GetID()), rowCursor)
				if err != nil {
					break
				}
				expanded = append(expanded, rowsRes.Results...)
				if !rowsRes.HasMore {
					break
				}
				rowCursor = notionapi.Cursor(rowsRes.NextCursor)
			}
		}
	}
	return expanded, nil
}
