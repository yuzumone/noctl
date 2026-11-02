package notion

import (
	"context"
	"fmt"
	"net/http"
)

// MarkdownWarning describes content changed or ignored by Notion's parser.
type MarkdownWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PageMarkdown contains the original Notion-flavored Markdown and completeness metadata.
type PageMarkdown struct {
	ID              string            `json:"id"`
	Markdown        string            `json:"markdown"`
	Truncated       bool              `json:"truncated"`
	UnknownBlockIDs []string          `json:"unknown_block_ids"`
	Warnings        []MarkdownWarning `json:"warnings"`
}

// UpdatePageMarkdown checks the current snapshot before replacing page content.
// This detects intervening edits but is not an atomic compare-and-swap.
func (c *Client) UpdatePageMarkdown(ctx context.Context, pageID, original, markdown string) (*PageMarkdown, error) {
	current, err := c.RetrievePageMarkdown(ctx, pageID)
	if err != nil {
		return nil, err
	}
	if current.Truncated || len(current.UnknownBlockIDs) > 0 || current.Markdown != original {
		return nil, fmt.Errorf("page changed or is incomplete; reopen the page before saving")
	}
	return c.ReplacePageMarkdown(ctx, pageID, markdown)
}

// RetrievePageMarkdown retrieves page content using Markdown v2.
func (c *Client) RetrievePageMarkdown(ctx context.Context, pageID string) (*PageMarkdown, error) {
	var result PageMarkdown
	path := fmt.Sprintf("/pages/%s/markdown?markdown_version=v2", formatID(pageID))
	if err := c.doNotionJSON(ctx, http.MethodGet, path, nil, &result, "retrieve page markdown"); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReplacePageMarkdown replaces page content synchronously, preserving child-page deletion protection.
func (c *Client) ReplacePageMarkdown(ctx context.Context, pageID, markdown string) (*PageMarkdown, error) {
	body := struct {
		Type    string `json:"type"`
		Version string `json:"markdown_version"`
		Content struct {
			NewString string `json:"new_str"`
		} `json:"replace_content"`
	}{Type: "replace_content", Version: "v2"}
	body.Content.NewString = markdown
	var result PageMarkdown
	path := fmt.Sprintf("/pages/%s/markdown", formatID(pageID))
	if err := c.doNotionJSON(ctx, http.MethodPatch, path, body, &result, "replace page markdown"); err != nil {
		return nil, err
	}
	return &result, nil
}
