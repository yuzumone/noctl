package notion

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jomei/notionapi"
)

// BlockToString converts a Notion block to its markdown representation.
func BlockToString(block notionapi.Block, includeID bool) string {
	prefix := ""
	if includeID {
		prefix = fmt.Sprintf("<!-- id: %s -->\n", block.GetID())
	}

	switch b := block.(type) {
	case *notionapi.ParagraphBlock:
		return prefix + RichTextToMarkdown(b.Paragraph.RichText) + "\n\n"
	case *notionapi.Heading1Block:
		return prefix + "# " + RichTextToMarkdown(b.Heading1.RichText) + "\n\n"
	case *notionapi.Heading2Block:
		return prefix + "## " + RichTextToMarkdown(b.Heading2.RichText) + "\n\n"
	case *notionapi.Heading3Block:
		return prefix + "### " + RichTextToMarkdown(b.Heading3.RichText) + "\n\n"
	case *notionapi.BulletedListItemBlock:
		return prefix + "- " + RichTextToMarkdown(b.BulletedListItem.RichText) + "\n"
	case *notionapi.NumberedListItemBlock:
		return prefix + "1. " + RichTextToMarkdown(b.NumberedListItem.RichText) + "\n"
	case *notionapi.ToDoBlock:
		check := "[ ]"
		if b.ToDo.Checked {
			check = "[x]"
		}
		return prefix + "- " + check + " " + RichTextToMarkdown(b.ToDo.RichText) + "\n"
	case *notionapi.ToggleBlock:
		return prefix + "> " + RichTextToMarkdown(b.Toggle.RichText) + "\n\n"
	case *notionapi.CodeBlock:
		return prefix + "```" + b.Code.Language + "\n" + RichTextToMarkdown(b.Code.RichText) + "\n```\n\n"
	case *notionapi.QuoteBlock:
		return prefix + "> " + RichTextToMarkdown(b.Quote.RichText) + "\n\n"
	case *notionapi.CalloutBlock:
		return prefix + "> 💡 " + RichTextToMarkdown(b.Callout.RichText) + "\n\n"
	case *notionapi.DividerBlock:
		return prefix + "---\n\n"
	}
	return ""
}

// RenderTable converts table rows into a Markdown table string.
func RenderTable(tableBlock notionapi.Block, rows []notionapi.Block, includeID bool) string {
	if len(rows) == 0 {
		return ""
	}

	var sb strings.Builder
	if includeID && tableBlock != nil {
		sb.WriteString(fmt.Sprintf("<!-- id: %s -->\n", tableBlock.GetID()))
	}

	hasHeader := false
	if tableBlock != nil {
		if tb, ok := tableBlock.(*notionapi.TableBlock); ok {
			hasHeader = tb.Table.HasRowHeader
		}
	}

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
	return BlocksToMarkdownExtended(blocks, false)
}

func BlocksToMarkdownExtended(blocks []notionapi.Block, includeIDs bool) string {
	var sb strings.Builder
	var tableRows []notionapi.Block
	var currentTableBlock notionapi.Block
	inTable := false

	for _, b := range blocks {
		if b.GetType() == notionapi.BlockTypeTableRowBlock {
			tableRows = append(tableRows, b)
			inTable = true
			continue
		}

		if inTable {
			sb.WriteString(RenderTable(currentTableBlock, tableRows, includeIDs))
			tableRows = nil
			inTable = false
			currentTableBlock = nil
		}

		if b.GetType() == notionapi.BlockTypeTableBlock {
			currentTableBlock = b
			continue
		}

		sb.WriteString(BlockToString(b, includeIDs))
	}

	// Catch any trailing table
	if inTable {
		sb.WriteString(RenderTable(currentTableBlock, tableRows, includeIDs))
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

// ParseContentToBlocks converts a string of markdown into a slice of Notion blocks.
func (c *Client) ParseContentToBlocks(content string) []notionapi.Block {
	var blocks []notionapi.Block
	lines := strings.Split(content, "\n")
	
	var currentLines []string
	var currentType string // "p", "h1", "h2", "h3", "li", "todo", "quote", "code", "div"

	finishBlock := func() {
		if len(currentLines) == 0 && currentType != "div" {
			return
		}
		text := strings.Join(currentLines, "\n")
		var b notionapi.Block

		switch currentType {
		case "h1":
			b = &notionapi.Heading1Block{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeHeading1},
				Heading1:   notionapi.Heading{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: text}}}},
			}
		case "h2":
			b = &notionapi.Heading2Block{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeHeading2},
				Heading2:   notionapi.Heading{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: text}}}},
			}
		case "h3":
			b = &notionapi.Heading3Block{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeHeading3},
				Heading3:   notionapi.Heading{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: text}}}},
			}
		case "li":
			b = &notionapi.BulletedListItemBlock{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeBulletedListItem},
				BulletedListItem: notionapi.ListItem{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: text}}}},
			}
		case "todo":
			checked := strings.HasPrefix(text, "[x] ")
			t := text[4:]
			b = &notionapi.ToDoBlock{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeToDo},
				ToDo:       notionapi.ToDo{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: t}}}, Checked: checked},
			}
		case "quote":
			b = &notionapi.QuoteBlock{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeQuote},
				Quote:      notionapi.Quote{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: text}}}},
			}
		case "code":
			lang := "plain text"
			if len(currentLines) > 0 {
				first := currentLines[0]
				if strings.HasPrefix(first, "```") {
					lang = strings.TrimPrefix(first, "```")
					if lang == "" {
						lang = "plain text"
					}
					currentLines = currentLines[1:]
				}
				if len(currentLines) > 0 && strings.HasSuffix(currentLines[len(currentLines)-1], "```") {
					last := currentLines[len(currentLines)-1]
					currentLines[len(currentLines)-1] = strings.TrimSuffix(last, "```")
				}
			}
			codeText := strings.TrimSpace(strings.Join(currentLines, "\n"))
			b = &notionapi.CodeBlock{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeCode},
				Code: notionapi.Code{
					RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: codeText}}},
					Language: lang,
				},
			}
		case "div":
			b = &notionapi.DividerBlock{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeDivider},
			}
		default:
			b = &notionapi.ParagraphBlock{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeParagraph},
				Paragraph:  notionapi.Paragraph{RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: text}}}},
			}
		}
		if b != nil {
			blocks = append(blocks, b)
		}
		currentLines = nil
		currentType = ""
	}

	inCode := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inCode {
			currentLines = append(currentLines, line)
			if strings.HasPrefix(trimmed, "```") {
				inCode = false
				finishBlock()
			}
			continue
		}

		if trimmed == "" {
			finishBlock()
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			finishBlock()
			inCode = true
			currentType = "code"
			currentLines = append(currentLines, line)
			continue
		}

		if trimmed == "---" {
			finishBlock()
			currentType = "div"
			finishBlock()
			continue
		}

		// Check for new block markers
		newType := ""
		content := line
		if strings.HasPrefix(trimmed, "# ") {
			newType = "h1"
			content = strings.TrimPrefix(trimmed, "# ")
		} else if strings.HasPrefix(trimmed, "## ") {
			newType = "h2"
			content = strings.TrimPrefix(trimmed, "## ")
		} else if strings.HasPrefix(trimmed, "### ") {
			newType = "h3"
			content = strings.TrimPrefix(trimmed, "### ")
		} else if strings.HasPrefix(trimmed, "- [ ] ") || strings.HasPrefix(trimmed, "- [x] ") {
			newType = "todo"
			content = trimmed[2:] // Keep [ ] or [x] for finishBlock to handle
		} else if strings.HasPrefix(trimmed, "- ") {
			newType = "li"
			content = strings.TrimPrefix(trimmed, "- ")
		} else if strings.HasPrefix(trimmed, "> ") {
			newType = "quote"
			content = strings.TrimPrefix(trimmed, "> ")
		}

		if newType != "" {
			finishBlock()
			currentType = newType
			currentLines = append(currentLines, content)
		} else {
			// If we were in a heading or divider, a line without a marker starts a new paragraph
			if currentType == "h1" || currentType == "h2" || currentType == "h3" || currentType == "div" {
				finishBlock()
				currentType = "p"
			}
			
			if currentType == "" {
				currentType = "p"
			}
			currentLines = append(currentLines, line)
		}
	}
	finishBlock()

	return blocks
}

// UpdatePageContent surgically updates blocks of a page based on Markdown with ID comments.
func (c *Client) UpdatePageContent(ctx context.Context, pageID string, markdown string) error {
	// 1. Get original blocks to know what we can update
	origBlocks, err := c.ListBlocksExpanded(ctx, pageID)
	if err != nil {
		return err
	}

	blockMap := make(map[string]notionapi.Block)
	for _, b := range origBlocks {
		blockMap[string(b.GetID())] = b
	}

	// 2. Parse Markdown into chunks (ID + Content)
	type chunk struct {
		id      string
		content string
	}
	var chunks []chunk
	idRegex := regexp.MustCompile(`<!-- id: ([a-f0-9-]+) -->`)
	
	lines := strings.Split(markdown, "\n")
	var currentID string
	var currentContent strings.Builder

	for _, line := range lines {
		matches := idRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			// Save previous chunk
			if currentID != "" || currentContent.Len() > 0 {
				chunks = append(chunks, chunk{id: currentID, content: strings.TrimSpace(currentContent.String())})
			}
			currentID = matches[1]
			currentContent.Reset()
			continue
		}
		currentContent.WriteString(line + "\n")
	}
	// Save last chunk
	if currentID != "" || currentContent.Len() > 0 {
		chunks = append(chunks, chunk{id: currentID, content: strings.TrimSpace(currentContent.String())})
	}

	// 3. Process chunks
	seenIDs := make(map[string]bool)
	var lastID string

	for _, ch := range chunks {
		parsedBlocks := c.ParseContentToBlocks(ch.content)
		
		if ch.id != "" {
			seenIDs[ch.id] = true
			lastID = ch.id
			orig, ok := blockMap[ch.id]
			if !ok {
				continue 
			}

			if len(parsedBlocks) > 0 {
				firstBlock := parsedBlocks[0]
				// Check if content changed
				oldContent := strings.TrimSpace(BlockToString(orig, false))
				if ch.content != oldContent {
					updateReq := blockToUpdateStage(firstBlock, orig)
					if updateReq != nil {
						_, err = c.Block.Update(ctx, notionapi.BlockID(ch.id), updateReq)
						if err != nil {
							fmt.Printf("Error updating block %s: %v\n", ch.id, err)
						}
					}
				}
				
				// Handle extra blocks in this chunk
				for i := 1; i < len(parsedBlocks); i++ {
					req := &notionapi.AppendBlockChildrenRequest{
						Children: []notionapi.Block{parsedBlocks[i]},
						After:    notionapi.BlockID(lastID),
					}
					res, err := c.Block.AppendChildren(ctx, notionapi.BlockID(pageID), req)
					if err == nil && len(res.Results) > 0 {
						lastID = string(res.Results[0].GetID())
					}
				}
			}
		} else {
			// New content: Append all parsed blocks
			for _, b := range parsedBlocks {
				req := &notionapi.AppendBlockChildrenRequest{
					Children: []notionapi.Block{b},
				}
				if lastID != "" {
					req.After = notionapi.BlockID(lastID)
				}
				res, err := c.Block.AppendChildren(ctx, notionapi.BlockID(pageID), req)
				if err == nil && len(res.Results) > 0 {
					lastID = string(res.Results[0].GetID())
				}
			}
		}
	}

	// 4. Delete blocks that were removed from Markdown
	for id, orig := range blockMap {
		if !seenIDs[id] {
			if BlockToString(orig, false) != "" {
				_, _ = c.Block.Delete(ctx, notionapi.BlockID(id))
			}
		}
	}

	return nil
}

func blockToUpdateStage(new notionapi.Block, orig notionapi.Block) *notionapi.BlockUpdateRequest {
	req := &notionapi.BlockUpdateRequest{}
	
	switch b := new.(type) {
	case *notionapi.ParagraphBlock:
		req.Paragraph = &b.Paragraph
	case *notionapi.Heading1Block:
		req.Heading1 = &b.Heading1
	case *notionapi.Heading2Block:
		req.Heading2 = &b.Heading2
	case *notionapi.Heading3Block:
		req.Heading3 = &b.Heading3
	case *notionapi.BulletedListItemBlock:
		req.BulletedListItem = &b.BulletedListItem
	case *notionapi.ToDoBlock:
		req.ToDo = &b.ToDo
	case *notionapi.QuoteBlock:
		// If original was Callout, try to keep it as callout
		if orig.GetType() == notionapi.BlockTypeCallout {
			req.Callout = &notionapi.Callout{
				RichText: b.Quote.RichText,
			}
		} else {
			req.Quote = &b.Quote
		}
	case *notionapi.CodeBlock:
		req.Code = &b.Code
	}
	return req
}

