package notion

import (
	"context"
	"fmt"

	"github.com/jomei/notionapi"
)

// BlockToString converts a Notion block to a string representation.
func BlockToString(block notionapi.Block) string {
	switch b := block.(type) {
	case *notionapi.ParagraphBlock:
		return TextItemsToString(b.Paragraph.RichText) + "\n"
	case *notionapi.Heading1Block:
		return "# " + TextItemsToString(b.Heading1.RichText) + "\n"
	case *notionapi.Heading2Block:
		return "## " + TextItemsToString(b.Heading2.RichText) + "\n"
	case *notionapi.Heading3Block:
		return "### " + TextItemsToString(b.Heading3.RichText) + "\n"
	case *notionapi.BulletedListItemBlock:
		return "• " + TextItemsToString(b.BulletedListItem.RichText) + "\n"
	case *notionapi.NumberedListItemBlock:
		return "1. " + TextItemsToString(b.NumberedListItem.RichText) + "\n"
	case *notionapi.ToDoBlock:
		check := "☐"
		if b.ToDo.Checked {
			check = "✅"
		}
		return check + " " + TextItemsToString(b.ToDo.RichText) + "\n"
	case *notionapi.ToggleBlock:
		return "> " + TextItemsToString(b.Toggle.RichText) + "\n"
	case *notionapi.CodeBlock:
		return "```" + b.Code.Language + "\n" + TextItemsToString(b.Code.RichText) + "\n```\n"
	case *notionapi.QuoteBlock:
		return "| " + TextItemsToString(b.Quote.RichText) + "\n"
	case *notionapi.CalloutBlock:
		return "💡 " + TextItemsToString(b.Callout.RichText) + "\n"
	case *notionapi.DividerBlock:
		return "---\n"
	}
	return ""
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
